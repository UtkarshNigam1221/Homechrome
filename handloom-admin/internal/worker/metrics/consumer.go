package metrics

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/repository/postgres"
	pkgmetrics "github.com/handloom/admin/pkg/metrics"
)

// metricsRepoIface is the minimal interface Handler needs from the repo.
// Satisfied by *postgres.MetricsRepository; declared here for testability.
type metricsRepoIface interface {
	UpsertCounters(ctx context.Context, rows []postgres.CounterRow) error
}

// aggregateKey uniquely identifies one (metric, labelHash, bucketStart, retentionClass) cell.
type aggregateKey struct {
	metric         string
	labelHash      string // hex of sha256 bytes
	bucketStart    time.Time
	retentionClass string
}

// aggregateCell accumulates count + sumValue for one cell.
type aggregateCell struct {
	labels         map[string]string
	labelHashBytes []byte
	count          int64
	sumValue       int64
}

// Handler is the SQS consumer handler for the metrics worker Lambda.
type Handler struct {
	repo  metricsRepoIface
	cache *IdempotencyCache
}

// NewHandler constructs a Handler.
func NewHandler(repo *postgres.MetricsRepository, cache *IdempotencyCache) *Handler {
	return &Handler{repo: repo, cache: cache}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
// It aggregates all events in the batch by (metric, labelHash, bucket5min,
// retentionClass), deduplicates via IdempotencyCache, then UPSERTs once.
// Per-message parse failures are reported as BatchItemFailures so SQS retries
// only those messages. A PG error marks all messages as failures.
func (h *Handler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	agg := make(map[aggregateKey]*aggregateCell)
	var parseFailures []events.SQSBatchItemFailure

	for _, record := range sqsEvent.Records {
		// Each SQS message body is a JSON array of events (one per emit site
		// in the originating request). Producer is pkg/metrics SQSPublisher.
		var batch []pkgmetrics.Event
		if err := json.Unmarshal([]byte(record.Body), &batch); err != nil {
			slog.ErrorContext(ctx, "metrics consumer: unmarshal failed",
				"message_id", record.MessageId, "error", err)
			parseFailures = append(parseFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		for _, evt := range batch {
			// Dedup by idempotency key.
			if evt.IdempotencyKey != "" {
				if h.cache.Has(evt.IdempotencyKey) {
					slog.DebugContext(ctx, "metrics consumer: dedup skip",
						"idempotency_key", evt.IdempotencyKey)
					continue
				}
				h.cache.Add(evt.IdempotencyKey)
			}

			labelHashBytes := computeLabelHash(evt.Labels)
			key := aggregateKey{
				metric:         evt.Metric,
				labelHash:      fmt.Sprintf("%x", labelHashBytes),
				bucketStart:    alignTo5Min(evt.EmittedAt),
				retentionClass: string(evt.RetentionClass),
			}

			cell, ok := agg[key]
			if !ok {
				labelsCopy := make(map[string]string, len(evt.Labels))
				for k, v := range evt.Labels {
					labelsCopy[k] = v
				}
				cell = &aggregateCell{
					labels:         labelsCopy,
					labelHashBytes: labelHashBytes,
				}
				agg[key] = cell
			}
			cell.count += evt.Value
			cell.sumValue += evt.SumValue
		}
	}

	if len(agg) == 0 {
		return events.SQSEventResponse{BatchItemFailures: parseFailures}, nil
	}

	rows := make([]postgres.CounterRow, 0, len(agg))
	for key, cell := range agg {
		rows = append(rows, postgres.CounterRow{
			Metric:         key.metric,
			Labels:         cell.labels,
			LabelHash:      cell.labelHashBytes,
			BucketStart:    key.bucketStart,
			Count:          cell.count,
			SumValue:       cell.sumValue,
			RetentionClass: key.retentionClass,
		})
	}

	if err := h.repo.UpsertCounters(ctx, rows); err != nil {
		slog.ErrorContext(ctx, "metrics consumer: upsert failed", "error", err)
		// Mark ALL messages as failures so SQS retries the full batch.
		allFailures := make([]events.SQSBatchItemFailure, 0, len(sqsEvent.Records))
		for _, r := range sqsEvent.Records {
			allFailures = append(allFailures, events.SQSBatchItemFailure{ItemIdentifier: r.MessageId})
		}
		return events.SQSEventResponse{BatchItemFailures: allFailures}, nil
	}

	return events.SQSEventResponse{BatchItemFailures: parseFailures}, nil
}

// alignTo5Min truncates t to the nearest 5-minute boundary (UTC).
func alignTo5Min(t time.Time) time.Time {
	return t.UTC().Truncate(5 * time.Minute)
}

// computeLabelHash returns the sha256 of canonical "k=v\x00" pairs sorted by key.
func computeLabelHash(labels map[string]string) []byte {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(labels[k])
		sb.WriteByte(0)
	}

	h := sha256.Sum256([]byte(sb.String()))
	return h[:]
}
