package postgres

import (
	"context"
	"fmt"
	"sort"
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
)

// existingOrderMovement returns the movement this order already recorded for the
// product, or nil. Reading reference_id back is what makes a repeat a no-op
// rather than a second bite at whatever reserved_qty happens to hold.
func existingOrderMovement(ctx context.Context, tx pgx.Tx, productID, orderID string, typ domain.InventoryTransactionType) (*domain.InventoryTransaction, error) {
	var txn domain.InventoryTransaction
	err := tx.QueryRow(ctx,
		`SELECT id, product_id, type, quantity, previous_qty, new_qty, reason,
		        reference_type, reference_id, created_at, created_by
		 FROM inventory_transactions
		 WHERE product_id = $1 AND reference_id = $2 AND type = $3 AND reference_type = $4`,
		productID, orderID, string(typ), inventoryRefTypeOrder,
	).Scan(&txn.ID, &txn.ProductID, &txn.Type, &txn.Quantity, &txn.PreviousQty,
		&txn.NewQty, &txn.Reason, &txn.ReferenceType, &txn.ReferenceID,
		&txn.CreatedAt, &txn.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read order movement")
	}
	return &txn, nil
}

// applyOrderMovement performs one product's stock movement for an order inside an
// existing transaction. A movement this order already recorded is returned
// unchanged rather than applied twice.
func applyOrderMovement(ctx context.Context, tx pgx.Tx, m orderMovement, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	// Lock before checking the ledger, not after. Two concurrent duplicates
	// would otherwise both read "not yet applied", and the second would reach
	// the insert and trip the unique index — an error where a no-op is correct.
	// Holding the row lock first serializes the check as well as the write.
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

	if existing, err := existingOrderMovement(ctx, tx, productID, orderID, m.typ); err != nil || existing != nil {
		return existing, err
	}

	newQty := currentQty + m.deltaQty*quantity
	newReserved := reservedQty + m.deltaReserve*quantity

	// Reserving needs free stock; releasing and committing need the reservation to
	// exist. The quantity guards also refuse to drive stock negative on a row
	// corrupted by the historical leak, where reserved_qty could exceed quantity.
	if newReserved < 0 || newQty < 0 || newQty < newReserved {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
	}

	now := time.Now()
	updSQL, updArgs := querybuilder.Update("inventory").
		Set(ColQuantity, newQty).
		Set(ColReservedQty, newReserved).
		Set(ColAvailableQty, newQty-newReserved).
		Set(ColUpdatedAt, now).
		Where(ColProductID, productID).
		Build()
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
		Reason:        fmt.Sprintf("ORDER %s", orderID),
		ReferenceType: inventoryRefTypeOrder,
		ReferenceID:   orderID,
		CreatedAt:     now,
	}
	if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

// singleOrderMovement wraps applyOrderMovement in its own transaction.
func (r *InventoryRepository) singleOrderMovement(ctx context.Context, m orderMovement, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		txn, err = applyOrderMovement(ctx, tx, m, productID, quantity, orderID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// ReserveStock reserves stock for an order. Idempotent per (product, order).
func (r *InventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementReserve, productID, quantity, orderID)
}

// ReleaseStock releases stock this order reserved. Idempotent per (product, order).
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementRelease, productID, quantity, orderID)
}

// CommitStock converts this order's reservation into a dispatch. Both quantity and
// reserved_qty drop by the same amount, so available_qty is unchanged: the units
// were already unavailable while reserved and are now physically gone as well.
// Idempotent per (product, order).
func (r *InventoryRepository) CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementCommit, productID, quantity, orderID)
}

