package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecord_appendsEventToBuffer(t *testing.T) {
	ctx := WithBuffer(context.Background())
	Record(ctx, "order.placed", L{"state": "MH"})

	buf := bufferFromContext(ctx)
	require.Len(t, buf.events, 1)
	ev := buf.events[0]
	assert.Equal(t, "order.placed", ev.Metric)
	assert.Equal(t, int64(1), ev.Value)
	assert.Equal(t, "MH", ev.Labels["state"])
	assert.Equal(t, RetentionBusiness, ev.RetentionClass)
	assert.NotEmpty(t, ev.IdempotencyKey)
	assert.False(t, ev.EmittedAt.IsZero())
}

func TestRecordSum_setsSumValue(t *testing.T) {
	ctx := WithBuffer(context.Background())
	RecordSum(ctx, "order.gmv", 4999, L{"state": "KA"})

	buf := bufferFromContext(ctx)
	require.Len(t, buf.events, 1)
	ev := buf.events[0]
	assert.Equal(t, int64(1), ev.Value)
	assert.Equal(t, int64(4999), ev.SumValue)
}

func TestRecordDuration_addsBucketLabel(t *testing.T) {
	ctx := WithBuffer(context.Background())
	RecordDuration(ctx, "cart.to_payment", 45*time.Second, L{})

	buf := bufferFromContext(ctx)
	require.Len(t, buf.events, 1)
	assert.Equal(t, "le_2m", buf.events[0].Labels["bucket"])
}

func TestRecord_serviceMetric_setsRetentionService(t *testing.T) {
	prefixes := []string{"http_requests", "db_query", "aws_sdk_call", "lambda_cold_start", "otp_send", "rum_lcp"}
	for _, name := range prefixes {
		ctx := WithBuffer(context.Background())
		Record(ctx, name, L{})
		buf := bufferFromContext(ctx)
		assert.Equal(t, RetentionService, buf.events[0].RetentionClass, "metric=%s", name)
	}
}

func TestRecord_noBufferInCtx_doesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Record(context.Background(), "order.placed", L{})
	})
}

func TestRecord_cloneLabels_doesNotAliasInput(t *testing.T) {
	labels := L{"a": "1"}
	ctx := WithBuffer(context.Background())
	Record(ctx, "test", labels)
	labels["a"] = "mutated"

	buf := bufferFromContext(ctx)
	assert.Equal(t, "1", buf.events[0].Labels["a"])
}

func TestRecord_idempotencyKeysAreUnique(t *testing.T) {
	ctx := WithBuffer(context.Background())
	Record(ctx, "x", L{})
	Record(ctx, "x", L{})

	buf := bufferFromContext(ctx)
	require.Len(t, buf.events, 2)
	assert.NotEqual(t, buf.events[0].IdempotencyKey, buf.events[1].IdempotencyKey)
}
