package postgres

import (
	"context"
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
)

// existingOrderMovement returns this order's already-recorded movement for the
// product, or nil. Reading reference_id back is what makes a repeat a no-op.
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

// applyOrderMovement performs one product's stock movement inside an existing
// transaction. A movement already recorded for this order is returned unchanged.
func applyOrderMovement(ctx context.Context, tx pgx.Tx, m orderMovement, productID string, quantity int, orderID, createdBy string) (*domain.InventoryTransaction, error) {
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

	existing, err := existingOrderMovement(ctx, tx, productID, orderID, m.typ)
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
	updSQL, updArgs := querybuilder.Update("inventory").
		Set(ColQuantity, newQty).
		Set(ColReservedQty, newReserved).
		Set(ColAvailableQty, newQty-newReserved).
		Set(ColUpdatedAt, now).
		Set(ColUpdatedBy, createdBy).
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
		CreatedBy:     createdBy,
	}
	if err := insertInventoryTransaction(ctx, tx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

// singleOrderMovement wraps applyOrderMovement in its own transaction.
func (r *InventoryRepository) singleOrderMovement(ctx context.Context, m orderMovement, productID string, quantity int, orderID, createdBy string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		txn, err = applyOrderMovement(ctx, tx, m, productID, quantity, orderID, createdBy)
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

// ReleaseStock releases stock this order reserved. Idempotent per (product, order).
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementRelease, productID, quantity, orderID, "")
}

// CommitStock turns this order's reservation into a dispatch: quantity and
// reserved_qty both drop, available_qty unchanged. Idempotent per (product, order).
func (r *InventoryRepository) CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.singleOrderMovement(ctx, movementCommit, productID, quantity, orderID, "")
}

// orderMovementAll applies one movement to every line of an order in a single
// transaction, so a line that cannot be applied rolls back the ones before it.
func (r *InventoryRepository) orderMovementAll(ctx context.Context, m orderMovement, orderID, createdBy string, quantities map[string]int) error {
	if len(quantities) == 0 {
		return nil
	}

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Sorted, not map order: two orders sharing products would otherwise take
		// the FOR UPDATE row locks in opposite sequences and deadlock.
		for _, productID := range slices.Sorted(maps.Keys(quantities)) {
			if _, err := applyOrderMovement(ctx, tx, m, productID, quantities[productID], orderID, createdBy); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReserveOrderStock reserves every line of an order at once, all or nothing.
// Aggregated first: one product on two lines would otherwise dedup into one.
func (r *InventoryRepository) ReserveOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementReserve, orderID, "", quantities)
}

// CommitOrderStock commits every line of an order at once, all or nothing.
func (r *InventoryRepository) CommitOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	return r.orderMovementAll(ctx, movementCommit, orderID, "", quantities)
}

// ReleaseOrderStock is best-effort, unlike commit: every caller is a rollback, so
// one unreleasable line must not strand what the other lines still hold.
func (r *InventoryRepository) ReleaseOrderStock(ctx context.Context, orderID string, quantities map[string]int) error {
	var firstErr error
	for _, productID := range slices.Sorted(maps.Keys(quantities)) {
		if _, err := r.singleOrderMovement(ctx, movementRelease, productID, quantities[productID], orderID, ""); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// RestockOrderStock returns an order's goods on a return, quantities read from its
// COMMIT ledger rows. Its own RETURN type so last_restock_at is not stamped.
func (r *InventoryRepository) RestockOrderStock(ctx context.Context, orderID, createdBy string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// FOR UPDATE OF i takes the inventory row locks while the ledger is read, so a
		// COMMIT row inserted concurrently cannot be missed and left unrestocked.
		rows, err := tx.Query(ctx,
			`SELECT t.product_id, t.quantity
			 FROM inventory_transactions t
			 JOIN inventory i ON i.product_id = t.product_id
			 WHERE t.reference_id = $1 AND t.reference_type = $2 AND t.type = $3
			 ORDER BY t.product_id
			 FOR UPDATE OF i`,
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
			if _, err := applyOrderMovement(ctx, tx, movementReturn, productID, committed[productID], orderID, createdBy); err != nil {
				return err
			}
		}

		// A line that reserved but never committed still holds its reservation, and a
		// returned order can no longer reach CANCELLED to free it. Release it here.
		stranded, err := tx.Query(ctx,
			`SELECT t.product_id, t.quantity
			 FROM inventory_transactions t
			 JOIN inventory i ON i.product_id = t.product_id
			 WHERE t.reference_id = $1 AND t.reference_type = $2 AND t.type = $3
			   AND NOT EXISTS (
			     SELECT 1 FROM inventory_transactions c
			     WHERE c.reference_id = t.reference_id AND c.reference_type = t.reference_type
			       AND c.product_id = t.product_id AND c.type = $4
			   )
			 ORDER BY t.product_id
			 FOR UPDATE OF i`,
			orderID, inventoryRefTypeOrder,
			string(domain.InventoryTransactionTypeReserve), string(domain.InventoryTransactionTypeCommit),
		)
		if err != nil {
			return errors.Wrap(err, "failed to read stranded reservations")
		}
		reserved := map[string]int{}
		strandedIDs := []string{}
		for stranded.Next() {
			var productID string
			var quantity int
			if err := stranded.Scan(&productID, &quantity); err != nil {
				stranded.Close()
				return errors.Wrap(err, "failed to scan stranded reservation")
			}
			reserved[productID] = quantity
			strandedIDs = append(strandedIDs, productID)
		}
		stranded.Close()
		if err := stranded.Err(); err != nil {
			return errors.Wrap(err, "failed to read stranded reservations")
		}

		for _, productID := range strandedIDs {
			if _, err := applyOrderMovement(ctx, tx, movementRelease, productID, reserved[productID], orderID, createdBy); err != nil {
				return err
			}
		}
		return nil
	})
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
		OrderBy(ColCreatedAt + " DESC").
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
