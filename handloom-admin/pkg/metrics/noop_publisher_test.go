package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopPublisher_alwaysReturnsNil(t *testing.T) {
	p := NoopPublisher{}
	err := p.Publish(context.Background(), []Event{{Metric: "x"}})
	assert.NoError(t, err)
}

func TestNoopPublisher_implementsPublisher(t *testing.T) {
	var _ Publisher = NoopPublisher{}
}
