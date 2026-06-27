package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/repository/postgres"
	pkgmetrics "github.com/handloom/admin/pkg/metrics"
)

// flakyRepo fails the first failFirstN UpsertCounters calls, then succeeds.
type flakyRepo struct {
	calls      [][]postgres.CounterRow
	failFirstN int
}

func (s *flakyRepo) UpsertCounters(_ context.Context, rows []postgres.CounterRow) error {
	s.calls = append(s.calls, rows)
	if len(s.calls) <= s.failFirstN {
		return errors.New("transient pg error")
	}
	return nil
}

// stubMetricsRepo captures calls to UpsertCounters.
type stubMetricsRepo struct {
	calls [][]postgres.CounterRow
	err   error
}

func (s *stubMetricsRepo) UpsertCounters(_ context.Context, rows []postgres.CounterRow) error {
	s.calls = append(s.calls, rows)
	return s.err
}

// makeEvent returns a JSON-array body matching the producer's wire format
// (pkg/metrics SQSPublisher marshals []Event, not a single Event).
func makeEvent(metric string, labels pkgmetrics.L, value int64, ikey string, at time.Time) string {
	evt := pkgmetrics.Event{
		Metric:         metric,
		Labels:         labels,
		Value:          value,
		SumValue:       value * 10,
		RetentionClass: pkgmetrics.RetentionService,
		EmittedAt:      at,
		IdempotencyKey: ikey,
	}
	b, _ := json.Marshal([]pkgmetrics.Event{evt})
	return string(b)
}

func TestConsumer_AggregatesSameKey(t *testing.T) {
	stub := &stubMetricsRepo{}
	h := &Handler{repo: stub, cache: NewIdempotencyCache(1000)}

	now := time.Now().UTC()

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "1", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 1, "ikey-1", now)},
			{MessageId: "2", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 2, "ikey-2", now)},
			{MessageId: "3", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 3, "ikey-3", now)},
		},
	}

	resp, err := h.HandleSQSEvent(context.Background(), sqsEvent)
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 1, "three same-key events must aggregate to one row")
	assert.Equal(t, int64(6), stub.calls[0][0].Count)
	assert.Equal(t, int64(60), stub.calls[0][0].SumValue)
}

func TestConsumer_DedupsByIdempotencyKey(t *testing.T) {
	stub := &stubMetricsRepo{}
	h := &Handler{repo: stub, cache: NewIdempotencyCache(1000)}

	now := time.Now().UTC()

	// Send the same idempotency key twice — second must be skipped.
	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "1", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 5, "dup-key", now)},
			{MessageId: "2", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 5, "dup-key", now)},
		},
	}

	resp, err := h.HandleSQSEvent(context.Background(), sqsEvent)
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	require.Len(t, stub.calls, 1)
	// Only the first observation should be counted.
	assert.Equal(t, int64(5), stub.calls[0][0].Count)
}

// Regression: a failed upsert must NOT cache the idempotency keys, so an SQS
// redelivery to the same warm container reprocesses the batch instead of
// dedup-skipping it and silently losing the metrics.
func TestConsumer_FailedUpsertLeavesKeysUncached_RetryReprocesses(t *testing.T) {
	repo := &flakyRepo{failFirstN: 1}
	cache := NewIdempotencyCache(1000)
	h := &Handler{repo: repo, cache: cache}

	now := time.Now().UTC()
	batch := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "1", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 4, "ikey-A", now)},
		},
	}

	// First delivery: upsert fails → whole batch reported for retry, key NOT cached.
	resp, err := h.HandleSQSEvent(context.Background(), batch)
	require.NoError(t, err)
	require.Len(t, resp.BatchItemFailures, 1, "failed upsert must report the message for retry")
	require.False(t, cache.Has("ikey-A"), "key must not be cached before a durable write")

	// SQS redelivers to the same warm container (same cache); upsert now succeeds.
	resp, err = h.HandleSQSEvent(context.Background(), batch)
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	require.Len(t, repo.calls, 2, "retry must reprocess, not dedup-skip")
	require.Len(t, repo.calls[1], 1)
	assert.Equal(t, int64(4), repo.calls[1][0].Count, "metric must be written on retry, not lost")
	assert.True(t, cache.Has("ikey-A"), "key cached only after the successful write")
}

func TestConsumer_MalformedJSON_ReportsItemFailure(t *testing.T) {
	stub := &stubMetricsRepo{}
	h := &Handler{repo: stub, cache: NewIdempotencyCache(1000)}

	now := time.Now().UTC()

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "good", Body: makeEvent("req.count", pkgmetrics.L{"svc": "auth"}, 1, "k1", now)},
			{MessageId: "bad", Body: `not-valid-json`},
		},
	}

	resp, err := h.HandleSQSEvent(context.Background(), sqsEvent)
	require.NoError(t, err)
	require.Len(t, resp.BatchItemFailures, 1)
	assert.Equal(t, "bad", resp.BatchItemFailures[0].ItemIdentifier)
	// Good message should still be upserted.
	require.Len(t, stub.calls, 1)
}
