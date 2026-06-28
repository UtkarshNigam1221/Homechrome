package embedder

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvecpgx "github.com/pgvector/pgvector-go/pgx"
)

// NewPGPool initializes a pgxpool with two AfterConnect hooks:
//  1. Register pgvector OID so vector(N) can be scanned/encoded.
//  2. SET hnsw.ef_search=40 so vector queries on this connection use the
//     correct recall/latency tradeoff without per-query SQL.
//
// maxConns caps the pool size. For the embedder Lambda (concurrency 10),
// 10 is a sensible upper bound.
func NewPGPool(ctx context.Context, dsn string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	cfg.MaxConns = maxConns
	// Emit an OTel span per query, nested under the active request span, so the
	// /search trace shows the hybrid-search SQL time (not just one flat span).
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := pgvecpgx.RegisterTypes(ctx, conn); err != nil {
			return fmt.Errorf("register pgvector types: %w", err)
		}
		if _, err := conn.Exec(ctx, "SET hnsw.ef_search = 40"); err != nil {
			return fmt.Errorf("set hnsw.ef_search: %w", err)
		}
		return nil
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
