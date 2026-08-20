package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/repository/postgres"
)

// TestInventoryRepository_FindOrphanReservations_Limit covers the one part of
// #230 case 14 that TestInventoryRepository_FindOrphanReservations does not:
// that the row cap is honoured.
//
// It matters because scripts/reconcile-inventory warns when the result length
// equals the limit — "there may be more" — and that warning is only meaningful
// if the limit actually truncates.
func TestInventoryRepository_FindOrphanReservations_Limit(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	// Three products, each with one stranded reservation older than the bound.
	for i := 0; i < 3; i++ {
		p := newTestProduct(category.ID)
		require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID)
		})
		_, err := pool.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, p.ID)
		require.NoError(t, err)
		seedInventory(t, pool, p.ID, 10, 2)

		_, err = repo.ReserveStock(ctx, p.ID, 2, "order_limit_"+p.ID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx,
			`UPDATE inventory_transactions SET created_at = NOW() - INTERVAL '48 hours'
			 WHERE product_id = $1`, p.ID)
		require.NoError(t, err)
	}

	all, err := repo.FindOrphanReservations(ctx, time.Hour, 500)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(all), 3, "the three fixtures must be strandable")

	capped, err := repo.FindOrphanReservations(ctx, time.Hour, 2)
	require.NoError(t, err)
	require.Len(t, capped, 2, "limit must truncate, or the script's 'there may be more' warning lies")

	// Oldest first, so a truncated report shows the longest-stranded stock —
	// the rows most worth acting on.
	for i := 1; i < len(capped); i++ {
		require.False(t, capped[i].ReservedAt.Before(capped[i-1].ReservedAt),
			"orphans must come back oldest first")
	}
}
