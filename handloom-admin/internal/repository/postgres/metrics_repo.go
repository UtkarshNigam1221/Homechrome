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

	// Pre-aggregate before building the statement. Postgres refuses an
	// INSERT ... ON CONFLICT whose VALUES name the same conflict target twice
	// ("cannot affect row a second time", SQLSTATE 21000) — and a counter batch
	// repeating one key is the normal case, not an edge one: that is what a
	// counter is. Without this the whole flush fails and the batch is lost.
	rows = aggregateCounterRows(rows)

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

// aggregateCounterRows sums rows sharing a conflict target
// (metric, label_hash, bucket_start), so each key appears once in the
// statement. Input order is preserved for the first occurrence of each key,
// which keeps the generated SQL stable and diffable.
func aggregateCounterRows(rows []CounterRow) []CounterRow {
	// label_hash is []byte, so it cannot be a map key directly; the string
	// conversion is a value copy, which is what a key needs anyway.
	type key struct {
		metric    string
		labelHash string
		bucket    time.Time
	}

	index := make(map[key]int, len(rows))
	out := make([]CounterRow, 0, len(rows))

	for _, row := range rows {
		k := key{row.Metric, string(row.LabelHash), row.BucketStart}
		if at, seen := index[k]; seen {
			out[at].Count += row.Count
			out[at].SumValue += row.SumValue
			continue
		}
		index[k] = len(out)
		out = append(out, row)
	}
	return out
}
