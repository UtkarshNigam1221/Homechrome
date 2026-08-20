package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// #237 — opening stock must appear in the ledger, or SUM(ledger deltas) can
// never equal inventory.quantity and the ledger cannot be reconciled against
// the stock it describes.
func TestProductRepository_LedgersOpeningStock(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	category := seedCategory(t, pool)
	repo := postgres.NewProductRepository(pool)

	create := func(t *testing.T, initialStock int) string {
		t.Helper()
		p := newTestProduct(category.ID)
		inv := &domain.Inventory{
			ID:        p.ID,
			ProductID: p.ID,
			Quantity:  initialStock,
		}
		inv.CreatedBy = "usr_opening_test"
		require.NoError(t, repo.Create(ctx, p, inv))
		t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID) })
		return p.ID
	}

	rowsFor := func(t *testing.T, productID string) []struct {
		Type     string
		Quantity int
		Prev     int
		New      int
		Reason   string
	} {
		t.Helper()
		rows, err := pool.Query(ctx,
			`SELECT type, quantity, previous_qty, new_qty, reason
			   FROM inventory_transactions WHERE product_id = $1 ORDER BY created_at`, productID)
		require.NoError(t, err)
		defer rows.Close()

		var out []struct {
			Type     string
			Quantity int
			Prev     int
			New      int
			Reason   string
		}
		for rows.Next() {
			var r struct {
				Type     string
				Quantity int
				Prev     int
				New      int
				Reason   string
			}
			require.NoError(t, rows.Scan(&r.Type, &r.Quantity, &r.Prev, &r.New, &r.Reason))
			out = append(out, r)
		}
		return out
	}

	t.Run("a product created with stock gets one ADD row for it", func(t *testing.T) {
		productID := create(t, 12)

		rows := rowsFor(t, productID)
		require.Len(t, rows, 1, "exactly one opening entry")
		require.Equal(t, string(domain.InventoryTransactionTypeAdd), rows[0].Type)
		require.Equal(t, 12, rows[0].Quantity)
		require.Equal(t, 0, rows[0].Prev, "stock starts from nothing")
		require.Equal(t, 12, rows[0].New)
		require.Equal(t, "Opening stock", rows[0].Reason)
	})

	t.Run("the ledger now replays to the live balance", func(t *testing.T) {
		productID := create(t, 7)

		// The invariant #237 exists to restore: sum the deltas from zero and
		// land on what inventory actually holds.
		var replayed int
		for _, r := range rowsFor(t, productID) {
			require.Equal(t, string(domain.InventoryTransactionTypeAdd), r.Type)
			replayed += r.Quantity
		}

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, qty, replayed)
	})

	t.Run("a product created with no stock gets no row", func(t *testing.T) {
		productID := create(t, 0)
		require.Empty(t, rowsFor(t, productID), "a zero-quantity entry is noise")
	})
}
