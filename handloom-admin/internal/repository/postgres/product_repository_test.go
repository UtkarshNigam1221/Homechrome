package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvecpgx "github.com/pgvector/pgvector-go/pgx"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// newTestPool creates a pgxpool.Pool connected to the test database.
// It reads POSTGRES_DSN from the environment. Locally it skips when no database
// is reachable, so `go test ./...` stays usable without Docker. Under CI it
// fails instead: `go test` reports ok for a skipped test, so a misconfigured
// runner would silently stop verifying every repository query on a green gate.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/handloom?sslmode=disable"
	}

	unavailable := func(format string, args ...any) {
		t.Helper()
		if os.Getenv("CI") != "" {
			t.Fatalf(format, args...)
		}
		t.Skipf(format, args...)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		unavailable("postgres DSN unparseable: %v", err)
		return nil
	}

	// Match NewPool (client.go): register the pgvector codec on every
	// connection, or scanning the products.embedding column fails with
	// "unsupported data type". Without this the test pool does not behave
	// like the pool every caller actually uses.
	cfg.AfterConnect = pgvecpgx.RegisterTypes

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		unavailable("postgres not available (pool create): %v", err)
		return nil
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		unavailable("postgres not available (ping): %v", err)
		return nil
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedCategory inserts a minimal category and returns it.
func seedCategory(t *testing.T, pool *pgxpool.Pool) *domain.Category {
	t.Helper()
	now := time.Now().UTC()
	cat := &domain.Category{
		ID:     uuid.New().String(),
		Name:   fmt.Sprintf("test-cat-%s", uuid.New().String()[:8]),
		Slug:   fmt.Sprintf("test-cat-%s", uuid.New().String()[:8]),
		Status: domain.CategoryStatusActive,
	}
	cat.CreatedAt = now
	cat.UpdatedAt = now
	repo := postgres.NewCategoryRepository(pool)
	require.NoError(t, repo.Create(context.Background(), cat))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, cat.ID)
	})
	return cat
}

// newTestProduct builds a minimal product ready for repo.Create.
func newTestProduct(categoryID string) *domain.Product {
	id := uuid.New().String()
	return &domain.Product{
		ID:           id,
		Name:         fmt.Sprintf("test-product-%s", id[:8]),
		Slug:         fmt.Sprintf("test-product-%s", id[:8]),
		SKU:          fmt.Sprintf("SKU-%s", id[:8]),
		CategoryID:   categoryID,
		BasePrice:    10000,
		SellingPrice: 9000,
		CostPrice:    5000,
		Currency:     "INR",
		Status:       domain.ProductStatusActive,
		Tags:         []string{},
	}
}

func TestProductRepository_VideoRoundTrip(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewProductRepository(pool)
	category := seedCategory(t, pool)

	t.Run("create with video round-trips", func(t *testing.T) {
		p := newTestProduct(category.ID)
		p.VideoURL = "https://cdn.example.com/assets/VIDEO/2026/05/22/uuid.mp4"
		p.VideoPosterURL = "https://cdn.example.com/assets/IMAGE/2026/05/22/poster.jpg"

		err := repo.Create(context.Background(), p, nil)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, p.ID)
		})

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Equal(t, p.VideoURL, got.VideoURL)
		require.Equal(t, p.VideoPosterURL, got.VideoPosterURL)
	})

	t.Run("empty video URLs persist as empty strings", func(t *testing.T) {
		p := newTestProduct(category.ID)

		err := repo.Create(context.Background(), p, nil)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, p.ID)
		})

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Empty(t, got.VideoURL)
		require.Empty(t, got.VideoPosterURL)
	})

	t.Run("update clears video URLs", func(t *testing.T) {
		p := newTestProduct(category.ID)
		p.VideoURL = "https://cdn.example.com/assets/VIDEO/old.mp4"
		require.NoError(t, repo.Create(context.Background(), p, nil))
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, p.ID)
		})

		p.VideoURL = ""
		p.VideoPosterURL = ""
		require.NoError(t, repo.Update(context.Background(), p))

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Empty(t, got.VideoURL)
		require.Empty(t, got.VideoPosterURL)
	})
}
