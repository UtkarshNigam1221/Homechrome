package metrics

import (
	"context"
	"sync"
)

type bufferKey struct{}

// buffer accumulates events for the lifetime of a request. Emits may happen
// from goroutines spawned by a handler, so all access is mutex-guarded;
// Flush snapshots under the same lock.
type buffer struct {
	mu     sync.Mutex
	events []Event
}

func (b *buffer) append(ev Event) {
	b.mu.Lock()
	b.events = append(b.events, ev)
	b.mu.Unlock()
}

// snapshot returns a copy of the buffered events under lock.
func (b *buffer) snapshot() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) == 0 {
		return nil
	}
	out := make([]Event, len(b.events))
	copy(out, b.events)
	return out
}

// WithBuffer returns a context carrying a fresh event buffer.
func WithBuffer(ctx context.Context) context.Context {
	return context.WithValue(ctx, bufferKey{}, &buffer{events: make([]Event, 0, 8)})
}

func bufferFromContext(ctx context.Context) *buffer {
	if v := ctx.Value(bufferKey{}); v != nil {
		return v.(*buffer)
	}
	return nil
}
