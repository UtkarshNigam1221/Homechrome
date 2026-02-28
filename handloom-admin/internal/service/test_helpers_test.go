package service

import (
	"context"
	"fmt"

	"github.com/handloom/admin/internal/event"
)

// spyPublisher records published events for test assertions.
type spyPublisher struct {
	events []event.Event
	err    error // if set, Publish returns this error
}

func newSpyPublisher() *spyPublisher {
	return &spyPublisher{}
}

func newFailingPublisher(err error) *spyPublisher {
	return &spyPublisher{err: err}
}

func (s *spyPublisher) Publish(_ context.Context, e event.Event) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, e)
	return nil
}

func (s *spyPublisher) hasEvent(t event.EventType) bool {
	for _, e := range s.events {
		if e.Type == t {
			return true
		}
	}
	return false
}

func (s *spyPublisher) eventCount() int {
	return len(s.events)
}

// spyCache records cache invalidation calls.
type spyCache struct {
	prefixes []string
}

func newSpyCache() *spyCache {
	return &spyCache{}
}

func (s *spyCache) DeletePrefix(prefix string) {
	s.prefixes = append(s.prefixes, prefix)
}

func (s *spyCache) calledWith(prefix string) bool {
	for _, p := range s.prefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

func (s *spyCache) callCount() int {
	return len(s.prefixes)
}

// ptr is a generic helper for creating pointers to literals in tests.
func ptr[T any](v T) *T {
	return &v
}

// Compile-time check: spyPublisher implements EventPublisher
var _ event.EventPublisher = (*spyPublisher)(nil)

// Compile-time check: spyCache implements CacheInvalidator
var _ CacheInvalidator = (*spyCache)(nil)

// Suppress unused import warning
var _ = fmt.Sprintf
