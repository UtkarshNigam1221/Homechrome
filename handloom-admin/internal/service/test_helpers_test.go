package service

import (
	"context"

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

// Compile-time check: spyPublisher implements EventPublisher
var _ event.EventPublisher = (*spyPublisher)(nil)
