// Package postgres provides PostgreSQL-backed repository implementations
// for catalog data (categories, products, inventory).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	appconfig "github.com/handloom/admin/internal/config"
)

// NewPool creates a PostgreSQL connection pool.
// Uses POSTGRES_DSN environment variable in all environments.
func NewPool(ctx context.Context, pgCfg *appconfig.PostgresConfig) (*pgxpool.Pool, error) {
	if pgCfg.DSN == "" {
		return nil, fmt.Errorf("no postgres DSN configured (set POSTGRES_DSN)")
	}

	cfg, err := pgxpool.ParseConfig(pgCfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
