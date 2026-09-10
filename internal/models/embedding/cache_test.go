package embedding

import (
	"context"
	"testing"
	"time"
)

// countingEmbedder 记录真实调用次数并返回可预测的向量，用于断言缓存是否拦截了重复调用。
type countingEmbedder struct {
	modelName   string
	dims        int
	embedCalls  int
	batchCalls  int
	batchCounts []int // 每次 BatchEmbed 收到的文本数量
}

func newCountingEmbedder(modelName string, dims int) *countingEmbedder {
	return &countingEmbedder{modelName: modelName, dims: dims}
}

func (c *countingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	c.embedCalls++
	return []float32{float32(len(text))}, nil
}

func (c *countingEmbedder) BatchEmbed(_ context.Context, texts []string) ([][]float32, error) {
	c.batchCalls++
	c.batchCounts = append(c.batchCounts, len(texts))
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = []float32{float32(len(t))}
	}
	return out, nil
}

func (c *countingEmbedder) BatchEmbedWithPool(_ context.Context, model Embedder, texts []string) ([][]float32, error) {
	return model.BatchEmbed(context.Background(), texts)
}

func (c *countingEmbedder) GetModelName() string { return c.modelName }
func (c *countingEmbedder) GetDimensions() int   { return c.dims }
func (c *countingEmbedder) GetModelID() string   { return c.modelName }

func TestEmbeddingCacheKey(t *testing.T) {
	base := embeddingCacheKey("text-embedding-3-small", 1536, "hello")
	if base == "" || len(base) != 16 {
		t.Fatalf("unexpected key %q", base)
	}

	same := embeddingCacheKey("text-embedding-3-small", 1536, "hello")
	if same != base {
		t.Fatalf("identical inputs must produce identical keys: %q vs %q", base, same)
	}

	diffText := embeddingCacheKey("text-embedding-3-small", 1536, "world")
	diffModel := embeddingCacheKey("text-embedding-3-large", 1536, "hello")
	diffDims := embeddingCacheKey("text-embedding-3-small", 1024, "hello")
	if diffText == base || diffModel == base || diffDims == base {
		t.Fatal("different model/dimensions/text must produce different keys")
	}
}

func TestCacheEmbedderEmbedSkipsRepeatedCall(t *testing.T) {
	inner := newCountingEmbedder("m", 4)
	c := wrapEmbeddingCache(inner, newMemoryEmbeddingCache(0, 0))
	ctx := context.Background()

	if _, err := c.Embed(ctx, "hello"); err != nil {
		t.Fatalf("first embed: %v", err)
	}
	if _, err := c.Embed(ctx, "hello"); err != nil {
		t.Fatalf("second embed: %v", err)
	}
	if _, err := c.Embed(ctx, "world"); err != nil {
		t.Fatalf("third embed: %v", err)
	}
	if inner.embedCalls != 2 {
		t.Fatalf("expected 2 real calls (hello + world), got %d", inner.embedCalls)
	}
}

func TestCacheEmbedderBatchEmbedOnlyMissesReachInner(t *testing.T) {
	inner := newCountingEmbedder("m", 4)
	c := wrapEmbeddingCache(inner, newMemoryEmbeddingCache(0, 0))
	ctx := context.Background()

	if _, err := c.BatchEmbed(ctx, []string{"a", "b", "c"}); err != nil {
		t.Fatalf("first batch: %v", err)
	}
	if _, err := c.BatchEmbed(ctx, []string{"a", "b", "d"}); err != nil {
		t.Fatalf("second batch: %v", err)
	}
	if inner.batchCalls != 2 {
		t.Fatalf("expected 2 real batch calls, got %d", inner.batchCalls)
	}
	// 第二次只对未命中的 "d" 真正调用。
	if got := inner.batchCounts[1]; got != 1 {
		t.Fatalf("second real call should process 1 miss, got %d", got)
	}
}

func TestCacheEmbedderBatchEmbedWithPoolSkipsRepeatedCall(t *testing.T) {
	inner := newCountingEmbedder("m", 4)
	c := wrapEmbeddingCache(inner, newMemoryEmbeddingCache(0, 0))
	ctx := context.Background()

	first, err := c.BatchEmbedWithPool(ctx, c, []string{"x", "y"})
	if err != nil {
		t.Fatalf("first pool batch: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("expected 2 results, got %d", len(first))
	}
	before := inner.batchCalls

	if _, err := c.BatchEmbedWithPool(ctx, c, []string{"x", "y"}); err != nil {
		t.Fatalf("second pool batch: %v", err)
	}
	if inner.batchCalls != before {
		t.Fatalf("repeated pool batch should be fully cached, real calls went from %d to %d", before, inner.batchCalls)
	}
}

func TestMemoryEmbeddingCacheTTLExpiry(t *testing.T) {
	cache := newMemoryEmbeddingCache(20*time.Millisecond, 100)
	cache.Set("k", []float32{1, 2, 3})

	if _, ok := cache.Get("k"); !ok {
		t.Fatal("expected hit right after set")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := cache.Get("k"); ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestMemoryEmbeddingCacheCloneIsolation(t *testing.T) {
	cache := newMemoryEmbeddingCache(0, 0)
	cache.Set("k", []float32{1, 2, 3})

	v, ok := cache.Get("k")
	if !ok {
		t.Fatal("expected hit")
	}
	v[0] = 99 // 修改返回的副本不应污染缓存

	v2, _ := cache.Get("k")
	if v2[0] != 1 {
		t.Fatalf("cache was polluted by caller mutation: %v", v2)
	}
}
