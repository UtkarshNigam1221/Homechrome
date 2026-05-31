// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"

	"github.com/handloom/admin/internal/domain"
)

// InventoryService implements domain.InventoryService
type InventoryService struct {
	inventoryRepo domain.InventoryRepository
}

// NewInventoryService creates a new InventoryService
func NewInventoryService(
	inventoryRepo domain.InventoryRepository,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
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

	return &domain.InventoryTransactionResult{
		ProductID:        productID,
		PreviousQuantity: txn.PreviousQty,
		ChangeQuantity:   -req.Quantity,
		NewQuantity:      txn.NewQty,
		AvailableQty:     txn.NewQty,
		TransactionID:    txn.ID,
	}, nil
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
	return s.inventoryRepo.GetTransactions(ctx, productID, pagination)
}

// GetLowStockProducts retrieves products with low stock
func (s *InventoryService) GetLowStockProducts(ctx context.Context, pagination domain.PaginationRequest) (*domain.ListInventoryResponse, error) {
	return s.inventoryRepo.GetLowStockProducts(ctx, pagination)
}

// Ensure interface compliance
var _ domain.InventoryService = (*InventoryService)(nil)
