package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

func TestInventoryService_GetByProductID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful get inventory", func(t *testing.T) {
		expectedInventory := &domain.Inventory{
			ProductID:         "prod_123",
			Quantity:          100,
			ReservedQty:       10,
			AvailableQty:      90,
			LowStockThreshold: 20,
		}

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(expectedInventory, nil)

		inventory, err := service.GetByProductID(ctx, "prod_123")

		require.NoError(t, err)
		assert.NotNil(t, inventory)
		assert.Equal(t, "prod_123", inventory.ProductID)
		assert.Equal(t, 100, inventory.Quantity)
		assert.Equal(t, 90, inventory.AvailableQty)
	})

	t.Run("inventory not found", func(t *testing.T) {
		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Inventory not found"))

		inventory, err := service.GetByProductID(ctx, "nonexistent")

		assert.Nil(t, inventory)
		require.Error(t, err)
	})
}

func TestInventoryService_AddStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful add stock", func(t *testing.T) {
		req := domain.AddStockRequest{
			Quantity: 50,
			Reason:   "New shipment received",
		}

		transaction := &domain.InventoryTransaction{
			ID:          "txn_123",
			ProductID:   "prod_123",
			Type:        domain.InventoryTransactionTypeAdd,
			Quantity:    50,
			PreviousQty: 100,
			NewQty:      150,
			Reason:      req.Reason,
		}

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 50, req.Reason, "user_123").
			Return(transaction, nil)

		result, err := service.AddStock(ctx, "prod_123", req, "user_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "prod_123", result.ProductID)
		assert.Equal(t, 100, result.PreviousQuantity)
		assert.Equal(t, 50, result.ChangeQuantity)
		assert.Equal(t, 150, result.NewQuantity)
	})

	t.Run("add stock failure", func(t *testing.T) {
		req := domain.AddStockRequest{
			Quantity: 50,
			Reason:   "Test",
		}

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 50, req.Reason, "user_123").
			Return(nil, errors.Internal("Database error"))

		result, err := service.AddStock(ctx, "prod_123", req, "user_123")

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestInventoryService_RemoveStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful remove stock", func(t *testing.T) {
		req := domain.RemoveStockRequest{
			Quantity: 20,
			Reason:   "Damaged items",
		}

		transaction := &domain.InventoryTransaction{
			ID:          "txn_123",
			ProductID:   "prod_123",
			Type:        domain.InventoryTransactionTypeRemove,
			Quantity:    -20,
			PreviousQty: 100,
			NewQty:      80,
			Reason:      req.Reason,
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 20, req.Reason, "user_123").
			Return(transaction, nil)
		// RemoveStock now re-reads inventory to emit stock-level metrics.
		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{ProductID: "prod_123", AvailableQty: 80, LowStockThreshold: 10}, nil)

		result, err := service.RemoveStock(ctx, "prod_123", req, "user_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "prod_123", result.ProductID)
		assert.Equal(t, 100, result.PreviousQuantity)
		assert.Equal(t, -20, result.ChangeQuantity)
		assert.Equal(t, 80, result.NewQuantity)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		req := domain.RemoveStockRequest{
			Quantity: 200,
			Reason:   "Too much",
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 200, req.Reason, "user_123").
			Return(nil, errors.New(errors.ErrCodeInsufficientStock, "Insufficient stock"))

		result, err := service.RemoveStock(ctx, "prod_123", req, "user_123")

		assert.Nil(t, result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient stock")
	})
}

