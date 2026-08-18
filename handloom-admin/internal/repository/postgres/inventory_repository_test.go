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

// A return adds back only what the order actually committed. The commit is
// best-effort, so "the order reached SHIPPED" does not mean the stock left —
// and adding back an amount that never left inflates stock and oversells.
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

// The drift signature: a reservation with no dispatch and no release. Every
// other pairing settles, so what this returns is stock no order transition will
// ever free.
func TestInventoryRepository_FindOrphanReservations(t *testing.T) {
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

	// Backdate so the age bound does not filter the fixtures out.
	age := func(t *testing.T, productID, orderID string) {
		t.Helper()
		_, err := pool.Exec(ctx,
			`UPDATE inventory_transactions SET created_at = NOW() - INTERVAL '48 hours'
			 WHERE product_id = $1 AND reference_id = $2`, productID, orderID)
		require.NoError(t, err)
	}

	findFor := func(t *testing.T, productID string) []*domain.OrphanReservation {
		t.Helper()
		all, err := repo.FindOrphanReservations(ctx, time.Hour, 100)
		require.NoError(t, err)

		var mine []*domain.OrphanReservation
		for _, o := range all {
			if o.ProductID == productID {
				mine = append(mine, o)
			}
		}
		return mine
	}

	// A refunded line's goods are written off — neither dispatched nor released.
	// Counting only those two made every fully refunded order look like stock
	// stuck in limbo, which is noise in the one report that has to be believed.
	t.Run("treats a write-off as settling the reservation", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 2, "order_written_off")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 2, "order_written_off", "refund_wo")
		require.NoError(t, err)
		age(t, productID, "order_written_off")

		require.Empty(t, findFor(t, productID), "the goods are accounted for")
	})

	t.Run("reports a reservation nothing settled", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_stuck")
		require.NoError(t, err)
		age(t, productID, "order_stuck")

		found := findFor(t, productID)
		require.Len(t, found, 1)
		require.Equal(t, "order_stuck", found[0].OrderID)
		require.Equal(t, 3, found[0].Quantity)
		require.NotEmpty(t, found[0].SKU, "the report has to name the product, not just its id")
	})

	t.Run("ignores a reservation the order dispatched", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_shipped")
		require.NoError(t, err)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_shipped", map[string]int{productID: 3}))
		age(t, productID, "order_shipped")

		require.Empty(t, findFor(t, productID))
	})

	t.Run("ignores a reservation the order released", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_cancelled")
		require.NoError(t, err)
		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_cancelled", map[string]int{productID: 3}))
		age(t, productID, "order_cancelled")

		require.Empty(t, findFor(t, productID))
	})

	// A checkout mid-payment holds a reservation legitimately; only an old one
	// is drift.
	t.Run("ignores a reservation younger than the age bound", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_in_flight")
		require.NoError(t, err)

		require.Empty(t, findFor(t, productID), "a fresh reservation is a live checkout")
	})

	// Another order settling its own reservation must not clear this one.
	t.Run("does not let one order settle another", func(t *testing.T) {
		productID := newProduct(t, 20, 0)
		_, err := repo.ReserveStock(ctx, productID, 2, "order_a")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 5, "order_b")
		require.NoError(t, err)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_b", map[string]int{productID: 5}))
		age(t, productID, "order_a")
		age(t, productID, "order_b")

		found := findFor(t, productID)
		require.Len(t, found, 1)
		require.Equal(t, "order_a", found[0].OrderID)
	})
}

// An order-scoped movement must be bounded by what that order still holds, not
// by the product's reserved_qty total. Once a refund has written part of an
// order off, a release sized from the order's lines is larger than the order's
// remaining reservation — and the difference comes out of another order's.
func TestInventoryRepository_OrderOutstandingReservation(t *testing.T) {
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

	t.Run("a cancel releases only what its own order still holds", func(t *testing.T) {
		productID := newProduct(t, 100, 0)

		_, err := repo.ReserveStock(ctx, productID, 3, "order_x")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 4, "order_y")
		require.NoError(t, err)

		// order_x refunds two of its three units, so it holds one.
		_, err = repo.WriteOffStock(ctx, productID, 2, "order_x", "refund_x")
		require.NoError(t, err)

		// Canceling passes the order's line quantity, which is now too large.
		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_x", map[string]int{productID: 3}))

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 4, reserved, "order_y's reservation must survive order_x's cancel")
	})

	t.Run("a dispatch commits only what its own order still holds", func(t *testing.T) {
		productID := newProduct(t, 100, 0)

		_, err := repo.ReserveStock(ctx, productID, 2, "order_c")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 5, "order_d")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 1, "order_c", "refund_c")
		require.NoError(t, err)

		require.NoError(t, repo.CommitOrderStock(ctx, "order_c", map[string]int{productID: 2}))

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 5, reserved, "order_d keeps its reservation")
		require.Equal(t, 98, qty, "one unit written off, one dispatched")
	})

	// A refund asking for more than the order holds is the caller being wrong,
	// not a remainder to settle — it must not quietly take a smaller bite.
	t.Run("refuses a refund larger than the order's outstanding reservation", func(t *testing.T) {
		productID := newProduct(t, 100, 0)

		_, err := repo.ReserveStock(ctx, productID, 1, "order_e")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 6, "order_f")
		require.NoError(t, err)

		_, err = repo.WriteOffStock(ctx, productID, 3, "order_e", "refund_e")
		require.Error(t, err)

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 100, qty, "a refused write-off changes nothing")
		require.Equal(t, 7, reserved)
	})
}

