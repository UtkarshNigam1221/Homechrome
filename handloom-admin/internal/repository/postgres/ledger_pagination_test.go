package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// #230 case 17 — paging the ledger must be stable when timestamps tie.
//
// Movements written inside the same second share created_at. Ordering on that
// alone is non-deterministic between pages, so a row can repeat on one page and
// vanish from the next. The ordering is (created_at, id) for exactly this
// reason; this pins it, because it is invisible until a busy product is paged.
func TestInventoryRepository_LedgerPagingIsStableOnTiedTimestamps(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	p := newTestProduct(category.ID)
	require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID) })
	_, err := pool.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, p.ID)
	require.NoError(t, err)
	seedInventory(t, pool, p.ID, 100, 0)

	// Nine movements as fast as the database will take them, then force every
	// created_at to the same instant so the tie is guaranteed rather than hoped for.
	const movements = 9
	for i := 0; i < movements; i++ {
		_, err := repo.ReserveStock(ctx, p.ID, 1, "order_tie_"+uuid.New().String()[:8])
		require.NoError(t, err)
	}
	_, err = pool.Exec(ctx,
		`UPDATE inventory_transactions SET created_at = NOW() WHERE product_id = $1`, p.ID)
	require.NoError(t, err)

	// Walk the cursor three at a time and collect ids.
	seen := make(map[string]int)
	var walked []string
	const pageSize = 3
	cursor := ""

	for guard := 0; guard < movements+pageSize; guard++ {
		page, err := repo.GetTransactions(ctx, p.ID, domain.PaginationRequest{
			Limit:  pageSize,
			Cursor: cursor,
		})
		require.NoError(t, err)
		if len(page.Transactions) == 0 {
			break
		}
		for _, txn := range page.Transactions {
			seen[txn.ID]++
			walked = append(walked, txn.ID)
		}
		if !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			break
		}
		cursor = page.Pagination.NextCursor
	}

	require.Len(t, walked, movements, "paging must return every row exactly once")
	for id, count := range seen {
		require.Equal(t, 1, count, "row %s appeared on more than one page", id)
	}

	// And the whole set in one page must be the same set.
	single, err := repo.GetTransactions(ctx, p.ID, domain.PaginationRequest{Limit: 100})
	require.NoError(t, err)
	require.Len(t, single.Transactions, movements)
}
