package postgres

import (
	"context"
	stderrors "errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
	"github.com/handloom/admin/pkg/errors"
)

// inventory_transactions.reference_type values.
const (
	inventoryRefTypeUser  = "USER"
	inventoryRefTypeOrder = "ORDER"
)

// InventoryRepository implements domain.InventoryRepository using PostgreSQL.
type InventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates a new InventoryRepository.
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// Create inserts a new inventory record.
func (r *InventoryRepository) Create(ctx context.Context, inventory *domain.Inventory) error {
	now := time.Now()
	inventory.CreatedAt = now
	inventory.UpdatedAt = now

	qb := querybuilder.Insert("inventory").
		Set(ColID, inventory.ID).
		Set(ColProductID, inventory.ProductID).
		Set(ColQuantity, inventory.Quantity).
		Set(ColReservedQty, inventory.ReservedQty).
		Set(ColAvailableQty, inventory.AvailableQty).
		Set(ColLowStockThreshold, inventory.LowStockThreshold).
		Set(ColReorderPoint, inventory.ReorderPoint).
		Set(ColLastRestockAt, inventory.LastRestockAt).
		Set(ColCreatedAt, inventory.CreatedAt).
		Set(ColUpdatedAt, inventory.UpdatedAt).
		Set(ColCreatedBy, inventory.CreatedBy).
		Set(ColUpdatedBy, inventory.UpdatedBy)

	query, args := qb.Build()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to create inventory")
	}
	return nil
}

// GetByProductID retrieves an inventory record by product ID.
func (r *InventoryRepository) GetByProductID(ctx context.Context, productID string) (*domain.Inventory, error) {
	qb := querybuilder.Select(inventoryColumns...).From("inventory").Where(ColProductID, productID)
	query, args := qb.Build()

	inv := &domain.Inventory{}
	if err := pgxscan.Get(ctx, r.pool, inv, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.NotFound("Inventory not found")
		}
		return nil, errors.Wrap(err, "failed to get inventory")
	}
	return inv, nil
}

// Update updates all mutable fields of an inventory record identified by product_id.
func (r *InventoryRepository) Update(ctx context.Context, inventory *domain.Inventory) error {
	inventory.UpdatedAt = time.Now()

	qb := querybuilder.Update("inventory").
		Set(ColQuantity, inventory.Quantity).
		Set(ColReservedQty, inventory.ReservedQty).
		Set(ColAvailableQty, inventory.AvailableQty).
		Set(ColLowStockThreshold, inventory.LowStockThreshold).
		Set(ColReorderPoint, inventory.ReorderPoint).
		Set(ColLastRestockAt, inventory.LastRestockAt).
		Set(ColUpdatedAt, inventory.UpdatedAt).
		Set(ColUpdatedBy, inventory.UpdatedBy).
		Where(ColProductID, inventory.ProductID)

	query, args := qb.Build()
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to update inventory")
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Inventory not found")
	}
	return nil
}

