package metrics

import (
	"context"
	"errors"
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
