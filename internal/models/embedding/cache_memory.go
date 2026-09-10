package embedding

import (
	"sync"
	"time"
)

// 默认缓存参数：embedding 向量由模型决定，模型不变则结果不变，因此 TTL 可以
// 设得较长；条目上限用于兜底，防止进程内存被无限增长的向量撑爆。
const (
	defaultEmbeddingCacheTTL        = 24 * time.Hour
	defaultEmbeddingCacheMaxEntries = 10000
)

// memoryEmbeddingCache 是进程内缓存实现：一个带读写锁的 map，配合惰性过期与
// 容量上限。简单、无外部依赖、任何部署形态（含 Lite 模式无 Redis）下都可用。
type memoryEmbeddingCache struct {
	mu      sync.RWMutex
	entries map[string]memoryCacheEntry
	ttl     time.Duration
	max     int
}

type memoryCacheEntry struct {
	vec       []float32
	expiresAt time.Time
}

func newMemoryEmbeddingCache(ttl time.Duration, maxEntries int) *memoryEmbeddingCache {
	if ttl <= 0 {
		ttl = defaultEmbeddingCacheTTL
	}
	if maxEntries <= 0 {
		maxEntries = defaultEmbeddingCacheMaxEntries
	}
	return &memoryEmbeddingCache{
		entries: make(map[string]memoryCacheEntry),
		ttl:     ttl,
		max:     maxEntries,
	}
}

func (m *memoryEmbeddingCache) Get(key string) ([]float32, bool) {
	m.mu.RLock()
	e, ok := m.entries[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		// 惰性删除过期项：加写锁后二次确认，避免并发下重复删除。
		m.mu.Lock()
		if cur, still := m.entries[key]; still && time.Now().After(cur.expiresAt) {
			delete(m.entries, key)
		}
		m.mu.Unlock()
		return nil, false
	}
	return cloneVec(e.vec), true
}

func (m *memoryEmbeddingCache) Set(key string, vec []float32) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	// 达到上限时先清理过期项；若仍超上限则整体重置，回到无缓存状态。
	// 最坏情况只是丢失缓存、重新计算，正确性不受影响。
	if len(m.entries) >= m.max {
		for k, e := range m.entries {
			if now.After(e.expiresAt) {
				delete(m.entries, k)
			}
		}
		if len(m.entries) >= m.max {
			m.entries = make(map[string]memoryCacheEntry)
		}
	}
	m.entries[key] = memoryCacheEntry{vec: cloneVec(vec), expiresAt: now.Add(m.ttl)}
}

// cloneVec 返回向量的独立副本，隔离调用方对切片的后续修改。
func cloneVec(vec []float32) []float32 {
	if vec == nil {
		return nil
	}
	out := make([]float32, len(vec))
	copy(out, vec)
	return out
}

// 全局共享缓存：所有 embedder 共用一个缓存实例。缓存键已包含模型名与维度，
// 不同模型/维度之间天然隔离，不会互相串用向量。
var (
	globalEmbeddingCacheOnce sync.Once
	globalEmbeddingCache     EmbeddingCache
)

func globalEmbeddingCacheInstance() EmbeddingCache {
	globalEmbeddingCacheOnce.Do(func() {
		globalEmbeddingCache = newMemoryEmbeddingCache(
			defaultEmbeddingCacheTTL,
			defaultEmbeddingCacheMaxEntries,
		)
	})
	return globalEmbeddingCache
}
