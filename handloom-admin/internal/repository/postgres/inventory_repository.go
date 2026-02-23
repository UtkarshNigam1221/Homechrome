package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
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

	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory (
			id, product_id, product_sku, product_name,
			quantity, reserved_qty, available_qty,
			low_stock_threshold, reorder_point, last_restock_at,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		inventory.ID, inventory.ProductID, inventory.ProductSKU, inventory.ProductName,
		inventory.Quantity, inventory.ReservedQty, inventory.AvailableQty,
		inventory.LowStockThreshold, inventory.ReorderPoint, inventory.LastRestockAt,
		inventory.CreatedAt, inventory.UpdatedAt, inventory.CreatedBy, inventory.UpdatedBy,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create inventory")
	}
	return nil
}

// GetByProductID retrieves an inventory record by product ID.
func (r *InventoryRepository) GetByProductID(ctx context.Context, productID string) (*domain.Inventory, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, product_id, product_sku, product_name,
			quantity, reserved_qty, available_qty,
			low_stock_threshold, reorder_point, last_restock_at,
			created_at, updated_at, created_by, updated_by
		FROM inventory WHERE product_id = $1`, productID)

	inv := &domain.Inventory{}
	err := row.Scan(
		&inv.ID, &inv.ProductID, &inv.ProductSKU, &inv.ProductName,
		&inv.Quantity, &inv.ReservedQty, &inv.AvailableQty,
		&inv.LowStockThreshold, &inv.ReorderPoint, &inv.LastRestockAt,
		&inv.CreatedAt, &inv.UpdatedAt, &inv.CreatedBy, &inv.UpdatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Inventory not found")
		}
		return nil, errors.Wrap(err, "failed to get inventory")
	}
	return inv, nil
}

// Update updates all mutable fields of an inventory record identified by product_id.
func (r *InventoryRepository) Update(ctx context.Context, inventory *domain.Inventory) error {
	inventory.UpdatedAt = time.Now()

	tag, err := r.pool.Exec(ctx,
		`UPDATE inventory SET
			product_sku = $1, product_name = $2,
			quantity = $3, reserved_qty = $4, available_qty = $5,
			low_stock_threshold = $6, reorder_point = $7, last_restock_at = $8,
			updated_at = $9, updated_by = $10
		WHERE product_id = $11`,
		inventory.ProductSKU, inventory.ProductName,
		inventory.Quantity, inventory.ReservedQty, inventory.AvailableQty,
		inventory.LowStockThreshold, inventory.ReorderPoint, inventory.LastRestockAt,
		inventory.UpdatedAt, inventory.UpdatedBy,
		inventory.ProductID,
	)
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

		_, err = tx.Exec(ctx,
			`UPDATE inventory SET quantity = $1, available_qty = $2, last_restock_at = $3, updated_at = $4, updated_by = $5
			WHERE product_id = $6`,
			newQty, availableQty, now, now, userID, productID,
		)
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
			ReferenceType: "USER",
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO inventory_transactions (id, product_id, type, quantity, previous_qty, new_qty, reason, reference_type, reference_id, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			txn.ID, txn.ProductID, string(txn.Type), txn.Quantity, txn.PreviousQty, txn.NewQty,
			txn.Reason, txn.ReferenceType, txn.ReferenceID, txn.CreatedAt, txn.CreatedBy,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create inventory transaction")
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

		_, err = tx.Exec(ctx,
			`UPDATE inventory SET quantity = $1, available_qty = $2, updated_at = $3, updated_by = $4
			WHERE product_id = $5`,
			newQty, newAvailable, now, userID, productID,
		)
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
			ReferenceType: "USER",
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO inventory_transactions (id, product_id, type, quantity, previous_qty, new_qty, reason, reference_type, reference_id, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			txn.ID, txn.ProductID, string(txn.Type), txn.Quantity, txn.PreviousQty, txn.NewQty,
			txn.Reason, txn.ReferenceType, txn.ReferenceID, txn.CreatedAt, txn.CreatedBy,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create inventory transaction")
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

		_, err = tx.Exec(ctx,
			`UPDATE inventory SET reserved_qty = $1, available_qty = $2, updated_at = $3
			WHERE product_id = $4`,
			newReserved, newAvailable, now, productID,
		)
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
			ReferenceType: "ORDER",
			ReferenceID:   orderID,
			CreatedAt:     now,
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO inventory_transactions (id, product_id, type, quantity, previous_qty, new_qty, reason, reference_type, reference_id, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			txn.ID, txn.ProductID, string(txn.Type), txn.Quantity, txn.PreviousQty, txn.NewQty,
			txn.Reason, txn.ReferenceType, txn.ReferenceID, txn.CreatedAt, txn.CreatedBy,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create inventory transaction")
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

		_, err = tx.Exec(ctx,
			`UPDATE inventory SET reserved_qty = $1, available_qty = $2, updated_at = $3
			WHERE product_id = $4`,
			newReserved, newAvailable, now, productID,
		)
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
			ReferenceType: "ORDER",
			ReferenceID:   orderID,
			CreatedAt:     now,
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO inventory_transactions (id, product_id, type, quantity, previous_qty, new_qty, reason, reference_type, reference_id, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			txn.ID, txn.ProductID, string(txn.Type), txn.Quantity, txn.PreviousQty, txn.NewQty,
			txn.Reason, txn.ReferenceType, txn.ReferenceID, txn.CreatedAt, txn.CreatedBy,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create inventory transaction")
		}

		return nil
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

		_, err = tx.Exec(ctx,
			`UPDATE inventory SET quantity = $1, available_qty = $2, updated_at = $3, updated_by = $4
			WHERE product_id = $5`,
			newQuantity, availableQty, now, userID, productID,
		)
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
			ReferenceType: "USER",
			ReferenceID:   "",
			CreatedAt:     now,
			CreatedBy:     userID,
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO inventory_transactions (id, product_id, type, quantity, previous_qty, new_qty, reason, reference_type, reference_id, created_at, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			txn.ID, txn.ProductID, string(txn.Type), txn.Quantity, txn.PreviousQty, txn.NewQty,
			txn.Reason, txn.ReferenceType, txn.ReferenceID, txn.CreatedAt, txn.CreatedBy,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create inventory transaction")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// GetTransactions retrieves inventory transactions for a product, ordered newest first.
func (r *InventoryRepository) GetTransactions(ctx context.Context, productID string, pagination domain.PaginationRequest) (*domain.ListInventoryTransactionsResponse, error) {
	limit, offset := pageParams(pagination)

	// Fetch one extra row to determine HasMore.
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, type, quantity, previous_qty, new_qty,
			reason, reference_type, reference_id, created_at, created_by
		FROM inventory_transactions
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		productID, limit+1, offset,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query inventory transactions")
	}
	defer rows.Close()

	var transactions []*domain.InventoryTransaction
	for rows.Next() {
		t := &domain.InventoryTransaction{}
		if err := rows.Scan(
			&t.ID, &t.ProductID, &t.Type, &t.Quantity, &t.PreviousQty, &t.NewQty,
			&t.Reason, &t.ReferenceType, &t.ReferenceID, &t.CreatedAt, &t.CreatedBy,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan inventory transaction")
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate inventory transactions")
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

	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, product_sku, product_name,
			quantity, reserved_qty, available_qty,
			low_stock_threshold, reorder_point, last_restock_at,
			created_at, updated_at, created_by, updated_by
		FROM inventory
		WHERE available_qty <= low_stock_threshold AND low_stock_threshold > 0
		ORDER BY available_qty ASC
		LIMIT $1 OFFSET $2`,
		limit+1, offset,
	)
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