// AddStock adds stock to inventory within a transaction. It also sets last_restock_at.
func (r *InventoryRepository) AddStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Lock the row.
		var currentQty, reservedQty int
		err := tx.QueryRow(ctx,
			`SELECT quantity, reserved_qty FROM inventory WHERE product_id = $1 FOR UPDATE`,
			productID,
		).Scan(&currentQty, &reservedQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errors.NotFound("Inventory not found")
			}
			return errors.Wrap(err, "failed to lock inventory row")
		}

		now := time.Now()
		previousQty := currentQty
		newQty := currentQty + quantity
		availableQty := newQty - reservedQty

		updQB := querybuilder.Update("inventory").
			Set(ColQuantity, newQty).
			Set(ColAvailableQty, availableQty).
			Set(ColLastRestockAt, now).
			Set(ColUpdatedAt, now).
			Set(ColUpdatedBy, userID).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeAdd,
			Quantity:      quantity,
			PreviousQty:   previousQty,
			NewQty:        newQty,
			Reason:        reason,
			ReferenceType: inventoryRefTypeUser,
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// RemoveStock removes stock from inventory within a transaction.
func (r *InventoryRepository) RemoveStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var currentQty, reservedQty int
		err := tx.QueryRow(ctx,
			`SELECT quantity, reserved_qty FROM inventory WHERE product_id = $1 FOR UPDATE`,
			productID,
		).Scan(&currentQty, &reservedQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errors.NotFound("Inventory not found")
			}
			return errors.Wrap(err, "failed to lock inventory row")
		}

		availableQty := currentQty - reservedQty
		if availableQty < quantity {
			return errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
		}

		now := time.Now()
		previousQty := currentQty
		newQty := currentQty - quantity
		newAvailable := newQty - reservedQty

		updQB := querybuilder.Update("inventory").
			Set(ColQuantity, newQty).
			Set(ColAvailableQty, newAvailable).
			Set(ColUpdatedAt, now).
			Set(ColUpdatedBy, userID).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeRemove,
			Quantity:      quantity,
			PreviousQty:   previousQty,
			NewQty:        newQty,
			Reason:        reason,
			ReferenceType: inventoryRefTypeUser,
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// orderMovement is one product's share of an order-scoped stock movement.
type orderMovement struct {
	typ          domain.InventoryTransactionType
	deltaQty     int // multiplier applied to inventory.quantity
	deltaReserve int // multiplier applied to inventory.reserved_qty
}

var (
	movementReserve = orderMovement{domain.InventoryTransactionTypeReserve, 0, +1}
	movementRelease = orderMovement{domain.InventoryTransactionTypeRelease, 0, -1}
	movementCommit  = orderMovement{domain.InventoryTransactionTypeCommit, -1, -1}
	movementReturn  = orderMovement{domain.InventoryTransactionTypeReturn, +1, 0}

	// Arithmetically a dispatch — both counters fall, available_qty holds — but
	// the ledger has to say the goods were written off, not shipped.
	movementWriteOff = orderMovement{domain.InventoryTransactionTypeWriteOff, -1, -1}
)

// existingOrderMovement returns the movement this order already recorded for the
// product and source, or nil. Reading it back is what makes a repeat a no-op.
func existingOrderMovement(ctx context.Context, tx pgx.Tx, productID, orderID, sourceID string, typ domain.InventoryTransactionType) (*domain.InventoryTransaction, error) {
	var txn domain.InventoryTransaction
	err := tx.QueryRow(ctx,
		`SELECT id, product_id, type, quantity, previous_qty, new_qty, reason,
		        reference_type, reference_id, source_id, created_at, created_by
		 FROM inventory_transactions
		 WHERE product_id = $1 AND reference_id = $2 AND type = $3 AND reference_type = $4
		   AND source_id = $5`,
		productID, orderID, string(typ), inventoryRefTypeOrder, sourceID,
	).Scan(&txn.ID, &txn.ProductID, &txn.Type, &txn.Quantity, &txn.PreviousQty,
		&txn.NewQty, &txn.Reason, &txn.ReferenceType, &txn.ReferenceID, &txn.SourceID,
		&txn.CreatedAt, &txn.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read order movement")
	}
	return &txn, nil
}

// movementReason is the ledger note. It names the source when one caused the
// movement, so two write-offs on an order are told apart by more than their order.
func movementReason(orderID, sourceID string) string {
	if sourceID == "" {
		return fmt.Sprintf("ORDER %s", orderID)
	}
	return fmt.Sprintf("ORDER %s (%s)", orderID, sourceID)
}

