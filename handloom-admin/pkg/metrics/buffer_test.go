package metrics

import (
	"context"
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
