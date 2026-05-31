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

// NewIdempotencyCache creates an IdempotencyCache with the given capacity.
func NewIdempotencyCache(size int) *IdempotencyCache {
	c, _ := lru.New[string, struct{}](size)
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
