package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/pkg/errors"
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

// reserveViaLedger records a RESERVE row so the order's outstanding reservation is
// derivable. seedInventory sets reserved_qty alone, which the ledger cannot explain.
func reserveViaLedger(t *testing.T, pool *pgxpool.Pool, productID, orderID string, quantity int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO inventory_transactions
		   (id, product_id, type, quantity, previous_qty, new_qty, reason,
		    reference_type, reference_id, source_id)
		 VALUES ($1, $2, 'RESERVE', $3, 0, $3, '', 'ORDER', $4, '')`,
		uuid.New().String(), productID, quantity, orderID,
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

	newProduct := func(t *testing.T, quantity, reserved int, id ...string) string {
		t.Helper()
		p := newTestProduct(category.ID)
		if len(id) > 0 {
			p.ID = id[0]
		}
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

	// A payment re-check can find a shipped order's payment FAILED, and the failure
	// path releases. The goods are already gone, so the release must move nothing.
	t.Run("release after a dispatch moves nothing back", func(t *testing.T) {
		productID := newProduct(t, 10, 5) // X holds 2, Y holds 3

		require.NoError(t, repo.CommitOrderStock(ctx, "order_X", map[string]int{productID: 2}))
		qtyAfterShip, reservedAfterShip, _ := readInventory(t, pool, productID)

		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_X", map[string]int{productID: 2}))

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, qtyAfterShip, qty, "dispatched goods must not come back on a release")
		require.Equal(t, reservedAfterShip, reserved, "the reservation was already settled by the commit")
		require.Equal(t, 3, reserved, "order Y's reservation must survive")
		require.Equal(t, 0, ledgerRows(t, productID, "order_X", domain.InventoryTransactionTypeRelease),
			"nothing was outstanding, so no release row may be written")
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

	// A replay of a different amount is divergence, not the no-op the index exists
	// for. Silently dropping it would move zero units and report success.
	t.Run("a replay with a different quantity is a conflict", func(t *testing.T) {
		productID := newProduct(t, 10, 5)

		_, err := repo.ReserveStock(ctx, productID, 2, "order_q")
		require.NoError(t, err)

		_, err = repo.ReserveStock(ctx, productID, 3, "order_q")
		require.Error(t, err, "a different quantity must not be dropped silently")
		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeConflict, appErr.Code)

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 7, reserved, "the first reservation stands, the replay moved nothing")
		require.Equal(t, 1, ledgerRows(t, productID, "order_q", domain.InventoryTransactionTypeReserve))
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
		// IDs are pinned: orderMovementAll applies in sorted order, so "good" must be
		// applied before "short" fails or the rollback is never exercised.
		good := newProduct(t, 10, 5, "prod_013_a_good")
		short := newProduct(t, 10, 1, "prod_013_z_short") // only 1 reserved, asking 3 fails

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

	// #226 F3: release is a rollback path. An unreleasable line must not strand the
	// reservations the other lines still hold, whichever order they are applied in.
	t.Run("ReleaseOrderStock releases what it can when one line fails", func(t *testing.T) {
		good := newProduct(t, 10, 5)
		unreserved := newProduct(t, 10, 0) // nothing reserved, releasing 3 must fail

		err := repo.ReleaseOrderStock(ctx, "order_rel_partial", map[string]int{good: 2, unreserved: 3})
		require.Error(t, err, "the failing line must still be reported")

		_, resGood, _ := readInventory(t, pool, good)
		require.Equal(t, 3, resGood, "the releasable line must not be rolled back")
		_, resOther, _ := readInventory(t, pool, unreserved)
		require.Equal(t, 0, resOther)
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

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r1", "admin_test"))

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "the committed units come back")
		require.Equal(t, 0, reserved)
		require.Equal(t, 10, available)
	})

	// The #200 regression: before the commit-gated restock, a return after a
	// failed commit added stock that had never been decremented.
	t.Run("restocks nothing when the commit never happened", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		// Reserved against this order but never committed, so the order does reference
		// the product — otherwise the assertion holds however the restock behaves.
		_, err := repo.ReserveStock(ctx, productID, 2, "order_never_shipped")
		require.NoError(t, err)

		require.NoError(t, repo.RestockOrderStock(ctx, "order_never_shipped", "admin_test"))

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "stock must not be inflated by a return with no commit")
		var returnRows int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT count(*) FROM inventory_transactions
			 WHERE product_id = $1 AND reference_id = $2 AND type = $3`,
			productID, "order_never_shipped", string(domain.InventoryTransactionTypeReturn),
		).Scan(&returnRows))
		require.Equal(t, 0, returnRows, "no RETURN row without a COMMIT to undo")
	})

	// #226 F5: a commit that failed at SHIPPED leaves the reservation held, and a
	// returned order can no longer reach CANCELLED to free it.
	t.Run("releases a reservation the order never committed", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_strand")
		require.NoError(t, err)
		_, res, avail := readInventory(t, pool, productID)
		require.Equal(t, 3, res)
		require.Equal(t, 7, avail)

		require.NoError(t, repo.RestockOrderStock(ctx, "order_strand", "admin_test"))

		qty, res, avail := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "nothing committed, so nothing to add back")
		require.Equal(t, 0, res, "the stranded reservation must be freed")
		require.Equal(t, 10, avail)
	})

	// DeleteByProductID drops only the inventory row, so a COMMIT row can outlive it.
	// That must surface as an error, not be silently skipped and never restocked.
	t.Run("a committed line with no inventory row errors", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_noinv", map[string]int{productID: 4}))
		require.NoError(t, repo.DeleteByProductID(ctx, productID))

		err := repo.RestockOrderStock(ctx, "order_noinv", "admin_test")
		require.Error(t, err, "a missing inventory row must not be skipped silently")
	})

	// The order-scoped rewrite wrote created_by='' and left updated_by alone, losing
	// the actor AddStock used to record.
	t.Run("records the actor on the return", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_actor", map[string]int{productID: 4}))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_actor", "admin_actor"))

		var ledgerActor, inventoryActor string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT created_by FROM inventory_transactions
			 WHERE product_id = $1 AND reference_id = $2 AND type = $3`,
			productID, "order_actor", string(domain.InventoryTransactionTypeReturn),
		).Scan(&ledgerActor))
		require.Equal(t, "admin_actor", ledgerActor)

		require.NoError(t, pool.QueryRow(ctx,
			`SELECT updated_by FROM inventory WHERE product_id = $1`, productID,
		).Scan(&inventoryActor))
		require.Equal(t, "admin_actor", inventoryActor)
	})

	// System-driven movements pass no actor and must not erase the one already there.
	t.Run("a commit does not clobber the recorded actor", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		_, err := pool.Exec(ctx,
			`UPDATE inventory SET updated_by = $1 WHERE product_id = $2`, "admin_earlier", productID)
		require.NoError(t, err)

		require.NoError(t, repo.CommitOrderStock(ctx, "order_keep", map[string]int{productID: 4}))

		var actor string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT updated_by FROM inventory WHERE product_id = $1`, productID,
		).Scan(&actor))
		require.Equal(t, "admin_earlier", actor, "a system movement must not blank the actor")
	})

	t.Run("is idempotent", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r2", map[string]int{productID: 4}))

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r2", "admin_test"))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_r2", "admin_test"))

		qty, _, _ := readInventory(t, pool, productID)
		require.Equal(t, 10, qty, "a replayed return must not add twice")
	})

	// #199: AddStock stamps last_restock_at, which means "when did we last
	// replenish this SKU". A customer return is not a replenishment.
	t.Run("does not stamp last_restock_at", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r3", map[string]int{productID: 4}))

		// Pre-stamped: starting from NULL, require.Nil would also pass if the restock
		// never ran at all.
		stamp := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
		_, err := pool.Exec(ctx,
			`UPDATE inventory SET last_restock_at = $1 WHERE product_id = $2`, stamp, productID)
		require.NoError(t, err)

		require.NoError(t, repo.RestockOrderStock(ctx, "order_r3", "admin_test"))

		var lastRestock *time.Time
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT last_restock_at FROM inventory WHERE product_id = $1`, productID,
		).Scan(&lastRestock))
		require.NotNil(t, lastRestock)
		require.True(t, stamp.Equal(*lastRestock), "a return is not a supplier replenishment")
	})

	// #199: returns land as their own ledger type, keyed to the order, rather
	// than as an ADD distinguishable only by a reason prefix.
	t.Run("writes a RETURN row keyed to the order", func(t *testing.T) {
		productID := newProduct(t, 10, 4)
		require.NoError(t, repo.CommitOrderStock(ctx, "order_r4", map[string]int{productID: 4}))
		require.NoError(t, repo.RestockOrderStock(ctx, "order_r4", "admin_test"))

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
		require.NoError(t, repo.RestockOrderStock(ctx, "order_partial", "admin_test"))

		qtyShipped, _, _ := readInventory(t, pool, shipped)
		qtyNever, _, _ := readInventory(t, pool, neverShipped)
		require.Equal(t, 10, qtyShipped, "the committed line comes back")
		require.Equal(t, 10, qtyNever, "the uncommitted line was never decremented, so nothing to add")
	})
}

// The drift signature: a reservation with no dispatch and no release. Every other
// pairing settles, so what is left is stock no transition will free.
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

	// A write-off settles a reservation: the units are gone, not stranded. Without
	// this every fully refunded order is reported as stuck stock, which is noise in
	// the one report that has to be believed.
	t.Run("ignores a reservation a refund wrote off", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_written_off")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_written_off", "refund_1")
		require.NoError(t, err)
		age(t, productID, "order_written_off")

		require.Empty(t, findFor(t, productID))
	})

	// Netted, not tested for presence: one unit settled out of three leaves two
	// stranded, and reporting nothing there hides exactly the drift this looks for.
	t.Run("reports what is left of a partly settled reservation", func(t *testing.T) {
		productID := newProduct(t, 10, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_partly")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 1, "order_partly", "refund_1")
		require.NoError(t, err)
		age(t, productID, "order_partly")

		found := findFor(t, productID)
		require.Len(t, found, 1, "two units are still held")
		require.Equal(t, 2, found[0].Quantity, "the report gives what is outstanding, not what was reserved")
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

// #227: same-second movements are routine, and created_at alone left the sort
// unstable, so pagination could repeat a row on one page and skip it on the next.
func TestInventoryRepository_GetTransactions_StableAcrossPages(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	p := newTestProduct(category.ID)
	require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID) })

	// Six movements sharing one timestamp: created_at cannot order them at all.
	tied := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	for i := range 6 {
		_, err := pool.Exec(ctx,
			`INSERT INTO inventory_transactions
			   (id, product_id, type, quantity, previous_qty, new_qty, reason,
			    reference_type, reference_id, created_at)
			 VALUES ($1, $2, 'ADD', 1, $3, $4, '', '', '', $5)`,
			fmt.Sprintf("txn_tie_%d", i), p.ID, i, i+1, tied)
		require.NoError(t, err)
	}

	// Page through two at a time, following next_cursor exactly as the UI does,
	// and collect every id the reader is shown.
	seen := []string{}
	cursor := ""
	for range 4 {
		page, err := repo.GetTransactions(ctx, p.ID,
			domain.PaginationRequest{Limit: 2, Cursor: cursor})
		require.NoError(t, err)
		for _, txn := range page.Transactions {
			seen = append(seen, txn.ID)
		}
		if page.Pagination.NextCursor == "" {
			break
		}
		cursor = page.Pagination.NextCursor
	}

	require.Len(t, seen, 6, "every movement must appear exactly once across the pages")

	// ids ascend with insertion, so id DESC reverses heap order. Asserting the exact
	// sequence discriminates; uniqueness does not, as heap order is stable here.
	require.Equal(t,
		[]string{"txn_tie_5", "txn_tie_4", "txn_tie_3", "txn_tie_2", "txn_tie_1", "txn_tie_0"},
		seen,
		"tied timestamps must fall back to id DESC, the same way on every page")
}

// A write-off is a refunded line whose goods are gone: reservation and stock both go.
// Same arithmetic as a dispatch, different meaning, and the ledger must say which.
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

		txn, err := repo.WriteOffStock(ctx, productID, 4, "order_wo", "refund_1")
		require.NoError(t, err)
		require.Equal(t, domain.InventoryTransactionTypeWriteOff, txn.Type)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "the units were already unavailable while reserved")
	})

	t.Run("refuses to write off more than the order reserved", func(t *testing.T) {
		productID := newProduct(t, 100, 2)

		_, err := repo.WriteOffStock(ctx, productID, 5, "order_wo_over", "refund_1")
		require.Error(t, err)

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 100, qty, "a refused write-off changes nothing")
		require.Equal(t, 2, reserved)
	})

	t.Run("is idempotent per refund", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		_, err := repo.WriteOffStock(ctx, productID, 3, "order_wo_twice", "refund_1")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_wo_twice", "refund_1")
		require.NoError(t, err, "a replay must be a no-op, not an error")

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 97, qty)
		require.Equal(t, 7, reserved)
	})

	// The defect this scoping exists for: before source_id the second refund of one
	// product was read as a replay of the first, so the money went back and the
	// stock stayed put.
	t.Run("a second refund of the same product moves stock again", func(t *testing.T) {
		productID := newProduct(t, 100, 10)
		reserveViaLedger(t, pool, productID, "order_wo_two_refunds", 10)

		_, err := repo.WriteOffStock(ctx, productID, 3, "order_wo_two_refunds", "refund_1")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 4, "order_wo_two_refunds", "refund_2")
		require.NoError(t, err, "a different refund is not a replay")

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 93, qty, "both write-offs must reach on-hand")
		require.Equal(t, 3, reserved)
	})

	// The order can only give back what it still holds, and a refund asking for more
	// is a caller error rather than a remainder to settle quietly.
	t.Run("refuses a refund larger than the order's outstanding reservation", func(t *testing.T) {
		productID := newProduct(t, 100, 10)
		reserveViaLedger(t, pool, productID, "order_wo_bound", 4)

		_, err := repo.WriteOffStock(ctx, productID, 5, "order_wo_bound", "refund_1")
		require.Error(t, err, "4 units held, 5 asked for")

		qty, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 100, qty, "a refused write-off changes nothing")
		require.Equal(t, 10, reserved)
	})

	// reserved_qty is product-wide, so sizing a cancel or dispatch from the order's own
	// lines takes the difference out of whatever another order is holding.
	t.Run("a cancel after a partial refund leaves another order's reservation alone", func(t *testing.T) {
		productID := newProduct(t, 100, 0)
		_, err := repo.ReserveStock(ctx, productID, 3, "order_ours")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 2, "order_theirs")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 1, "order_ours", "refund_1")
		require.NoError(t, err)

		// The caller passes the ordered 3; only 2 are still held.
		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_ours", map[string]int{productID: 3}))

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 2, reserved, "the other order's two units must survive")
	})

	// The caller keeps passing the ordered quantity, so a replayed transition has to
	// compare against what the clamp produced rather than what it asked for.
	t.Run("a replayed cancel after a partial refund is a no-op, not a conflict", func(t *testing.T) {
		productID := newProduct(t, 100, 0)
		_, err := repo.ReserveStock(ctx, productID, 5, "order_replay")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 2, "order_replay", "refund_1")
		require.NoError(t, err)

		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_replay", map[string]int{productID: 5}))
		_, afterFirst, _ := readInventory(t, pool, productID)

		require.NoError(t, repo.ReleaseOrderStock(ctx, "order_replay", map[string]int{productID: 5}),
			"a replay must not be reported as divergence")

		_, afterSecond, _ := readInventory(t, pool, productID)
		require.Equal(t, afterFirst, afterSecond, "and it must move nothing")
	})

	// Without the write-off in the outstanding sum, a second refund passes the
	// product-level guard and eats another order's reservation.
	t.Run("a second refund cannot exceed what the first left", func(t *testing.T) {
		productID := newProduct(t, 100, 0)
		_, err := repo.ReserveStock(ctx, productID, 5, "order_two_refunds")
		require.NoError(t, err)
		_, err = repo.ReserveStock(ctx, productID, 4, "order_other")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_two_refunds", "refund_1")
		require.NoError(t, err)

		// 2 left on this order, so 3 is a caller error even though the product holds 6.
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_two_refunds", "refund_2")
		require.Error(t, err, "the order holds 2, not 3")

		_, reserved, _ := readInventory(t, pool, productID)
		require.Equal(t, 6, reserved, "nothing may come out of the other order")
	})

	// A dispatch and a write-off move the same numbers, so only the type tells
	// an auditor what happened to the goods.
	t.Run("records itself as a write-off, not a dispatch", func(t *testing.T) {
		productID := newProduct(t, 100, 10)
		_, err := repo.WriteOffStock(ctx, productID, 2, "order_wo_type", "refund_1")
		require.NoError(t, err)

		var typ string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT type FROM inventory_transactions WHERE product_id = $1 AND reference_id = $2`,
			productID, "order_wo_type").Scan(&typ))
		require.Equal(t, "WRITE_OFF", typ)
	})
}
