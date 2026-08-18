// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/metrics"
)

// InventoryService implements domain.InventoryService
type InventoryService struct {
	inventoryRepo domain.InventoryRepository
	userRepo      domain.UserRepository
}

// NewInventoryService creates a new InventoryService. userRepo may be nil: the
// storefront builds this service for stock levels alone and never reads the
// ledger, so it wires no user directory rather than pulling an admin repository
// into a customer-facing Lambda.
func NewInventoryService(
	inventoryRepo domain.InventoryRepository,
	userRepo domain.UserRepository,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

// GetByProductID retrieves inventory for a product
func (s *InventoryService) GetByProductID(ctx context.Context, productID string) (*domain.Inventory, error) {
	return s.inventoryRepo.GetByProductID(ctx, productID)
}

// AddStock adds stock to a product
func (s *InventoryService) AddStock(ctx context.Context, productID string, req domain.AddStockRequest, userID string) (*domain.InventoryTransactionResult, error) {
	txn, err := s.inventoryRepo.AddStock(ctx, productID, req.Quantity, req.Reason, userID)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Added stock", keyProductID, productID, "quantity", req.Quantity)

	return &domain.InventoryTransactionResult{
		ProductID:        productID,
		PreviousQuantity: txn.PreviousQty,
		ChangeQuantity:   req.Quantity,
		NewQuantity:      txn.NewQty,
		AvailableQty:     txn.NewQty,
		TransactionID:    txn.ID,
	}, nil
}

// RemoveStock removes stock from a product
func (s *InventoryService) RemoveStock(ctx context.Context, productID string, req domain.RemoveStockRequest, userID string) (*domain.InventoryTransactionResult, error) {
	txn, err := s.inventoryRepo.RemoveStock(ctx, productID, req.Quantity, req.Reason, userID)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Removed stock", keyProductID, productID, "quantity", req.Quantity)

	s.emitStockLevelMetrics(ctx, productID)

	return &domain.InventoryTransactionResult{
		ProductID:        productID,
		PreviousQuantity: txn.PreviousQty,
		ChangeQuantity:   -req.Quantity,
		NewQuantity:      txn.NewQty,
		AvailableQty:     txn.NewQty,
		TransactionID:    txn.ID,
	}, nil
}

// emitStockLevelMetrics records out-of-stock / low-stock signals after a stock
// removal. Replaces the InventoryOutOfStock / InventoryLowStock events the old
// event system emitted. Best-effort: a failed read is logged, never returned.
func (s *InventoryService) emitStockLevelMetrics(ctx context.Context, productID string) {
	inv, err := s.inventoryRepo.GetByProductID(ctx, productID)
	if err != nil {
		slog.WarnContext(ctx, "Failed to read inventory for stock-level metrics", keyProductID, productID, "error", err)
		return
	}
	switch {
	case inv.AvailableQty <= 0:
		metrics.Record(ctx, "inventory_out_of_stock", metrics.L{keyProductID: productID})
	case inv.AvailableQty <= inv.LowStockThreshold:
		metrics.Record(ctx, "inventory_low_stock", metrics.L{keyProductID: productID})
	}
}

// AdjustStock adjusts stock to a specific quantity
func (s *InventoryService) AdjustStock(ctx context.Context, productID string, req domain.AdjustStockRequest, userID string) (*domain.InventoryTransactionResult, error) {
	txn, err := s.inventoryRepo.AdjustStock(ctx, productID, req.NewQuantity, req.Reason, userID)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Adjusted stock", keyProductID, productID, "previous_qty", txn.PreviousQty, "new_qty", txn.NewQty)

	return &domain.InventoryTransactionResult{
		ProductID:        productID,
		PreviousQuantity: txn.PreviousQty,
		ChangeQuantity:   txn.NewQty - txn.PreviousQty,
		NewQuantity:      txn.NewQty,
		AvailableQty:     txn.NewQty,
		TransactionID:    txn.ID,
	}, nil
}

// GetTransactions retrieves inventory transactions
func (s *InventoryService) GetTransactions(ctx context.Context, productID string, pagination domain.PaginationRequest) (*domain.ListInventoryTransactionsResponse, error) {
	result, err := s.inventoryRepo.GetTransactions(ctx, productID, pagination)
	if err != nil {
		return nil, err
	}

	s.resolveActorNames(ctx, result.Transactions)
	return result, nil
}

// resolveActorNames fills in CreatedByName for the movements an admin made.
// One lookup per distinct user, not per row: a page of stocktake corrections is
// usually one person. A name that cannot be resolved is left empty rather than
// failing the read — the history is still worth showing without it.
func (s *InventoryService) resolveActorNames(ctx context.Context, txns []*domain.InventoryTransaction) {
	if s.userRepo == nil {
		return
	}

	names := make(map[string]string)

	for _, txn := range txns {
		if txn.CreatedBy == "" {
			continue
		}

		name, seen := names[txn.CreatedBy]
		if !seen {
			user, err := s.userRepo.GetByID(ctx, txn.CreatedBy)
			if err != nil {
				slog.WarnContext(ctx, "Failed to resolve inventory actor",
					"user_id", txn.CreatedBy, "error", err)
			} else if user != nil {
				name = strings.TrimSpace(user.FirstName + " " + user.LastName)
				if name == "" {
					name = user.Email
				}
			}
			names[txn.CreatedBy] = name
		}

		txn.CreatedByName = name
	}
}

// GetLowStockProducts retrieves products with low stock
func (s *InventoryService) GetLowStockProducts(ctx context.Context, pagination domain.PaginationRequest) (*domain.ListInventoryResponse, error) {
	return s.inventoryRepo.GetLowStockProducts(ctx, pagination)
}

// Ensure interface compliance
var _ domain.InventoryService = (*InventoryService)(nil)
