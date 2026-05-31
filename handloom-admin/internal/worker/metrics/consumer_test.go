package metrics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/repository/postgres"
	pkgmetrics "github.com/handloom/admin/pkg/metrics"
)

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
