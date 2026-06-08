package metrics

import (
	"context"
	"sync/atomic"
)

// publisherHolder is a fixed-type wrapper so atomic.Value always stores the same type.
type publisherHolder struct{ p Publisher }

var defaultPublisher atomic.Value

func init() {
	defaultPublisher.Store(publisherHolder{p: NoopPublisher{}})
}

// SetDefault sets the package-level default publisher.
func SetDefault(p Publisher) {
	defaultPublisher.Store(publisherHolder{p: p})
}

// Flush sends all buffered events from ctx to the default publisher.
// It is a no-op when the buffer is empty or absent.
func Flush(ctx context.Context) error {
	buf := bufferFromContext(ctx)
	if buf == nil {
		return nil
	}
	events := buf.snapshot()
	if len(events) == 0 {
		return nil
	}
	p := defaultPublisher.Load().(publisherHolder).p
	return p.Publish(ctx, events)
}
