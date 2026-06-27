package metrics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

// Record records a count-1 event for the given metric name and labels.
func Record(ctx context.Context, name string, labels L) {
	appendEvent(ctx, Event{
		Metric:         name,
		Labels:         cloneLabels(labels),
		Value:          1,
		RetentionClass: retentionClassFor(name),
		EmittedAt:      time.Now().UTC(),
		IdempotencyKey: newKey(),
	})
}

// RecordSum records a count-1 event with an additional sum value.
func RecordSum(ctx context.Context, name string, value int64, labels L) {
	appendEvent(ctx, Event{
		Metric:         name,
		Labels:         cloneLabels(labels),
		Value:          1,
		SumValue:       value,
		RetentionClass: retentionClassFor(name),
		EmittedAt:      time.Now().UTC(),
		IdempotencyKey: newKey(),
	})
}

// RecordDuration resolves a histogram bucket for d and records a count-1 event.
func RecordDuration(ctx context.Context, name string, d time.Duration, labels L) {
	lb := cloneLabels(labels)
	boundaries, bLabels := boundariesForMetric(name)
	lb["bucket"] = BucketForDuration(d, boundaries, bLabels)
	appendEvent(ctx, Event{
		Metric:         name,
		Labels:         lb,
		Value:          1,
		RetentionClass: retentionClassFor(name),
		EmittedAt:      time.Now().UTC(),
		IdempotencyKey: newKey(),
	})
}

func appendEvent(ctx context.Context, ev Event) {
	buf := bufferFromContext(ctx)
	if buf == nil {
		return
	}
	buf.append(ev)
}

// retentionClassFor returns RetentionService for infrastructure metrics, else RetentionBusiness.
func retentionClassFor(name string) RetentionClass {
	for _, prefix := range []string{"http_", "db_", "aws_sdk_", "lambda_", "otp_", "rum_"} {
		if strings.HasPrefix(name, prefix) {
			return RetentionService
		}
	}
	return RetentionBusiness
}

// boundariesForMetric returns the histogram boundaries and labels for a metric name.
func boundariesForMetric(name string) ([]float64, []string) {
	switch {
	case strings.HasPrefix(name, "http_"):
		return DurationHTTPServerBoundaries, DurationHTTPServerLabels
	case strings.HasPrefix(name, "db_"):
		return DurationDBQueryBoundaries, DurationDBQueryLabels
	default:
		return DurationCartToPaymentBoundaries, DurationCartToPaymentLabels
	}
}

// cloneLabels returns a defensive copy of the label map.
func cloneLabels(labels L) L {
	if labels == nil {
		return L{}
	}
	out := make(L, len(labels))
	for k, v := range labels {
		out[k] = v
	}
	return out
}

// newKey returns a hex-encoded random idempotency key.
func newKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
