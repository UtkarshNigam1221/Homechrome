// Package dbtracer wires pgx queries into the native metrics pipeline.
// Attaches as cfg.ConnConfig.Tracer on the pgx pool config.
package dbtracer

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/handloom/admin/pkg/metrics"
)

// Tracer satisfies pgx.QueryTracer. Emits db_query{} +
// db_query_duration{} for every query.
type Tracer struct {
	Service string
}

type queryStartTime struct{}

// TraceQueryStart stamps query-start time on the context.
func (t *Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartTime{}, time.Now())
}

// TraceQueryEnd emits the metrics for the completed query.
func (t *Tracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	start, _ := ctx.Value(queryStartTime{}).(time.Time)
	if start.IsZero() {
		return
	}
	op := opFromSQL(data.CommandTag.String())
	status := "ok"
	if data.Err != nil {
		status = "err"
	}
	metrics.Record(ctx, "db_query", metrics.L{
		metrics.LabelService:   t.Service,
		metrics.LabelOperation: op,
		metrics.LabelStatus:    status,
	})
	metrics.RecordDuration(ctx, "db_query_duration", time.Since(start), metrics.L{
		metrics.LabelService:   t.Service,
		metrics.LabelOperation: op,
	})
}

func opFromSQL(tag string) string {
	fields := strings.Fields(strings.ToLower(tag))
	if len(fields) == 0 {
		return "unknown"
	}
	switch fields[0] {
	case "select":
		return "select"
	case "insert":
		return "insert"
	case "update":
		return "update"
	case "delete":
		return "delete"
	default:
		return "other"
	}
}