// ReleaseRefundedStock is the restock half of a refund: the units go back on
// sale, so only the reservation moves. Refund-scoped for the same reason the
// write-off is — two refunds against one order must both count.
func TestInventoryRepository_ReleaseRefundedStock(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	productID := func(t *testing.T, quantity, reserved int) string {
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

	t.Run("returns the units to sale without touching on hand", func(t *testing.T) {
		id := productID(t, 100, 10)

		txn, err := repo.ReleaseRefundedStock(ctx, id, 3, "order_rr", "refund_rr")
		require.NoError(t, err)
		require.Equal(t, domain.InventoryTransactionTypeRelease, txn.Type)

		qty, reserved, available := readInventory(t, pool, id)
		require.Equal(t, 100, qty)
		require.Equal(t, 7, reserved)
		require.Equal(t, 93, available, "the units are back on sale")
	})

	t.Run("counts two refunds against the same order and product", func(t *testing.T) {
		id := productID(t, 100, 10)

		_, err := repo.ReleaseRefundedStock(ctx, id, 2, "order_rr_two", "refund_a")
		require.NoError(t, err)
		_, err = repo.ReleaseRefundedStock(ctx, id, 2, "order_rr_two", "refund_b")
		require.NoError(t, err)

		_, reserved, _ := readInventory(t, pool, id)
		require.Equal(t, 6, reserved)
	})

	t.Run("is idempotent per refund", func(t *testing.T) {
		id := productID(t, 100, 10)

		_, err := repo.ReleaseRefundedStock(ctx, id, 2, "order_rr_same", "refund_same")
		require.NoError(t, err)
		_, err = repo.ReleaseRefundedStock(ctx, id, 2, "order_rr_same", "refund_same")
		require.NoError(t, err)

		_, reserved, _ := readInventory(t, pool, id)
		require.Equal(t, 8, reserved)
	})

	// The ledger still has to say which order the goods belonged to, or the
	// history and the orphan-reservation pairing lose the thread.
	t.Run("records the order as the reference, and the refund as the source", func(t *testing.T) {
		id := productID(t, 100, 10)
		_, err := repo.ReleaseRefundedStock(ctx, id, 1, "order_rr_ref", "refund_rr_ref")
		require.NoError(t, err)

		var refType, refID, sourceID string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT reference_type, reference_id, source_id FROM inventory_transactions
			 WHERE product_id = $1`, id).Scan(&refType, &refID, &sourceID))
		require.Equal(t, "ORDER", refType)
		require.Equal(t, "order_rr_ref", refID)
		require.Equal(t, "refund_rr_ref", sourceID)
	})
}

// A write-off is a refunded line whose goods are gone: the reservation goes and
// the stock goes with it. Same arithmetic as a dispatch, different meaning, and
// the ledger has to say which.
func TestInventoryRepository_WriteOffStock(t *testing.T) {
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

	t.Run("drops on hand and reserved together, leaving available alone", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		txn, err := repo.WriteOffStock(ctx, productID, 4, "order_wo", "refund_wo")
		require.NoError(t, err)
		require.Equal(t, domain.InventoryTransactionTypeWriteOff, txn.Type)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "the units were already unavailable while reserved")
	})

	t.Run("refuses to write off more than the order reserved", func(t *testing.T) {
		productID := newProduct(t, 100, 2)

		_, err := repo.WriteOffStock(ctx, productID, 5, "order_wo_over", "refund_over")
		require.Error(t, err)

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 100, qty, "a refused write-off changes nothing")
		require.Equal(t, 2, reserved)
	})

	t.Run("is idempotent per refund", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		_, err := repo.WriteOffStock(ctx, productID, 3, "order_wo_twice", "refund_twice")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_wo_twice", "refund_twice")
		require.NoError(t, err, "a replay must be a no-op, not an error")

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 97, qty)
		require.Equal(t, 7, reserved)
	})

	// An order can be refunded line by line over days. Keying the movement on
	// the order alone deduped the second refund into the first: the money went
	// back and the stock never moved.
	t.Run("moves stock for a second refund against the same order and product", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		_, err := repo.WriteOffStock(ctx, productID, 1, "order_wo_two", "refund_one")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 1, "order_wo_two", "refund_two")
		require.NoError(t, err)

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 98, qty, "both refunds have to move stock")
		require.Equal(t, 8, reserved)
	})

	// The order lifecycle still gets one movement of each type per product: an
	// order reserves once, dispatches once, releases once.
	t.Run("leaves the order-scoped guard intact", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		_, err := repo.ReleaseStock(ctx, productID, 2, "order_wo_guard")
		require.NoError(t, err)
		_, err = repo.ReleaseStock(ctx, productID, 2, "order_wo_guard")
		require.NoError(t, err, "a replayed release must still be a no-op")

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 8, reserved)
	})

	// A dispatch and a write-off move the same numbers, so only the type tells
	// an auditor what happened to the goods.
	t.Run("records itself as a write-off, not a dispatch", func(t *testing.T) {
		productID := newProduct(t, 100, 10)
		_, err := repo.WriteOffStock(ctx, productID, 2, "order_wo_type", "refund_type")
		require.NoError(t, err)

		var typ string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT type FROM inventory_transactions WHERE product_id = $1 AND reference_id = $2`,
			productID, "order_wo_type").Scan(&typ))
		require.Equal(t, "WRITE_OFF", typ)
	})
}
