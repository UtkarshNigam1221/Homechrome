package metrics

import "context"

type bufferKey struct{}

type buffer struct {
	events []Event
}

func (b *buffer) append(ev Event) {
	b.events = append(b.events, ev)
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
