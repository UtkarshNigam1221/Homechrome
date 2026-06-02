// Package main is the entry point for the worker-embedding-backfill Lambda.
// It re-embeds products whose embedding is NULL or stale relative to
// products.updated_at, in time-bounded batches. Triggered manually via
// GitHub Actions workflow_dispatch.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/embedder"
	"github.com/handloom/admin/internal/wire"
	"github.com/handloom/admin/pkg/slogx"
)

// Request is the JSON payload posted by the GitHub Actions workflow.
type Request struct {
	BatchSize          int  `json:"batch_size"`
	ForceReembed       bool `json:"force_reembed"`
	MaxDurationSeconds int  `json:"max_duration_seconds"`
}

// Response is the JSON payload returned. The workflow loops until
// `remaining_estimate` is 0.
type Response struct {
	Processed         int    `json:"processed"`
	RemainingEstimate int    `json:"remaining_estimate"`
	DurationMs        int64  `json:"duration_ms"`
	Error             string `json:"error,omitempty"`
}

const (
	defaultBatchSize   = 100
	defaultMaxDuration = 840 // seconds (~14 min, leaves 1 min headroom on a 15-min Lambda)
	embedderBatchSize  = 32
	deadlineSafetyMs   = 5_000
)

func main() {
	cfg := config.Load()
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting worker-embedding-backfill Lambda")

	ctx := context.Background()
	deps, err := wire.InitializeBackfillDeps(ctx, cfg)
	if err != nil {
		slog.Error("init backfill deps", "err", err)
		os.Exit(1)
	}
	h := &handler{pool: deps.Pool, emb: deps.EmbedderClient}
	lambda.Start(h.Handle)
}

type handler struct {
	pool *pgxpool.Pool
	emb  *embedder.Client
}

func (h *handler) Handle(ctx context.Context, req Request) (Response, error) {
	if req.BatchSize <= 0 {
		req.BatchSize = defaultBatchSize
	}
	if req.MaxDurationSeconds <= 0 {
		req.MaxDurationSeconds = defaultMaxDuration
	}
	if h.emb == nil {
		errMsg := "embedder client is nil — check EMBEDDER_FN_NAME and EMBEDDER_AUTH_KEY_PARAM env vars"
		slog.ErrorContext(ctx, errMsg)
		return Response{Error: errMsg}, nil
	}

	start := time.Now()
	deadline := start.Add(time.Duration(req.MaxDurationSeconds) * time.Second)
	processed := 0

	for time.Until(deadline) > deadlineSafetyMs*time.Millisecond {
		batch, err := h.fetchBatch(ctx, req.BatchSize, req.ForceReembed)
		if err != nil {
			return Response{Processed: processed, DurationMs: time.Since(start).Milliseconds(), Error: err.Error()}, err
		}
		if len(batch) == 0 {
			break
		}
		// embedAndWriteBatch returns the number of rows successfully written even
		// when it also returns an error (partial-chunk failure). Accumulate
		// progress before deciding whether to continue.
		written, err := h.embedAndWriteBatch(ctx, batch)
		processed += written
		if err != nil {
			// Stop processing further batches — the embedder may be unhealthy.
			// Return partial progress so the caller can report it; GH Actions
			// will see the error field and stop the workflow loop. Re-running
			// the workflow picks up where it left off (rows with
			// embedding IS NULL OR embedding_updated_at < updated_at).
			return Response{Processed: processed, DurationMs: time.Since(start).Milliseconds(), Error: err.Error()}, err
		}
	}

	remaining, _ := h.estimateRemaining(ctx, req.ForceReembed)
	return Response{
		Processed:         processed,
		RemainingEstimate: remaining,
		DurationMs:        time.Since(start).Milliseconds(),
	}, nil
}

type productRow struct {
	ID          string
	Name        string
	Description string
}

func (h *handler) fetchBatch(ctx context.Context, n int, force bool) ([]productRow, error) {
	var q string
	if force {
		q = `SELECT id, name, COALESCE(description, '') FROM products ORDER BY id LIMIT $1`
	} else {
		q = `SELECT id, name, COALESCE(description, '') FROM products
		     WHERE embedding IS NULL OR embedding_updated_at < updated_at
		     ORDER BY id LIMIT $1`
	}
	rows, err := h.pool.Query(ctx, q, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]productRow, 0, n)
	for rows.Next() {
		var p productRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// embedAndWriteBatch embeds and persists all chunks in batch.
// It returns (written, err) where written is the count of rows successfully
// persisted. On a chunk failure the function logs a warning, records the error,
// and returns immediately so the caller can stop further processing and report
// partial progress. Rows from successfully completed chunks before the failure
// are already persisted in Postgres.
func (h *handler) embedAndWriteBatch(ctx context.Context, batch []productRow) (int, error) {
	written := 0
	for i := 0; i < len(batch); i += embedderBatchSize {
		end := i + embedderBatchSize
		if end > len(batch) {
			end = len(batch)
		}
		chunk := batch[i:end]

		texts := make([]string, len(chunk))
		for j, p := range chunk {
			texts[j] = buildEmbeddingInput(p.Name, p.Description)
		}

		vecs, err := h.emb.Embed(ctx, texts...)
		if err != nil {
			slog.WarnContext(ctx, "embedder call failed; stopping batch",
				"chunk_start", i, "chunk_end", end-1, "err", err)
			return written, err
		}
		if len(vecs) != len(chunk) {
			err = fmt.Errorf("embedder returned %d vectors, expected %d", len(vecs), len(chunk))
			slog.WarnContext(ctx, "embedder vector count mismatch; stopping batch",
				"chunk_start", i, "chunk_end", end-1, "err", err)
			return written, err
		}

		for j, v := range vecs {
			if _, err := h.pool.Exec(ctx,
				`UPDATE products SET embedding = $1, embedding_updated_at = now() WHERE id = $2`,
				pgvector.NewVector(v), chunk[j].ID,
			); err != nil {
				slog.WarnContext(ctx, "postgres write failed; stopping batch",
					"product_id", chunk[j].ID, "err", err)
				return written, err
			}
		}
		written += len(chunk)
	}
	return written, nil
}

func (h *handler) estimateRemaining(ctx context.Context, force bool) (int, error) {
	var q string
	if force {
		q = `SELECT COUNT(*) FROM (SELECT 1 FROM products LIMIT 1000) sub`
	} else {
		q = `SELECT COUNT(*) FROM (
			SELECT 1 FROM products
			WHERE embedding IS NULL OR embedding_updated_at < updated_at
			LIMIT 1000
		) sub`
	}
	var n int
	if err := h.pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// buildEmbeddingInput mirrors the catalog service's embedding text format
// (name + "\n" + description) so newly-embedded rows match historic ones.
func buildEmbeddingInput(name, description string) string {
	if description == "" {
		return name
	}
	return name + "\n" + description
}
