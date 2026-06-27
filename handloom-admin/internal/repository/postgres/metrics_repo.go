package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CounterRow is a single 5-min-bucket increment for one (metric, labels) key.
type CounterRow struct {
	Metric         string
	Labels         map[string]string
	LabelHash      []byte
	BucketStart    time.Time
	Count          int64
	SumValue       int64
	RetentionClass string
}

// MetricsRepository persists metric counter rows to PostgreSQL.
type MetricsRepository struct {
	pool *pgxpool.Pool
}

// NewMetricsRepository creates a new MetricsRepository backed by PostgreSQL.
func NewMetricsRepository(pool *pgxpool.Pool) *MetricsRepository {
	return &MetricsRepository{pool: pool}
}

// UpsertCounters atomically applies a batch of counter deltas to metric_counters.
// Idempotency is the caller's responsibility (dedup before calling).
func (r *MetricsRepository) UpsertCounters(ctx context.Context, rows []CounterRow) error {
	if len(rows) == 0 {
		return nil
	}

	sql := `INSERT INTO metric_counters
              (metric, labels, label_hash, bucket_start, count, sum_value, retention_class)
            VALUES `

	placeholders := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*7)
	for i, row := range rows {
		labelsJSON, err := json.Marshal(row.Labels)
		if err != nil {
			return fmt.Errorf("marshal labels for %s: %w", row.Metric, err)
		}
		base := i * 7
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7,
		))
		args = append(args,
			row.Metric, labelsJSON, row.LabelHash, row.BucketStart,
			row.Count, row.SumValue, row.RetentionClass,
		)
	}

	sql += strings.Join(placeholders, ", ")
	sql += ` ON CONFLICT (metric, label_hash, bucket_start) DO UPDATE SET
                count      = metric_counters.count     + EXCLUDED.count,
                sum_value  = metric_counters.sum_value + EXCLUDED.sum_value,
                updated_at = now()`

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("upsert counters: %w", err)
	}
	return tx.Commit(ctx)
}
