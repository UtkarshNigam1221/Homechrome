package metrics

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

// IdempotencyCache is an LRU set used by the consumer to dedup events
// across SQS at-least-once redelivery. Lost on Lambda cold start
// (~0.1% drift accepted per spec).
type IdempotencyCache struct {
	cache *lru.Cache[string, struct{}]
}

// defaultIdempotencyCacheSize is used when a non-positive size is requested,
// so the underlying LRU constructor never errors and leaves a nil cache.
const defaultIdempotencyCacheSize = 1024

// NewIdempotencyCache creates an IdempotencyCache with the given capacity.
// A non-positive size falls back to defaultIdempotencyCacheSize; lru.New only
// errors on a non-positive size, so the fallback makes the error impossible.
func NewIdempotencyCache(size int) *IdempotencyCache {
	if size <= 0 {
		size = defaultIdempotencyCacheSize
	}
	c, err := lru.New[string, struct{}](size)
	if err != nil {
		// Unreachable given the size guard above, but never return a nil cache.
		c, _ = lru.New[string, struct{}](defaultIdempotencyCacheSize)
	}
	return &IdempotencyCache{cache: c}
}

// Has reports whether key was recently seen.
func (c *IdempotencyCache) Has(key string) bool {
	_, ok := c.cache.Get(key)
	return ok
}

// Add marks key as seen.
func (c *IdempotencyCache) Add(key string) {
	c.cache.Add(key, struct{}{})
}
