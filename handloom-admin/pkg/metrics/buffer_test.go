package metrics

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithBuffer_storesAndRetrievesEvents(t *testing.T) {
	ctx := WithBuffer(context.Background())
	buf := bufferFromContext(ctx)
	assert.NotNil(t, buf)
	assert.Len(t, buf.events, 0)

	buf.append(Event{Metric: "test", Labels: L{"a": "b"}, Value: 1, RetentionClass: RetentionBusiness})

	buf2 := bufferFromContext(ctx)
	assert.Len(t, buf2.events, 1)
	assert.Equal(t, "test", buf2.events[0].Metric)
}

func TestBufferFromContext_returnsNilWhenAbsent(t *testing.T) {
	buf := bufferFromContext(context.Background())
	assert.Nil(t, buf)
}

// TestBuffer_concurrentAppend exercises the mutex: emits from many goroutines
// while a reader snapshots. Run with -race to catch regressions.
func TestBuffer_concurrentAppend(t *testing.T) {
	ctx := WithBuffer(context.Background())
	buf := bufferFromContext(ctx)

	const goroutines, perG = 16, 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				buf.append(Event{Metric: "concurrent", Value: 1})
				_ = buf.snapshot() // concurrent read under lock
			}
		}()
	}
	wg.Wait()

	assert.Len(t, buf.snapshot(), goroutines*perG)
}