func TestInventoryService_AdjustStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful adjust stock up", func(t *testing.T) {
		req := domain.AdjustStockRequest{
			NewQuantity: 150,
			Reason:      "Physical count correction",
		}

		transaction := &domain.InventoryTransaction{
			ID:          "txn_123",
			ProductID:   "prod_123",
			Type:        domain.InventoryTransactionTypeAdjust,
			Quantity:    50,
			PreviousQty: 100,
			NewQty:      150,
			Reason:      req.Reason,
		}

		mockInventoryRepo.EXPECT().
			AdjustStock(ctx, "prod_123", 150, req.Reason, "user_123").
			Return(transaction, nil)

		result, err := service.AdjustStock(ctx, "prod_123", req, "user_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.PreviousQuantity)
		assert.Equal(t, 50, result.ChangeQuantity)
		assert.Equal(t, 150, result.NewQuantity)
	})

	t.Run("successful adjust stock down", func(t *testing.T) {
		req := domain.AdjustStockRequest{
			NewQuantity: 80,
			Reason:      "Physical count correction",
		}

		transaction := &domain.InventoryTransaction{
			ID:          "txn_123",
			ProductID:   "prod_123",
			Type:        domain.InventoryTransactionTypeAdjust,
			Quantity:    -20,
			PreviousQty: 100,
			NewQty:      80,
			Reason:      req.Reason,
		}

		mockInventoryRepo.EXPECT().
			AdjustStock(ctx, "prod_123", 80, req.Reason, "user_123").
			Return(transaction, nil)

		result, err := service.AdjustStock(ctx, "prod_123", req, "user_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, -20, result.ChangeQuantity)
	})
}

func TestInventoryService_GetTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful get transactions", func(t *testing.T) {
		pagination := domain.PaginationRequest{
			Limit: 20,
		}

		expectedResponse := &domain.ListInventoryTransactionsResponse{
			Transactions: []*domain.InventoryTransaction{
				{ID: "txn_1", Type: domain.InventoryTransactionTypeAdd, Quantity: 50},
				{ID: "txn_2", Type: domain.InventoryTransactionTypeRemove, Quantity: -10},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockInventoryRepo.EXPECT().
			GetTransactions(ctx, "prod_123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetTransactions(ctx, "prod_123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Transactions, 2)
	})
}

func TestInventoryService_GetLowStockProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	service := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("successful get low stock products", func(t *testing.T) {
		pagination := domain.PaginationRequest{
			Limit: 20,
		}

		expectedResponse := &domain.ListInventoryResponse{
			Inventories: []*domain.Inventory{
				{ProductID: "prod_1", Quantity: 5, LowStockThreshold: 10},
				{ProductID: "prod_2", Quantity: 3, LowStockThreshold: 5},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockInventoryRepo.EXPECT().
			GetLowStockProducts(ctx, pagination).
			Return(expectedResponse, nil)

		response, err := service.GetLowStockProducts(ctx, pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Inventories, 2)
	})
}

func TestInventoryService_ResultFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	svc := NewInventoryService(mockInventoryRepo)
	ctx := context.Background()

	t.Run("AddStock result fields match transaction", func(t *testing.T) {
		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 25, "shipment", "user_1").
			Return(&domain.InventoryTransaction{
				ID: "txn_abc", PreviousQty: 75, NewQty: 100,
			}, nil)

		result, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 25, Reason: "shipment",
		}, "user_1")

		require.NoError(t, err)
		assert.Equal(t, "prod_123", result.ProductID)
		assert.Equal(t, 75, result.PreviousQuantity)
		assert.Equal(t, 25, result.ChangeQuantity)
		assert.Equal(t, 100, result.NewQuantity)
		assert.Equal(t, 100, result.AvailableQty)
		assert.Equal(t, "txn_abc", result.TransactionID)
	})

	t.Run("RemoveStock result has negative change quantity", func(t *testing.T) {
		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 30, "damaged", "user_1").
			Return(&domain.InventoryTransaction{
				ID: "txn_def", PreviousQty: 100, NewQty: 70,
			}, nil)
		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{ProductID: "prod_123", AvailableQty: 70, LowStockThreshold: 10}, nil)

		result, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 30, Reason: "damaged",
		}, "user_1")

		require.NoError(t, err)
		assert.Equal(t, -30, result.ChangeQuantity)
		assert.Equal(t, 100, result.PreviousQuantity)
		assert.Equal(t, 70, result.NewQuantity)
	})
}
