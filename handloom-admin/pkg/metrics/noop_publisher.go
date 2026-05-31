package metrics

import "context"

// NoopPublisher discards all events silently.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ []Event) error { return nil }
