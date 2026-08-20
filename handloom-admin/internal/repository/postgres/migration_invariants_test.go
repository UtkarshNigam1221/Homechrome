package postgres_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
	apperrors "github.com/handloom/admin/pkg/errors"
)

// #230 cases 11 and 33 — the uniqueness that migrations 013 and 014 establish.
//
// The migrations themselves run once, at deploy, and cannot be re-run here. So
// this asserts the invariant they leave behind rather than the act of applying
// them: that the index admits what it must and rejects what it must. That is
// the part a later migration could silently undo.
func TestMigration_OrderScopedUniqueness(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	newProduct := func(t *testing.T, quantity, reserved int) string {
		t.Helper()
		p := newTestProduct(category.ID)
		require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID)
		})
		_, err := pool.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, p.ID)
		require.NoError(t, err)
		seedInventory(t, pool, p.ID, quantity, reserved)
		return p.ID
	}

	t.Run("013: the index carries source_id after 014", func(t *testing.T) {
		var indexDef string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT indexdef FROM pg_indexes WHERE indexname = 'idx_inv_txn_order_scoped'`,
		).Scan(&indexDef))

		require.Contains(t, indexDef, "UNIQUE", "order-scoped movements must be enforced, not merely checked")
		require.Contains(t, indexDef, "product_id")
		require.Contains(t, indexDef, "reference_id")
		require.Contains(t, indexDef, "type")
		require.Contains(t, indexDef, "source_id",
			"014 re-keys on source_id; without it a second refund is deduped into the first")
	})

	t.Run("013: source_id is NOT NULL, so nulls cannot defeat the index", func(t *testing.T) {
		var nullable string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT is_nullable FROM information_schema.columns
			 WHERE table_name = 'inventory_transactions' AND column_name = 'source_id'`,
		).Scan(&nullable))

		// Postgres treats NULLs as distinct in a unique index, so a nullable
		// column would quietly stop deduplicating the order-lifecycle movements
		// this index exists to protect.
		require.Equal(t, "NO", nullable)
	})

	t.Run("013: replaying the same movement is a no-op, not a second row", func(t *testing.T) {
		productID := newProduct(t, 20, 0)
		orderID := "order_" + uuid.New().String()[:8]

		first, err := repo.ReserveStock(ctx, productID, 3, orderID)
		require.NoError(t, err)

		// Same product, order, type and (empty) source. The index makes this an
		// enforced no-op rather than a convention: the existing row comes back
		// and stock does not move twice.
		again, err := repo.ReserveStock(ctx, productID, 3, orderID)
		require.NoError(t, err, "a same-quantity replay is idempotent, not an error")
		require.Equal(t, first.ID, again.ID, "the original row is returned")

		var rows int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_transactions
			 WHERE product_id = $1 AND reference_id = $2 AND type = $3`,
			productID, orderID, string(domain.InventoryTransactionTypeReserve),
		).Scan(&rows))
		require.Equal(t, 1, rows)

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 3, reserved, "the replay must not reserve again")
	})

	// #230 case 9. A replay with a different quantity is not the same request
	// arriving twice — it is two different intentions colliding, and answering
	// it with a silent no-op would lose one of them.
	t.Run("013: a replay with a different quantity is a conflict", func(t *testing.T) {
		productID := newProduct(t, 20, 0)
		orderID := "order_" + uuid.New().String()[:8]

		_, err := repo.ReserveStock(ctx, productID, 3, orderID)
		require.NoError(t, err)

		_, err = repo.ReserveStock(ctx, productID, 5, orderID)
		require.Error(t, err, "3 then 5 for one order is a conflict, not a no-op")

		appErr, ok := apperrors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, apperrors.ErrCodeConflict, appErr.Code)

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 3, reserved, "the refused replay must change nothing")
	})

	t.Run("033: two refunds on one order+product both move stock", func(t *testing.T) {
		productID := newProduct(t, 20, 6)
		orderID := "order_" + uuid.New().String()[:8]

		// Two write-offs against the same order and product, each caused by a
		// different refund. Pre-014 the second was deduped away: the money went
		// back twice and the stock moved once.
		first, err := repo.WriteOffStock(ctx, productID, 2, orderID, "refund_aaa")
		require.NoError(t, err)
		second, err := repo.WriteOffStock(ctx, productID, 2, orderID, "refund_bbb")
		require.NoError(t, err, "a second refund must keep its stock movement — this is what 014 re-keys")

		require.NotEqual(t, first.ID, second.ID)

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 16, qty, "both write-offs must have moved on-hand")
		require.Equal(t, 2, reserved)

		// And the same refund replayed is still a no-op: two refunds move stock
		// twice, one refund delivered twice moves it once.
		replay, err := repo.WriteOffStock(ctx, productID, 2, orderID, "refund_aaa")
		require.NoError(t, err)
		require.Equal(t, first.ID, replay.ID, "replaying one refund returns its original row")

		qtyAfter, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 16, qtyAfter, "the replay must not move stock a third time")
	})

	t.Run("013: the dedup archive from the migration exists", func(t *testing.T) {
		// 013 moved pre-existing duplicates aside rather than deleting them
		// outright. The table is the audit trail for that; losing it loses the
		// record of what the migration touched.
		var exists bool
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			                WHERE table_name = 'inventory_transactions_dedup_013')`,
		).Scan(&exists))
		if !exists {
			t.Skip("no dedup archive on this database — it is created only where 013 found duplicates")
		}
	})

}
