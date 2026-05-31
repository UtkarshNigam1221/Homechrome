//go:build integration

package postgres_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/repository/postgres"
)

func TestUpsertCounters_InsertsNewRow(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewMetricsRepository(pool)
	ctx := context.Background()

	bucket := time.Now().UTC().Truncate(5 * time.Minute)
	h := sha256.Sum256([]byte("metric=test_insert\x00service=test\x00"))

	row := postgres.CounterRow{
		Metric:         "test.insert",
		Labels:         map[string]string{"service": "test"},
		LabelHash:      h[:],
		BucketStart:    bucket,
		Count:          1,
		SumValue:       42,
		RetentionClass: "service",
	}

	require.NoError(t, repo.UpsertCounters(ctx, []postgres.CounterRow{row}))

	// Verify the row exists.
	var count int64
	err := pool.QueryRow(ctx,
		`SELECT count FROM metric_counters WHERE metric = $1 AND bucket_start = $2`,
		row.Metric, bucket,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Cleanup.
	_, _ = pool.Exec(ctx, `DELETE FROM metric_counters WHERE metric = $1`, row.Metric)
}

func TestUpsertCounters_IncrementsExistingRow(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewMetricsRepository(pool)
	ctx := context.Background()

	bucket := time.Now().UTC().Truncate(5 * time.Minute)
	h := sha256.Sum256([]byte("metric=test_increment\x00service=test\x00"))

	row := postgres.CounterRow{
		Metric:         "test.increment",
		Labels:         map[string]string{"service": "test"},
		LabelHash:      h[:],
		BucketStart:    bucket,
		Count:          3,
		SumValue:       100,
		RetentionClass: "service",
	}

	require.NoError(t, repo.UpsertCounters(ctx, []postgres.CounterRow{row}))
	require.NoError(t, repo.UpsertCounters(ctx, []postgres.CounterRow{row}))

	var count, sumValue int64
	err := pool.QueryRow(ctx,
		`SELECT count, sum_value FROM metric_counters WHERE metric = $1 AND bucket_start = $2`,
		row.Metric, bucket,
	).Scan(&count, &sumValue)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
	assert.Equal(t, int64(200), sumValue)

	_, _ = pool.Exec(ctx, `DELETE FROM metric_counters WHERE metric = $1`, row.Metric)
}

func TestUpsertCounters_HandlesEmpty(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewMetricsRepository(pool)
	ctx := context.Background()

	err := repo.UpsertCounters(ctx, nil)
	require.NoError(t, err)

	err = repo.UpsertCounters(ctx, []postgres.CounterRow{})
	require.NoError(t, err)
}

func TestUpsertCounters_AggregatesSumValue(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewMetricsRepository(pool)
	ctx := context.Background()

	bucket := time.Now().UTC().Truncate(5 * time.Minute)
	h := sha256.Sum256([]byte("metric=test_sum\x00service=test\x00"))

	rows := []postgres.CounterRow{
		{
			Metric:         "test.sum",
			Labels:         map[string]string{"service": "test"},
			LabelHash:      h[:],
			BucketStart:    bucket,
			Count:          2,
			SumValue:       500,
			RetentionClass: "business",
		},
		{
			Metric:         "test.sum",
			Labels:         map[string]string{"service": "test"},
			LabelHash:      h[:],
			BucketStart:    bucket,
			Count:          3,
			SumValue:       750,
			RetentionClass: "business",
		},
	}

	// Second row in same batch will hit the ON CONFLICT path.
	require.NoError(t, repo.UpsertCounters(ctx, rows))

	var count, sumValue int64
	err := pool.QueryRow(ctx,
		`SELECT count, sum_value FROM metric_counters WHERE metric = $1 AND bucket_start = $2`,
		"test.sum", bucket,
	).Scan(&count, &sumValue)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.Equal(t, int64(1250), sumValue)

	_, _ = pool.Exec(ctx, `DELETE FROM metric_counters WHERE metric = $1`, "test.sum")
}
