package metrics

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdempotencyCache_AddAndCheck(t *testing.T) {
	c := NewIdempotencyCache(100)

	assert.False(t, c.Has("key1"))
	c.Add("key1")
	assert.True(t, c.Has("key1"))

	assert.False(t, c.Has("key2"))
}

func TestIdempotencyCache_EvictsLRU(t *testing.T) {
	const size = 5
	c := NewIdempotencyCache(size)

	// Fill to capacity.
	for i := range size {
		c.Add(fmt.Sprintf("k%d", i))
	}

	// All should be present.
	for i := range size {
		assert.True(t, c.Has(fmt.Sprintf("k%d", i)), "k%d should be present", i)
	}

	// Add one more — k0 (LRU) should be evicted.
	c.Add("new-key")
	assert.True(t, c.Has("new-key"))

	// Exactly one of the original keys was evicted.
	evicted := 0
	for i := range size {
		if !c.Has(fmt.Sprintf("k%d", i)) {
			evicted++
		}
	}
	assert.Equal(t, 1, evicted, "exactly one entry should have been evicted")
}
