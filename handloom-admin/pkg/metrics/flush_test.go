package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	events []Event
	err    error
}

func (r *recordingPublisher) Publish(_ context.Context, events []Event) error {
	r.events = append(r.events, events...)
	return r.err
}

func TestFlush_sendsEventsToPublisher(t *testing.T) {
	rec := &recordingPublisher{}
	SetDefault(rec)
	defer SetDefault(NoopPublisher{})

	ctx := WithBuffer(context.Background())
	Record(ctx, "order.placed", L{"state": "MH"})

	err := Flush(ctx)
	require.NoError(t, err)
	require.Len(t, rec.events, 1)
	assert.Equal(t, "order.placed", rec.events[0].Metric)
}

func TestFlush_emptyBuffer_noOp(t *testing.T) {
	rec := &recordingPublisher{}
	SetDefault(rec)
	defer SetDefault(NoopPublisher{})

	ctx := WithBuffer(context.Background())
	err := Flush(ctx)
	require.NoError(t, err)
	assert.Len(t, rec.events, 0)
}

func TestFlush_noBufferInCtx_noOp(t *testing.T) {
	rec := &recordingPublisher{}
	SetDefault(rec)
	defer SetDefault(NoopPublisher{})

	err := Flush(context.Background())
	require.NoError(t, err)
	assert.Len(t, rec.events, 0)
}

func TestFlush_publisherError_returnsError(t *testing.T) {
	rec := &recordingPublisher{err: errors.New("downstream")}
	SetDefault(rec)
	defer SetDefault(NoopPublisher{})

	ctx := WithBuffer(context.Background())
	Record(ctx, "x", L{})

	err := Flush(ctx)
	assert.ErrorContains(t, err, "downstream")
}
