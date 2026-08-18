package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// seedInventory inserts an inventory row directly, bypassing product-create
// semantics so the starting numbers are explicit.
func seedInventory(t *testing.T, pool *pgxpool.Pool, productID string, quantity, reserved int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO inventory (id, product_id, quantity, reserved_qty, available_qty)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(), productID, quantity, reserved, quantity-reserved,
	)
	require.NoError(t, err)
}

// readInventory returns the current quantity, reserved_qty and available_qty.
func readInventory(t *testing.T, pool *pgxpool.Pool, productID string) (qty, reserved, available int) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT quantity, reserved_qty, available_qty FROM inventory WHERE product_id = $1`,
		productID,
	).Scan(&qty, &reserved, &available)
	require.NoError(t, err)
	return
}

func TestInventoryRepository_CommitStock(t *testing.T) {
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

	t.Run("commit decrements quantity and reserved, leaving available unchanged", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		txn, err := repo.CommitStock(ctx, productID, 4, "order_abc")
		require.NoError(t, err)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "available_qty must not move on commit")

		require.Equal(t, domain.InventoryTransactionTypeCommit, txn.Type)
		require.Equal(t, 4, txn.Quantity)
		require.Equal(t, 100, txn.PreviousQty)
		require.Equal(t, 96, txn.NewQty)
		require.Equal(t, "order_abc", txn.ReferenceID)
	})

	t.Run("commit writes exactly one ledger row", func(t *testing.T) {
		productID := newProduct(t, 50, 5)

		_, err := repo.CommitStock(ctx, productID, 2, "order_def")
		require.NoError(t, err)

		var count int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_transactions WHERE product_id = $1 AND type = $2`,
			productID, string(domain.InventoryTransactionTypeCommit),
		).Scan(&count))
		require.Equal(t, 1, count)
	})

	t.Run("commit beyond reserved is rejected and changes nothing", func(t *testing.T) {
		productID := newProduct(t, 100, 3)

		_, err := repo.CommitStock(ctx, productID, 5, "order_ghi")
		require.Error(t, err)
		require.ErrorContains(t, err, "insufficient stock")

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 100, qty)
		require.Equal(t, 3, reserved)
		require.Equal(t, 97, available)
	})

	t.Run("commit on a missing product returns not found", func(t *testing.T) {
		_, err := repo.CommitStock(ctx, "does-not-exist", 1, "order_jkl")
		require.Error(t, err)
	})
}

// The guard on ReleaseStock and CommitStock only reads the product-level
// reserved_qty total, so a repeat consumes whatever another order reserved.
// The ledger already records reference_id on every order-scoped mutation; these
// tests pin that it is now read back.
func TestInventoryRepository_OrderScopedIdempotency(t *testing.T) {
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

	ledgerRows := func(t *testing.T, productID, orderID string, typ domain.InventoryTransactionType) int {
		t.Helper()
		var n int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT count(*) FROM inventory_transactions
			 WHERE product_id = $1 AND reference_id = $2 AND type = $3`,
			productID, orderID, string(typ),
		).Scan(&n))
		return n
	}

	// The failure from the backlog: order X holds 2, order Y holds 3, and X
	// commits twice. The second commit passes the total-only guard and eats Y's
	// reservation, leaving Y unable to ship.
	t.Run("repeat commit does not consume another order's reservation", func(t *testing.T) {
		productID := newProduct(t, 10, 5) // X holds 2, Y holds 3

		_, err := repo.CommitStock(ctx, productID, 2, "order_X")
		require.NoError(t, err)

		_, err = repo.CommitStock(ctx, productID, 2, "order_X")
		require.NoError(t, err, "a repeat commit must be a no-op, not an error")

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 8, qty, "quantity must drop once, not twice")
		require.Equal(t, 3, reserved, "order Y's reservation must survive")
		require.Equal(t, 1, ledgerRows(t, productID, "order_X", domain.InventoryTransactionTypeCommit))

		// Y can still ship.
		_, err = repo.CommitStock(ctx, productID, 3, "order_Y")
		require.NoError(t, err)
		qty, reserved, _ = readInventory(t, pool, productID)
		require.Equal(t, 5, qty)
		require.Equal(t, 0, reserved)
	})

	t.Run("repeat release does not free another order's reservation", func(t *testing.T) {
		productID := newProduct(t, 10, 5)

		_, err := repo.ReleaseStock(ctx, productID, 2, "order_X")
		require.NoError(t, err)
		_, err = repo.ReleaseStock(ctx, productID, 2, "order_X")
		require.NoError(t, err, "a repeat release must be a no-op, not an error")

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 3, reserved, "only order X's 2 units may be released")
		require.Equal(t, 1, ledgerRows(t, productID, "order_X", domain.InventoryTransactionTypeRelease))
	})

	t.Run("repeat reserve does not double-reserve", func(t *testing.T) {
		productID := newProduct(t, 10, 0)

		_, err := repo.ReserveStock(ctx, productID, 4, "order_X")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 4, "order_X")
		require.NoError(t, err, "a repeat reserve must be a no-op, not an error")

		_, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 4, reserved)
		require.Equal(t, 6, available)
		require.Equal(t, 1, ledgerRows(t, productID, "order_X", domain.InventoryTransactionTypeReserve))
	})

	// #198: one transaction for the whole order, so a mid-order failure leaves
	// no half-applied state.
	t.Run("CommitOrderStock applies every line or none", func(t *testing.T) {
		good := newProduct(t, 10, 5)
		short := newProduct(t, 10, 1) // only 1 reserved, asking for 3 must fail

		err := repo.CommitOrderStock(ctx, "order_multi", map[string]int{good: 2, short: 3})
		require.Error(t, err, "a line that cannot commit must fail the whole order")

		qty, reserved, _ := readInventory(t, pool, good)
		require.Equal(t, 10, qty, "the good line must be rolled back")
		require.Equal(t, 5, reserved)
		require.Equal(t, 0, ledgerRows(t, good, "order_multi", domain.InventoryTransactionTypeCommit))
	})

	t.Run("CommitOrderStock commits all lines together", func(t *testing.T) {
		a := newProduct(t, 10, 5)
		b := newProduct(t, 20, 6)

		require.NoError(t, repo.CommitOrderStock(ctx, "order_ok", map[string]int{a: 2, b: 3}))

		qtyA, resA, _ := readInventory(t, pool, a)
		qtyB, resB, _ := readInventory(t, pool, b)
		require.Equal(t, 8, qtyA)
		require.Equal(t, 3, resA)
		require.Equal(t, 17, qtyB)
		require.Equal(t, 3, resB)

		// And it is idempotent as a whole.
		require.NoError(t, repo.CommitOrderStock(ctx, "order_ok", map[string]int{a: 2, b: 3}))
		qtyA, _, _ = readInventory(t, pool, a)
		require.Equal(t, 8, qtyA, "a replayed order commit must change nothing")
	})

	t.Run("ReleaseOrderStock releases all lines together", func(t *testing.T) {
		a := newProduct(t, 10, 5)
		b := newProduct(t, 20, 6)

		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_rel", map[string]int{a: 2, b: 3}))

		_, resA, _ := readInventory(t, pool, a)
		_, resB, _ := readInventory(t, pool, b)
		require.Equal(t, 3, resA)
		require.Equal(t, 3, resB)
	})
}
