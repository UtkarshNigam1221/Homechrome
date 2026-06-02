package embedder

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/repository/postgres"
)

// TestSearcher_Hybrid_ReturnsKnownProduct is an integration test against a
// real Postgres with pgvector + migrations applied. Skipped when no DSN.
func TestSearcher_Hybrid_ReturnsKnownProduct(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set; run via make test-integration with pgvector-enabled postgres")
	}
	ctx := context.Background()
	pool, err := NewPGPool(ctx, dsn, 2)
	require.NoError(t, err)
	defer pool.Close()

	// Pick any existing category to satisfy FK; tests assume seed data exists.
	var catID string
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&catID))

	// Seed: insert one product with a known unit vector along axis 0.
	knownVec := make([]float32, EmbeddingDim)
	knownVec[0] = 1.0
	const id = "prod_test_search"

	_, err = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO products (id, name, slug, sku, category_id, status, base_price, selling_price, embedding, embedding_updated_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', 10000, 10000, $6, now(), 'tester', now(), now())
	`), id, "Silk Saree Test", "silk-saree-test-"+id, "TEST-SKU-"+id, catID, pgvector.NewVector(knownVec))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id) })

	productRepo := postgres.NewProductRepository(pool)
	s := NewSearcher(pool, productRepo, Weights{Semantic: 0.6, Keyword: 0.3, Trigram: 0.1})

	out, err := s.Search(ctx, knownVec, SearchRequest{Query: "silk", Limit: 5}, 0)
	require.NoError(t, err)
	require.NotEmpty(t, out.Data, "search should return at least one row")
	// Known product should be in top results.
	found := false
	for _, p := range out.Data {
		if p.ID == id {
			found = true
			break
		}
	}
	require.True(t, found, "seeded product must appear in results")
}
