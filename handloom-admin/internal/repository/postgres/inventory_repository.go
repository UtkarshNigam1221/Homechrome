package postgres

import (
	"context"
	"fmt"
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

// ReserveStock reserves stock for an order within a transaction.
func (r *InventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
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
		newReserved := reservedQty + quantity
		newAvailable := currentQty - newReserved

		updQB := querybuilder.Update("inventory").
			Set(ColReservedQty, newReserved).
			Set(ColAvailableQty, newAvailable).
			Set(ColUpdatedAt, now).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeReserve,
			Quantity:      quantity,
			PreviousQty:   reservedQty,
			NewQty:        newReserved,
			Reason:        fmt.Sprintf("ORDER %s", orderID),
			ReferenceType: inventoryRefTypeOrder,
			ReferenceID:   orderID,
			CreatedAt:     now,
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

// ReleaseStock releases previously reserved stock within a transaction.
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
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

		if reservedQty < quantity {
			return errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
		}

		now := time.Now()
		newReserved := reservedQty - quantity
		newAvailable := currentQty - newReserved

		updQB := querybuilder.Update("inventory").
			Set(ColReservedQty, newReserved).
			Set(ColAvailableQty, newAvailable).
			Set(ColUpdatedAt, now).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeRelease,
			Quantity:      quantity,
			PreviousQty:   reservedQty,
			NewQty:        newReserved,
			Reason:        fmt.Sprintf("ORDER %s", orderID),
			ReferenceType: inventoryRefTypeOrder,
			ReferenceID:   orderID,
			CreatedAt:     now,
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

// CommitStock converts a reservation into a dispatch within a transaction.
// Both quantity and reserved_qty drop by the same amount, so available_qty is
// unchanged: the units were already unavailable when reserved, and are now
// physically gone as well.
func (r *InventoryRepository) CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
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

		// Both guards matter. reservedQty is the real invariant, but a row
		// corrupted by the historical leak can violate reservedQty <= quantity,
		// and driving quantity negative would be worse than refusing.
		if reservedQty < quantity || currentQty < quantity {
			return errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
		}

		now := time.Now()
		newQty := currentQty - quantity
		newReserved := reservedQty - quantity
		newAvailable := newQty - newReserved

		updQB := querybuilder.Update("inventory").
			Set(ColQuantity, newQty).
			Set(ColReservedQty, newReserved).
			Set(ColAvailableQty, newAvailable).
			Set(ColUpdatedAt, now).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeCommit,
			Quantity:      quantity,
			PreviousQty:   currentQty,
			NewQty:        newQty,
			Reason:        fmt.Sprintf("ORDER %s", orderID),
			ReferenceType: inventoryRefTypeOrder,
			ReferenceID:   orderID,
			CreatedAt:     now,
		}

		return insertInventoryTransaction(ctx, tx, txn)
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
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
