// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
)

// CacheInvalidator allows services to invalidate cached product data after mutations.
type CacheInvalidator interface {
	DeletePrefix(prefix string)
}

// InventoryService implements domain.InventoryService
type InventoryService struct {
	inventoryRepo domain.InventoryRepository
	cache         CacheInvalidator
	publisher     event.EventPublisher
}

// NewInventoryService creates a new InventoryService
func NewInventoryService(
	inventoryRepo domain.InventoryRepository,
	cache CacheInvalidator,
	publisher event.EventPublisher,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		cache:         cache,
		publisher:     publisher,
	}
}

// invalidateProductCache clears cached product lists/items so the next read
// picks up the latest inventory data from the JOIN.
func (s *InventoryService) invalidateProductCache() {
	if s.cache != nil {
		s.cache.DeletePrefix("prod:")
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

	if pubErr := s.publisher.Publish(ctx, event.New(event.InventoryRestocked, map[string]interface{}{
		"product_id":   productID,
		"quantity":     req.Quantity,
		"new_quantity": txn.NewQty,
	})); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish inventory.restocked event", "error", pubErr)
	}

	s.invalidateProductCache()
	slog.InfoContext(ctx, "Added stock", "product_id", productID, "quantity", req.Quantity)

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

	// Check stock levels and emit appropriate events
	inventory, invErr := s.inventoryRepo.GetByProductID(ctx, productID)
	if invErr != nil {
		slog.ErrorContext(ctx, "Failed to fetch inventory for stock event check",
			"product_id", productID,
			"error", invErr,
		)
	}
	if inventory != nil {
		if inventory.AvailableQty <= 0 {
			if pubErr := s.publisher.Publish(ctx, event.New(event.InventoryOutOfStock, map[string]interface{}{
				"product_id":    productID,
				"available_qty": inventory.AvailableQty,
			})); pubErr != nil {
				slog.ErrorContext(ctx, "Failed to publish inventory.out_of_stock event", "error", pubErr)
			}
		} else if inventory.AvailableQty <= inventory.LowStockThreshold {
			if pubErr := s.publisher.Publish(ctx, event.New(event.InventoryLowStock, map[string]interface{}{
				"product_id":          productID,
				"available_qty":       inventory.AvailableQty,
				"low_stock_threshold": inventory.LowStockThreshold,
			})); pubErr != nil {
				slog.ErrorContext(ctx, "Failed to publish inventory.low_stock event", "error", pubErr)
			}
		}
	}

	s.invalidateProductCache()
	slog.InfoContext(ctx, "Removed stock", "product_id", productID, "quantity", req.Quantity)

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

	s.invalidateProductCache()
	slog.InfoContext(ctx, "Adjusted stock", "product_id", productID, "previous_qty", txn.PreviousQty, "new_qty", txn.NewQty)

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
