package main

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/migrations"
	"github.com/handloom/admin/pkg/logger"
)

// advisoryLockID is a fixed key for pg_advisory_lock to prevent concurrent migration runs.
const advisoryLockID = 7_248_301_945

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Migrator Lambda")

	lambda.Start(func(ctx context.Context) (string, error) {
		pool, err := postgres.NewPool(ctx, &cfg.Postgres)
		if err != nil {
			return "", fmt.Errorf("connect to postgres: %w", err)
		}
		defer pool.Close()

		m := &migrator{pool: pool, log: log}
		return m.run(ctx)
	})
}

type migrator struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

func (m *migrator) run(ctx context.Context) (string, error) {
	if err := m.withAdvisoryLock(ctx, func() error {
		return m.applyPending(ctx)
	}); err != nil {
		return "", err
	}
	return "migrations complete", nil
}

// withAdvisoryLock holds a session-level PostgreSQL advisory lock for the
// duration of fn, ensuring only one migrator runs at a time.
func (m *migrator) withAdvisoryLock(ctx context.Context, fn func() error) error {
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID) }()

	return fn()
}

// applyPending creates the tracking table, reads embedded SQL files, and
// executes any that haven't been applied yet.
func (m *migrator) applyPending(ctx context.Context) error {
	if err := m.ensureTrackingTable(ctx); err != nil {
		return err
	}

	applied, err := m.loadApplied(ctx)
	if err != nil {
		return err
	}

	pending, err := listPendingFiles(applied)
	if err != nil {
		return err
	}

	for _, filename := range pending {
		m.log.Info(fmt.Sprintf("Applying migration: %s", filename))
		if err := m.exec(ctx, filename); err != nil {
			return fmt.Errorf("apply %s: %w", filename, err)
		}
	}

	m.log.Info(fmt.Sprintf("Done: %d applied, %d already up-to-date", len(pending), len(applied)))
	return nil
}

func (m *migrator) ensureTrackingTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func (m *migrator) loadApplied(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	applied, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("scan applied migrations: %w", err)
	}

	set := make(map[string]bool, len(applied))
	for _, name := range applied {
		set[name] = true
	}
	return set, nil
}

// exec runs a single migration file inside a transaction and records it.
func (m *migrator) exec(ctx context.Context, filename string) error {
	sql, err := fs.ReadFile(migrations.FS, filename)
	if err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (filename, applied_at) VALUES ($1, $2)",
		filename, time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit(ctx)
}

// listPendingFiles returns embedded .sql filenames not yet in applied, sorted by name.
func listPendingFiles(applied map[string]bool) ([]string, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	var pending []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if !applied[e.Name()] {
			pending = append(pending, e.Name())
		}
	}
	sort.Strings(pending)
	return pending, nil
}
