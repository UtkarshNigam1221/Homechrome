package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSQS struct {
	calls     int
	failUntil int // fail on attempts <= failUntil
	err       error
}

func (f *fakeSQS) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.calls++
	if f.calls <= f.failUntil {
		return nil, f.err
	}
	return &sqs.SendMessageOutput{}, nil
}

// zeroDelays replaces retryDelays with zero-duration slice for fast tests.
func zeroDelays() (restore func()) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	return func() { retryDelays = orig }
}

func TestSQSPublisher_successFirstTry(t *testing.T) {
	defer zeroDelays()()

	fake := &fakeSQS{}
	p := NewSQSPublisher(fake, "https://sqs.test/queue")
	err := p.Publish(context.Background(), []Event{{Metric: "x", Value: 1}})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
}

func TestSQSPublisher_successAfterTwoRetries(t *testing.T) {
	defer zeroDelays()()

	fake := &fakeSQS{failUntil: 2, err: errors.New("throttle")}
	p := NewSQSPublisher(fake, "https://sqs.test/queue")
	err := p.Publish(context.Background(), []Event{{Metric: "x"}})
	require.NoError(t, err)
	assert.Equal(t, 3, fake.calls)
}

func TestSQSPublisher_failAfterThreeAttempts(t *testing.T) {
	defer zeroDelays()()

	fake := &fakeSQS{failUntil: 99, err: errors.New("permanent")}
	p := NewSQSPublisher(fake, "https://sqs.test/queue")
	err := p.Publish(context.Background(), []Event{{Metric: "x"}})
	require.Error(t, err)
	assert.Equal(t, 3, fake.calls)
}

func TestSQSPublisher_emptyEventsNoOp(t *testing.T) {
	fake := &fakeSQS{}
	p := NewSQSPublisher(fake, "https://sqs.test/queue")
	err := p.Publish(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, fake.calls)
}

func TestSQSPublisher_chunksOversizedBatch(t *testing.T) {
	defer zeroDelays()()

	// 20 events of ~30 KB each ≈ 600 KB total — must split into several
	// messages, each under maxSQSBodyBytes (240 KB).
	big := strings.Repeat("x", 30*1024)
	events := make([]Event, 20)
	for i := range events {
		events[i] = Event{Metric: "m", Labels: L{"reason": big}, Value: 1}
	}

	fake := &fakeSQS{}
	p := NewSQSPublisher(fake, "https://sqs.test/queue")
	require.NoError(t, p.Publish(context.Background(), events))
	assert.Greater(t, fake.calls, 1, "oversized batch should be split into multiple messages")
}

func TestChunkEvents_keepsChunksUnderLimit(t *testing.T) {
	big := strings.Repeat("y", 50*1024)
	events := make([]Event, 12)
	for i := range events {
		events[i] = Event{Metric: "m", Labels: L{"reason": big}}
	}
	chunks, err := chunkEvents(events)
	require.NoError(t, err)
	require.Greater(t, len(chunks), 1)
	for _, c := range chunks {
		body, err := json.Marshal(c)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(body), maxSQSBodyBytes)
	}
}