// orderMovementAll applies one movement to every line of an order in a single
// transaction, so a line that cannot be applied rolls back the ones before it.
// Products are locked in sorted order: two orders sharing products could
// otherwise grab the same rows in opposite orders and deadlock.
func (r *InventoryRepository) orderMovementAll(ctx context.Context, m orderMovement, orderID string, quantities map[string]int) error {
	if len(quantities) == 0 {
		return nil
	}

	productIDs := make([]string, 0, len(quantities))
	for productID := range quantities {
		productIDs = append(productIDs, productID)
	}
	sort.Strings(productIDs)

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, productID := range productIDs {
			if _, err := applyOrderMovement(ctx, tx, m, productID, quantities[productID], orderID); err != nil {
				return err
			}
		}
		return nil
	})
}

// CommitOrderStock commits every line of an order at once, all or nothing.
func (r *InventoryRepository) CommitOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementCommit, orderID, quantities)
}

// ReleaseOrderStock releases every line of an order at once, all or nothing.
func (r *InventoryRepository) ReleaseOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementRelease, orderID, quantities)
}

// RestockOrderStock returns an order's goods to stock on a return.
//
// Quantities come from the order's COMMIT ledger rows rather than its lines.
// Committing is best-effort, so reaching SHIPPED does not prove the stock ever
// left: adding back a line that never committed inflates quantity and oversells.
// A line with no COMMIT row therefore restocks nothing, which is correct — there
// was no decrement to undo.
//
// Recorded as its own RETURN type rather than through AddStock: a customer
// return is not a supplier replenishment, and AddStock stamps last_restock_at.
func (r *InventoryRepository) RestockOrderStock(ctx context.Context, orderID string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT product_id, quantity FROM inventory_transactions
			 WHERE reference_id = $1 AND reference_type = $2 AND type = $3
			 ORDER BY product_id`,
			orderID, inventoryRefTypeOrder, string(domain.InventoryTransactionTypeCommit),
		)
		if err != nil {
			return errors.Wrap(err, "failed to read committed lines")
		}

		committed := map[string]int{}
		productIDs := []string{}
		for rows.Next() {
			var productID string
			var quantity int
			if err := rows.Scan(&productID, &quantity); err != nil {
				rows.Close()
				return errors.Wrap(err, "failed to scan committed line")
			}
			committed[productID] = quantity
			productIDs = append(productIDs, productID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return errors.Wrap(err, "failed to read committed lines")
		}

		for _, productID := range productIDs {
			if _, err := applyOrderMovement(ctx, tx, movementReturn, productID, committed[productID], orderID); err != nil {
				return err
			}
		}
		return nil
	})
}

// FindOrphanReservations implements domain.InventoryRepository.
//
// The drift signature is a RESERVE with no COMMIT and no RELEASE for the same
// product and order. Everything else settles: a dispatch consumes the
// reservation, a cancel or payment failure gives it back. What is left is stock
// held against an order that did neither, which no reachable transition frees.
func (r *InventoryRepository) FindOrphanReservations(ctx context.Context, minAge time.Duration, limit int) ([]*domain.OrphanReservation, error) {
	const query = `
		SELECT res.product_id,
		       p.name  AS product_name,
		       p.sku   AS sku,
		       res.reference_id AS order_id,
		       res.quantity,
		       res.created_at   AS reserved_at
		FROM inventory_transactions res
		JOIN products p ON p.id = res.product_id
		WHERE res.reference_type = $1
		  AND res.type = $2
		  AND res.created_at < $3
		  AND NOT EXISTS (
		      SELECT 1 FROM inventory_transactions settled
		      WHERE settled.product_id = res.product_id
		        AND settled.reference_id = res.reference_id
		        AND settled.reference_type = $1
		        AND settled.type IN ($4, $5)
		  )
		ORDER BY res.created_at
		LIMIT $6`

	var orphans []*domain.OrphanReservation
	err := pgxscan.Select(ctx, r.pool, &orphans, query,
		inventoryRefTypeOrder,
		string(domain.InventoryTransactionTypeReserve),
		time.Now().Add(-minAge),
		string(domain.InventoryTransactionTypeCommit),
		string(domain.InventoryTransactionTypeRelease),
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
		// id breaks ties: movements a second apart are common, and an unstable
		// order both scrambles the history and can repeat or skip a row across
		// offset-paginated pages.
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
