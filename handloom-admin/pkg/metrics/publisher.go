package metrics

import "context"

// Publisher flushes a batch of events to a backing store.
type Publisher interface {
	Publish(ctx context.Context, events []Event) error
}
