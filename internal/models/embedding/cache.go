package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// EmbeddingCache 是 embedding 结果缓存的存储抽象。
// 当前提供进程内实现（memoryEmbeddingCache），未来可替换为 Redis 等
// 跨实例实现，以支持多副本之间或进程重启后的复用。
type EmbeddingCache interface {
	// Get 返回 key 对应的向量；未命中（或已过期）时 ok 为 false。
	Get(key string) ([]float32, bool)
	// Set 写入 key 对应的向量。实现必须持有向量的独立副本，调用方
	// 之后修改传入的切片不应影响缓存内容。
	Set(key string, vec []float32)
}

// cacheEmbedder 是 embedding 缓存的装饰器：相同「模型 + 维度 + 文本」的
// 向量结果直接复用，避免对模型 API 的重复计费调用。
//
// 它放在装饰器链的最外层，因此命中时会在进入 langfuse 观测、debug 日志和
// 并发限流（concurrencyEmbedder）之前就短路返回——一次命中既不计费、也不
// 产生下游观测、更不占用后台并发额度。
type cacheEmbedder struct {
	inner Embedder
	cache EmbeddingCache
}

func (c *cacheEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	key := embeddingCacheKey(c.inner.GetModelName(), c.inner.GetDimensions(), text)
	if vec, ok := c.cache.Get(key); ok {
		return vec, nil
	}
	vec, err := c.inner.Embed(ctx, text)
	if err == nil {
		c.cache.Set(key, vec)
	}
	return vec, err
}

func (c *cacheEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	modelName := c.inner.GetModelName()
	dims := c.inner.GetDimensions()
	results := make([][]float32, len(texts))
	var missIdx []int
	var missTexts []string
	for i, text := range texts {
		if vec, ok := c.cache.Get(embeddingCacheKey(modelName, dims, text)); ok {
			results[i] = vec
		} else {
			missIdx = append(missIdx, i)
			missTexts = append(missTexts, text)
		}
	}
	if len(missTexts) == 0 {
		return results, nil
	}
	vecs, err := c.inner.BatchEmbed(ctx, missTexts)
	if err != nil {
		return nil, err
	}
	for j, idx := range missIdx {
		results[idx] = vecs[j]
		c.cache.Set(embeddingCacheKey(modelName, dims, missTexts[j]), vecs[j])
	}
	return results, nil
}

// BatchEmbedWithPool 同样先查缓存，只对 miss 的文本走下游的池化并发路径。
// 缓存命中在进入池化与并发限流之前就短路，因此重建索引时重复文档的 embedding
// 不再产生真实的 provider 调用。
func (c *cacheEmbedder) BatchEmbedWithPool(ctx context.Context, model Embedder, texts []string) ([][]float32, error) {
	modelName := c.inner.GetModelName()
	dims := c.inner.GetDimensions()
	results := make([][]float32, len(texts))
	var missIdx []int
	var missTexts []string
	for i, text := range texts {
		if vec, ok := c.cache.Get(embeddingCacheKey(modelName, dims, text)); ok {
			results[i] = vec
		} else {
			missIdx = append(missIdx, i)
			missTexts = append(missTexts, text)
		}
	}
	if len(missTexts) == 0 {
		return results, nil
	}
	vecs, err := c.inner.BatchEmbedWithPool(ctx, c, missTexts)
	if err != nil {
		return nil, err
	}
	for j, idx := range missIdx {
		results[idx] = vecs[j]
		c.cache.Set(embeddingCacheKey(modelName, dims, missTexts[j]), vecs[j])
	}
	return results, nil
}

func (c *cacheEmbedder) GetModelName() string { return c.inner.GetModelName() }
func (c *cacheEmbedder) GetDimensions() int   { return c.inner.GetDimensions() }
func (c *cacheEmbedder) GetModelID() string   { return c.inner.GetModelID() }

// wrapEmbeddingCache 把缓存装饰器包到 embedder 上；embedder 或 cache 为 nil 时原样返回。
func wrapEmbeddingCache(e Embedder, cache EmbeddingCache) Embedder {
	if e == nil || cache == nil {
		return e
	}
	return &cacheEmbedder{inner: e, cache: cache}
}

// embeddingCacheKey 生成缓存键：模型名 + 维度 + 文本三者的 SHA256 指纹（取前 16 位十六进制）。
// 维度参与指纹，保证「同一文本但维度配置不同」不会互相串用向量。
func embeddingCacheKey(modelName string, dimensions int, text string) string {
	h := sha256.New()
	h.Write([]byte(modelName))
	h.Write([]byte{0})
	h.Write([]byte(strconv.Itoa(dimensions)))
	h.Write([]byte{0})
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
