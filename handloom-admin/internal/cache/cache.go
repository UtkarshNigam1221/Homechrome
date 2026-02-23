// Package cache provides an in-process TTL cache for catalog data.
package cache

import (
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache is a thin wrapper around go-cache providing typed operations
// and prefix-based invalidation.
type Cache struct {
	store *gocache.Cache
}

// New creates a cache with the given default TTL and cleanup interval.
func New(defaultTTL, cleanupInterval time.Duration) *Cache {
	return &Cache{
		store: gocache.New(defaultTTL, cleanupInterval),
	}
}

// Get retrieves a value by key. Returns the value and whether it was found.
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

// Set stores a value with a specific TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.store.Set(key, value, ttl)
}

// Delete removes a single key.
func (c *Cache) Delete(key string) {
	c.store.Delete(key)
}

// DeletePrefix removes all keys that start with the given prefix.
func (c *Cache) DeletePrefix(prefix string) {
	for k := range c.store.Items() {
		if strings.HasPrefix(k, prefix) {
			c.store.Delete(k)
		}
	}
}