// orderOutstandingReservation is how much of a product this order still holds
// reserved: what it reserved, less everything since settled or refunded away.
//
// known is false when the ledger holds no reservation for this order at all.
// reserved_qty can also be set through UpdateInventory, which writes no ledger row,
// and refusing to release stock the ledger cannot explain would strand it.
func orderOutstandingReservation(ctx context.Context, tx pgx.Tx, productID, orderID string) (outstanding int, known bool, err error) {
	var reservedRows int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE
		          WHEN type = $3 THEN quantity
		          WHEN type IN ($4, $5, $6) THEN -quantity
		          ELSE 0 END), 0),
		        COUNT(*) FILTER (WHERE type = $3)
		 FROM inventory_transactions
		 WHERE product_id = $1 AND reference_id = $2 AND reference_type = $7`,
		productID, orderID,
		string(domain.InventoryTransactionTypeReserve),
		string(domain.InventoryTransactionTypeRelease),
		string(domain.InventoryTransactionTypeCommit),
		string(domain.InventoryTransactionTypeWriteOff),
		inventoryRefTypeOrder,
	).Scan(&outstanding, &reservedRows)
	if err != nil {
		return 0, false, errors.Wrap(err, "failed to read the order's outstanding reservation")
	}
	if outstanding < 0 {
		outstanding = 0
	}
	return outstanding, reservedRows > 0, nil
}

// applyOrderMovement performs one product's stock movement inside an existing
// transaction. A movement already recorded for this order and source is returned
// unchanged. sourceID names a refund; createdBy names an admin, and they are not
// the same thing — a refund has no actor on the movement it causes.
func applyOrderMovement(ctx context.Context, tx pgx.Tx, m orderMovement, productID string, quantity int, orderID, sourceID, createdBy string) (*domain.InventoryTransaction, error) {
	// Lock before the ledger check, not after: concurrent duplicates would both
	// read "not yet applied" and the second would trip the unique index.
	var currentQty, reservedQty int
	err := tx.QueryRow(ctx,
		`SELECT quantity, reserved_qty FROM inventory WHERE product_id = $1 FOR UPDATE`,
		productID,
	).Scan(&currentQty, &reservedQty)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Inventory not found")
		}
		return nil, errors.Wrap(err, "failed to lock inventory row")
	}

	existing, err := existingOrderMovement(ctx, tx, productID, orderID, sourceID, m.typ)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// A replay of the same amount is the no-op the unique index exists for. A
		// different amount is divergence, and silently dropping it loses stock.
		if existing.Quantity != quantity {
			return nil, errors.New(errors.ErrCodeConflict,
				fmt.Sprintf("order %s already recorded %s of %d for product %s, cannot apply %d",
					orderID, m.typ, existing.Quantity, productID, quantity))
		}
		return existing, nil
	}

	// An order can only give back what it still holds. reserved_qty is a product-wide
	// total, so sizing a release from the order's lines takes the difference out of
	// another order's reservation once a refund has written part of this one off.
	if m.deltaReserve < 0 {
		outstanding, known, boundErr := orderOutstandingReservation(ctx, tx, productID, orderID)
		if boundErr != nil {
			return nil, boundErr
		}
		switch {
		case !known:
			// Nothing in the ledger to bound against; the guards below are the check.
		case sourceID == "":
			// A cancel or a dispatch settles whatever the order has left, and the caller
			// cannot know that figure — it passes the ordered quantity.
			if quantity > outstanding {
				quantity = outstanding
			}
			if quantity == 0 {
				return nil, nil
			}
		case quantity > outstanding:
			// A refund naming more units than the order holds is a caller error, not a
			// remainder to settle. Taking a smaller bite would hide it.
			return nil, errors.New(errors.ErrCodeInsufficientStock,
				"refund exceeds the order's outstanding reservation")
		}
	}

	newQty := currentQty + m.deltaQty*quantity
	newReserved := reservedQty + m.deltaReserve*quantity

	// Neither counter may go negative: release and commit need the reservation to exist.
	if newReserved < 0 || newQty < 0 {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
	}

	// Only a reservation has to fit inside free stock. Release and commit reduce
	// reserved_qty, so they must stay possible on rows the historical leak corrupted.
	if m.deltaReserve > 0 && newQty < newReserved {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
	}

	now := time.Now()
	upd := querybuilder.Update("inventory").
		Set(ColQuantity, newQty).
		Set(ColReservedQty, newReserved).
		Set(ColAvailableQty, newQty-newReserved).
		Set(ColUpdatedAt, now)
	// Only when we know the actor: reserve, release and commit are system-driven and
	// must not overwrite whoever AddStock or AdjustStock last recorded.
	if createdBy != "" {
		upd = upd.Set(ColUpdatedBy, createdBy)
	}
	updSQL, updArgs := upd.Where(ColProductID, productID).Build()
	if _, err := tx.Exec(ctx, updSQL, updArgs...); err != nil {
		return nil, errors.Wrap(err, "failed to update inventory")
	}

	// PreviousQty/NewQty track whichever counter the movement is about:
	// reserved_qty for RESERVE/RELEASE, quantity for COMMIT.
	prev, next := reservedQty, newReserved
	if m.deltaQty != 0 {
		prev, next = currentQty, newQty
	}

	txn := &domain.InventoryTransaction{
		ID:            "inv_txn_" + uuid.New().String()[:8],
		ProductID:     productID,
		Type:          m.typ,
		Quantity:      quantity,
		PreviousQty:   prev,
		NewQty:        next,
		Reason:        movementReason(orderID, sourceID),
		ReferenceType: inventoryRefTypeOrder,
		ReferenceID:   orderID,
		SourceID:      sourceID,
		CreatedAt:     now,
		CreatedBy:     createdBy,
	}
	if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

// singleOrderMovement wraps applyOrderMovement in its own transaction. No actor:
// reserve, release and commit are system-driven; only a return carries one.
func (r *InventoryRepository) singleOrderMovement(ctx context.Context, m orderMovement, productID string, quantity int, orderID, sourceID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		txn, err = applyOrderMovement(ctx, tx, m, productID, quantity, orderID, sourceID, "")
		return err
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// ReserveStock reserves stock for an order. Idempotent per (product, order).
func (r *InventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementReserve, productID, quantity, orderID, "")
}

// ReleaseStock releases stock this order reserved. Idempotent per (product, order),
// so it belongs to the order lifecycle however many refunds preceded it.
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementRelease, productID, quantity, orderID, "")
}

// ReleaseRefundedStock returns a refunded line to sale: only the reservation moves.
// Scoped to the refund, so a second refund of the same product is not a replay.
func (r *InventoryRepository) ReleaseRefundedStock(ctx context.Context, productID string, quantity int, orderID, refundID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementRelease, productID, quantity, orderID, refundID)
}

// CommitStock turns this order's reservation into a dispatch: quantity and
// reserved_qty both drop, available_qty unchanged. Idempotent per (product, order).
func (r *InventoryRepository) CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementCommit, productID, quantity, orderID, "")
}

// orderMovementAll applies one movement to every line of an order in a single
// transaction, so a line that cannot be applied rolls back the ones before it.
func (r *InventoryRepository) orderMovementAll(ctx context.Context, m orderMovement, orderID string, quantities map[string]int) error {
	if len(quantities) == 0 {
		return nil
	}

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Sorted, not map order: two orders sharing products would otherwise take
		// the FOR UPDATE row locks in opposite sequences and deadlock.
		for _, productID := range slices.Sorted(maps.Keys(quantities)) {
			if _, err := applyOrderMovement(ctx, tx, m, productID, quantities[productID], orderID, "", ""); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReserveOrderStock reserves every line of an order at once, all or nothing.
// Aggregated first: one product on two lines would otherwise dedup into one.
func (r *InventoryRepository) ReserveOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementReserve, orderID, quantities)
}

// CommitOrderStock commits every line of an order at once, all or nothing.
func (r *InventoryRepository) CommitOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementCommit, orderID, quantities)
}

// ReleaseOrderStock is best-effort, unlike commit: every caller is a rollback, so
// one unreleasable line must not strand what the other lines still hold.
func (r *InventoryRepository) ReleaseOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	var failures []error
	for _, productID := range slices.Sorted(maps.Keys(quantities)) {
		if _, err := r.singleOrderMovement(ctx, movementRelease, productID, quantities[productID], orderID, ""); err != nil {
			failures = append(failures, fmt.Errorf("product %s: %w", productID, err))
		}
	}
	// Every failure, not just the first: these are the only leak signal an operator
	// gets, and they need to know which products to reconcile.
	return stderrors.Join(failures...)
}

// WriteOffStock drops the reservation and on-hand together for a refunded line whose
// goods are gone. Scoped to the refund, so a second refund is not read as a replay.
func (r *InventoryRepository) WriteOffStock(ctx context.Context, productID string, quantity int, orderID, refundID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementWriteOff, productID, quantity, orderID, refundID)
}

// ledgerLine is one (product_id, quantity) ledger row, in query order.
type ledgerLine struct {
	productID string
	quantity  int
}

// scanLedgerQuantities reads ledger rows in the order the query returned them.
func scanLedgerQuantities(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]ledgerLine, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []ledgerLine
	for rows.Next() {
		var l ledgerLine
		if scanErr := rows.Scan(&l.productID, &l.quantity); scanErr != nil {
			return nil, scanErr
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// RestockOrderStock returns an order's goods on a return, quantities read from its
// COMMIT ledger rows. Its own RETURN type so last_restock_at is not stamped.
func (r *InventoryRepository) RestockOrderStock(ctx context.Context, orderID, createdBy string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Lock the order's inventory rows before reading the ledger, so a COMMIT row
		// inserted concurrently cannot be missed and left unrestocked.
		if _, err := tx.Exec(ctx,
			`SELECT product_id FROM inventory
			 WHERE product_id IN (
			   SELECT product_id FROM inventory_transactions
			   WHERE reference_id = $1 AND reference_type = $2
			 )
			 ORDER BY product_id
			 FOR UPDATE`,
			orderID, inventoryRefTypeOrder,
		); err != nil {
			return errors.Wrap(err, "failed to lock order inventory")
		}

		// Not joined to inventory: a COMMIT row whose inventory row was deleted must
		// still reach applyOrderMovement and fail there, not vanish from the result.
		committed, readErr := scanLedgerQuantities(ctx, tx,
			`SELECT product_id, quantity FROM inventory_transactions
			 WHERE reference_id = $1 AND reference_type = $2 AND type = $3
			 ORDER BY product_id`,
			orderID, inventoryRefTypeOrder, string(domain.InventoryTransactionTypeCommit))
		if readErr != nil {
			return errors.Wrap(readErr, "failed to read committed lines")
		}

		for _, line := range committed {
			if _, err := applyOrderMovement(ctx, tx, movementReturn, line.productID, line.quantity, orderID, "", createdBy); err != nil {
				return err
			}
		}

		// Reserved, never committed, not already released: still holding, and a returned
		// order can no longer reach CANCELLED to free it. Already-released lines are not.
		stranded, strandedErr := scanLedgerQuantities(ctx, tx,
			`SELECT t.product_id, t.quantity
			 FROM inventory_transactions t
			 WHERE t.reference_id = $1 AND t.reference_type = $2 AND t.type = $3
			   AND NOT EXISTS (
			     SELECT 1 FROM inventory_transactions c
			     WHERE c.reference_id = t.reference_id AND c.reference_type = t.reference_type
			       AND c.product_id = t.product_id AND c.type IN ($4, $5)
			   )
			 ORDER BY t.product_id`,
			orderID, inventoryRefTypeOrder,
			string(domain.InventoryTransactionTypeReserve),
			string(domain.InventoryTransactionTypeCommit),
			string(domain.InventoryTransactionTypeRelease))
		if strandedErr != nil {
			return errors.Wrap(strandedErr, "failed to read stranded reservations")
		}

		for _, line := range stranded {
			if _, err := applyOrderMovement(ctx, tx, movementRelease, line.productID, line.quantity, orderID, "", createdBy); err != nil {
				return err
			}
		}
		return nil
	})
}

// FindOrphanReservations implements domain.InventoryRepository. The signature is a
// RESERVE with no COMMIT and no RELEASE for the same product and order.
func (r *InventoryRepository) FindOrphanReservations(ctx context.Context, minAge time.Duration, limit int) ([]*domain.OrphanReservation, error) {
	// Netted, not tested for presence: a reservation that settled partly is still
	// partly stranded, and asking only whether any settlement exists hides exactly
	// that. The quantity reported is what is still held, not what was reserved.
	const query = `
		SELECT t.product_id,
		       p.name AS product_name,
		       p.sku  AS sku,
		       t.reference_id AS order_id,
		       t.outstanding  AS quantity,
		       t.reserved_at
		FROM (
		    SELECT product_id,
		           reference_id,
		           SUM(CASE WHEN type = $2 THEN quantity
		                    WHEN type IN ($4, $5, $6) THEN -quantity
		                    ELSE 0 END) AS outstanding,
		           MIN(created_at) FILTER (WHERE type = $2) AS reserved_at
		    FROM inventory_transactions
		    WHERE reference_type = $1
		    GROUP BY product_id, reference_id
		) t
		JOIN products p ON p.id = t.product_id
		WHERE t.outstanding > 0
		  AND t.reserved_at < $3
		ORDER BY t.reserved_at
		LIMIT $7`

	var orphans []*domain.OrphanReservation
	err := pgxscan.Select(ctx, r.pool, &orphans, query,
		inventoryRefTypeOrder,
		string(domain.InventoryTransactionTypeReserve),
		time.Now().Add(-minAge),
		string(domain.InventoryTransactionTypeRelease),
		string(domain.InventoryTransactionTypeCommit),
		// A write-off settles a reservation too: the units are gone, not stranded.
		string(domain.InventoryTransactionTypeWriteOff),
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find orphan reservations")
	}

	return orphans, nil
}

// AdjustStock sets the inventory quantity to an absolute value within a transaction.
func (r *InventoryRepository) AdjustStock(ctx context.Context, productID string, newQuantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var currentQty, reservedQty int
		err := tx.QueryRow(ctx,
			`SELECT quantity, reserved_qty FROM inventory WHERE product_id = $1 FOR UPDATE`,
			productID,
		).Scan(&currentQty, &reservedQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errors.NotFound("Inventory not found")
			}
			return errors.Wrap(err, "failed to lock inventory row")
		}

		now := time.Now()
		availableQty := newQuantity - reservedQty

		updQB := querybuilder.Update("inventory").
			Set(ColQuantity, newQuantity).
			Set(ColAvailableQty, availableQty).
			Set(ColUpdatedAt, now).
			Set(ColUpdatedBy, userID).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		delta := newQuantity - currentQty
		absQty := delta
		if absQty < 0 {
			absQty = -absQty
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeAdjust,
			Quantity:      absQty,
			PreviousQty:   currentQty,
			NewQty:        newQuantity,
			Reason:        reason,
			ReferenceType: inventoryRefTypeUser,
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// insertInventoryTransaction inserts a single inventory_transactions row.
func insertInventoryTransaction(ctx context.Context, tx pgx.Tx, txn *domain.InventoryTransaction) error {
	qb := querybuilder.Insert("inventory_transactions").
		Set(ColID, txn.ID).
		Set(ColProductID, txn.ProductID).
		Set(ColType, string(txn.Type)).
		Set(ColQuantity, txn.Quantity).
		Set(ColPreviousQty, txn.PreviousQty).
		Set(ColNewQty, txn.NewQty).
		Set(ColReason, txn.Reason).
		Set(ColReferenceType, txn.ReferenceType).
		Set(ColReferenceID, txn.ReferenceID).
		Set(ColSourceID, txn.SourceID).
		Set(ColCreatedAt, txn.CreatedAt).
		Set(ColCreatedBy, txn.CreatedBy)

	query, args := qb.Build()
	_, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to create inventory transaction")
	}
	return nil
}

// GetTransactions retrieves inventory transactions for a product, ordered newest first.
func (r *InventoryRepository) GetTransactions(ctx context.Context, productID string, pagination domain.PaginationRequest) (*domain.ListInventoryTransactionsResponse, error) {
	limit, offset := pageParams(pagination)

	qb := querybuilder.Select(inventoryTxnColumns...).
		From("inventory_transactions").
		Where(ColProductID, productID).
		// id breaks ties: same-second movements are common, and an unstable order
		// repeats or skips rows across pages.
		OrderBy(ColCreatedAt + " DESC, " + ColID + " DESC").
		Limit(limit + 1).
		Offset(offset)

	query, args := qb.Build()

	var transactions []*domain.InventoryTransaction
	if err := pgxscan.Select(ctx, r.pool, &transactions, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to query inventory transactions")
	}

	pg := buildPaginationResponse(limit, offset, len(transactions))

	// Trim the extra probe row before returning.
	if len(transactions) > limit {
		transactions = transactions[:limit]
	}

	return &domain.ListInventoryTransactionsResponse{
		Transactions: transactions,
		Pagination:   pg,
	}, nil
}

// GetLowStockProducts retrieves inventory records where available_qty <= low_stock_threshold.
func (r *InventoryRepository) GetLowStockProducts(ctx context.Context, pagination domain.PaginationRequest) (*domain.ListInventoryResponse, error) {
	limit, offset := pageParams(pagination)

	qb := querybuilder.Select(
		"i.id", "i.product_id", "p.sku", "p.name",
		"i.quantity", "i.reserved_qty", "i.available_qty",
		"i.low_stock_threshold", "i.reorder_point", "i.last_restock_at",
		"i.created_at", "i.updated_at", "i.created_by", "i.updated_by",
	).
		From("inventory i").
		LeftJoin("products p", "p.id = i.product_id").
		WithRaw(true, "i.available_qty <= i.low_stock_threshold AND i.low_stock_threshold > 0").
		OrderBy("i.available_qty ASC").
		Limit(limit + 1).
		Offset(offset)

	query, args := qb.Build()
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query low stock inventory")
	}
	defer rows.Close()

	var inventories []*domain.Inventory
	for rows.Next() {
		inv := &domain.Inventory{}
		if err := rows.Scan(
			&inv.ID, &inv.ProductID, &inv.ProductSKU, &inv.ProductName,
			&inv.Quantity, &inv.ReservedQty, &inv.AvailableQty,
			&inv.LowStockThreshold, &inv.ReorderPoint, &inv.LastRestockAt,
			&inv.CreatedAt, &inv.UpdatedAt, &inv.CreatedBy, &inv.UpdatedBy,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan inventory")
		}
		inventories = append(inventories, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate inventory rows")
	}

	pg := buildPaginationResponse(limit, offset, len(inventories))

	if len(inventories) > limit {
		inventories = inventories[:limit]
	}

	return &domain.ListInventoryResponse{
		Inventories: inventories,
		Pagination:  pg,
	}, nil
}

// DeleteByProductID deletes the inventory record for a product.
// The ON DELETE CASCADE constraint on inventory_transactions handles transaction cleanup.
func (r *InventoryRepository) DeleteByProductID(ctx context.Context, productID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM inventory WHERE product_id = $1`, productID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to delete inventory")
	}
	return nil
}

// Ensure interface compliance at compile time.
var _ domain.InventoryRepository = (*InventoryRepository)(nil)
