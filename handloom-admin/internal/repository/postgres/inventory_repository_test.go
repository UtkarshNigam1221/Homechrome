package postgres_test

import (
	"context"
	"testing"
	"time"

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

// The old guard read only the product-level reserved_qty total, so a repeat
// consumed another order's reservation. These pin that reference_id is read back.
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

	// The failure from the backlog: X holds 2, Y holds 3, X commits twice — the
	// second passes the total-only guard and eats Y's reservation.
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

// A return adds back only what the order committed. Commit is best-effort, so
// reaching SHIPPED does not mean the stock left; adding it back would oversell.
func TestInventoryRepository_RestockOrderStock(t *testing.T) {
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

	t.Run("restocks exactly what was committed", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r1", map[string]int{productID: 4}))

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 6, qty)

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r1"))

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "the committed units come back")
		require.Equal(t, 0, reserved)
		require.Equal(t, 10, available)
	})

	// The #200 regression: before the commit-gated restock, a return after a
	// failed commit added stock that had never been decremented.
	t.Run("restocks nothing when the commit never happened", func(t *testing.T) {
		productID := newProduct(t, 10, 4)

		require.NoError(t, repo.RestockOrderStock(ctx, "order_never_shipped"))

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "stock must not be inflated by a return with no commit")
	})

	t.Run("is idempotent", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r2", map[string]int{productID: 4}))

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r2"))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_r2"))

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "a replayed return must not add twice")
	})

	// #199: AddStock stamps last_restock_at, which means "when did we last
	// replenish this SKU". A customer return is not a replenishment.
	t.Run("does not stamp last_restock_at", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r3", map[string]int{productID: 4}))

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r3"))

		var lastRestock *time.Time
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT last_restock_at FROM inventory WHERE product_id = $1`, productID,
		).Scan(&lastRestock))
		require.Nil(t, lastRestock, "a return is not a supplier replenishment")
	})

	// #199: returns land as their own ledger type, keyed to the order, rather
	// than as an ADD distinguishable only by a reason prefix.
	t.Run("writes a RETURN row keyed to the order", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r4", map[string]int{productID: 4}))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_r4"))

		var typ, refType, refID string
		var quantity int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT type, reference_type, reference_id, quantity
			 FROM inventory_transactions
			 WHERE product_id = $1 AND reference_id = $2 AND type = $3`,
			productID, "order_r4", string(domain.InventoryTransactionTypeReturn),
		).Scan(&typ, &refType, &refID, &quantity))

		require.Equal(t, string(domain.InventoryTransactionTypeReturn), typ)
		require.Equal(t, "ORDER", refType)
		require.Equal(t, "order_r4", refID)
		require.Equal(t, 4, quantity)
	})

	// Only the committed lines come back, not every line on the order.
	t.Run("a partially committed order restocks only the committed line", func(t *testing.T) {
		shipped := newProduct(t, 10, 3)
		neverShipped := newProduct(t, 10, 3)

		require.NoError(t, repo.CommitOrderStock(ctx, "order_partial", map[string]int{shipped: 3}))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_partial"))

		qtyShipped, _, _ := readInventory(t, pool, shipped)
		qtyNever, _, _ := readInventory(t, pool, neverShipped)
		require.Equal(t, 10, qtyShipped, "the committed line comes back")
		require.Equal(t, 10, qtyNever, "the uncommitted line was never decremented, so nothing to add")
	})
}
