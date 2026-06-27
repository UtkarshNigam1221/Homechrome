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

func TestOrderService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful order creation with standard price", func(t *testing.T) {
		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
			Phone:     "+1234567890",
		}

		product := &domain.Product{
			ID:           "prod_123",
			Name:         "Silk Saree",
			SKU:          "SKU-001",
			CategoryID:   "cat_123",
			SellingPrice: 500000, // 5000 INR in paise
		}

		inventory := &domain.Inventory{
			ProductID:    "prod_123",
			Quantity:     100,
			AvailableQty: 90,
		}

		req := domain.CreateOrderRequest{
			CustomerID: "cust_123",
			Items: []domain.OrderItemInput{
				{
					ProductID: "prod_123",
					Quantity:  2,
				},
			},
			ShippingAddress: domain.Address{
				AddressLine1: "123 Main St",
				City:         "Mumbai",
				State:        "Maharashtra",
				PostalCode:   "400001",
				Country:      "India",
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(gomock.Any(), "cust_123").
			Return(customer, nil)

		mockProductRepo.EXPECT().
			GetByID(gomock.Any(), "prod_123").
			Return(product, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(gomock.Any(), "prod_123").
			Return(inventory, nil)

		mockOrderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, "cust_123", order.CustomerID)
				assert.Len(t, order.Items, 1)
				assert.Equal(t, int64(1000000), order.Subtotal) // 2 * 500000
				assert.Equal(t, domain.OrderStatusPending, order.Status)
				return nil
			})

		mockInventoryRepo.EXPECT().
			ReserveStock(gomock.Any(), "prod_123", 2, gomock.Any()).
			Return(&domain.InventoryTransaction{}, nil)

		mockCustomerRepo.EXPECT().
			IncrementOrderCount(gomock.Any(), "cust_123").
			Return(int64(1), nil)

		order, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, "cust_123", order.CustomerID)
	})

	t.Run("customer not found", func(t *testing.T) {
		req := domain.CreateOrderRequest{
			CustomerID: "nonexistent",
			Items: []domain.OrderItemInput{
				{ProductID: "prod_123", Quantity: 1},
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(gomock.Any(), "nonexistent").
			Return(nil, errors.NotFound("Customer not found"))

		order, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, order)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer not found")
	})

	t.Run("product not found", func(t *testing.T) {
		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "John",
			LastName:  "Doe",
		}

		req := domain.CreateOrderRequest{
			CustomerID: "cust_123",
			Items: []domain.OrderItemInput{
				{ProductID: "prod_nonexistent", Quantity: 1},
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(gomock.Any(), "cust_123").
			Return(customer, nil)

		mockProductRepo.EXPECT().
			GetByID(gomock.Any(), "prod_nonexistent").
			Return(nil, errors.NotFound("Product"))

		order, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, order)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product not found")
	})

	t.Run("insufficient stock", func(t *testing.T) {
		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "John",
			LastName:  "Doe",
		}

		product := &domain.Product{
			ID:           "prod_123",
			Name:         "Silk Saree",
			SKU:          "SKU-001",
			SellingPrice: 500000,
		}

		inventory := &domain.Inventory{
			ProductID:    "prod_123",
			Quantity:     5,
			AvailableQty: 5,
		}

		req := domain.CreateOrderRequest{
			CustomerID: "cust_123",
			Items: []domain.OrderItemInput{
				{ProductID: "prod_123", Quantity: 10}, // More than available
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(gomock.Any(), "cust_123").
			Return(customer, nil)

		mockProductRepo.EXPECT().
			GetByID(gomock.Any(), "prod_123").
			Return(product, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(gomock.Any(), "prod_123").
			Return(inventory, nil)

		order, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, order)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient stock")
	})
}

func TestOrderService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful get order with details", func(t *testing.T) {
		order := &domain.Order{
			ID:          "order_123",
			OrderNumber: "HL202401010001",
			CustomerID:  "cust_123",
			Items: []domain.OrderItem{
				{ProductID: "prod_123", ProductName: "Silk Saree", ProductSKU: "SKU-001", Quantity: 1},
			},
			Status:      domain.OrderStatusPending,
			TotalAmount: 500000,
		}

		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "John",
			LastName:  "Doe",
		}

		product := &domain.Product{
			ID:   "prod_123",
			Name: "Silk Saree",
			Images: []domain.ProductImage{
				{URL: "https://example.com/image.jpg"},
			},
		}

		mockOrderRepo.EXPECT().
			GetByID(ctx, "order_123").
			Return(order, nil)

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_123").
			Return(customer, nil)

		mockProductRepo.EXPECT().
			BatchGetByIDs(ctx, []string{"prod_123"}).
			Return([]*domain.Product{product}, nil)

		result, err := service.GetByID(ctx, "order_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "order_123", result.ID)
		assert.NotNil(t, result.Customer)
		assert.Len(t, result.ItemDetails, 1)
	})

	t.Run("order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Order not found"))

		result, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestOrderService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("valid status transition: pending to confirmed", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusPending,
			Items:  []domain.OrderItem{},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.OrderStatusConfirmed, order.Status)
				return nil
			})

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusConfirmed, "admin_123")

		require.NoError(t, err)
	})

	t.Run("invalid status transition", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusDelivered, // Cannot go back to pending
			Items:  []domain.OrderItem{},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusPending, "admin_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot transition")
	})

	t.Run("confirmed to processing", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusConfirmed,
			Items:  []domain.OrderItem{},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.OrderStatusProcessing, order.Status)
				return nil
			})

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusProcessing, "admin_123")
		require.NoError(t, err)
	})

	t.Run("processing to shipped", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusProcessing,
			Items:  []domain.OrderItem{},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusShipped, "admin_123")
		require.NoError(t, err)
	})

	t.Run("shipped to delivered", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusShipped,
			Items:  []domain.OrderItem{},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusDelivered, "admin_123")
		require.NoError(t, err)
	})

	t.Run("order not found for status update", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "nonexistent").
			Return(nil, errors.NotFound("Order"))

		err := service.UpdateStatus(ctx, "nonexistent", domain.OrderStatusConfirmed, "admin_123")
		require.Error(t, err)
	})

	t.Run("status transition to canceled releases stock", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusPending,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
			},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		mockInventoryRepo.EXPECT().
			ReleaseStock(gomock.Any(), "prod_123", 2, "order_123").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusCancelled, "admin_123")

		require.NoError(t, err)
	})
}

func TestOrderService_AddNote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("add internal note", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			AddNote(ctx, "order_123", gomock.Any()).
			DoAndReturn(func(ctx context.Context, id string, note domain.OrderNote) error {
				assert.Equal(t, "Test note", note.Note)
				assert.Equal(t, "admin_123", note.CreatedBy)
				return nil
			})

		err := service.AddNote(ctx, "order_123", "Test note", true, "admin_123")

		require.NoError(t, err)
	})

	t.Run("add note - order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			AddNote(ctx, "nonexistent", gomock.Any()).
			Return(errors.NotFound("Order"))

		err := service.AddNote(ctx, "nonexistent", "Test", true, "admin_123")
		require.Error(t, err)
	})

	t.Run("add customer-visible note", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			AddNote(ctx, "order_123", gomock.Any()).
			DoAndReturn(func(ctx context.Context, id string, note domain.OrderNote) error {
				assert.NotEmpty(t, note.Note)
				return nil
			})

		err := service.AddNote(ctx, "order_123", "Customer note", false, "admin_123")

		require.NoError(t, err)
	})
}

func TestOrderService_UpdateTracking(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful tracking update", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			UpdateTracking(ctx, "order_123", "TRACK123456", "BlueDart").
			Return(nil)

		err := service.UpdateTracking(ctx, "order_123", "TRACK123456", "BlueDart", "admin_123")

		require.NoError(t, err)
	})

	t.Run("tracking update - order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			UpdateTracking(ctx, "nonexistent", "TRACK123", "BlueDart").
			Return(errors.NotFound("Order"))

		err := service.UpdateTracking(ctx, "nonexistent", "TRACK123", "BlueDart", "admin_123")
		require.Error(t, err)
	})
}

func TestOrderService_CancelOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful cancellation from pending", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusPending,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
				{ProductID: "prod_456", Quantity: 1},
			},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.OrderStatusCancelled, order.Status)
				assert.NotNil(t, order.CancelledAt)
				return nil
			})

		mockInventoryRepo.EXPECT().
			ReleaseStock(gomock.Any(), "prod_123", 2, "order_123").
			Return(&domain.InventoryTransaction{}, nil)

		mockInventoryRepo.EXPECT().
			ReleaseStock(gomock.Any(), "prod_456", 1, "order_123").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.CancelOrder(ctx, "order_123", "Customer requested", "admin_123")

		require.NoError(t, err)
	})

	t.Run("successful cancellation from confirmed", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_456",
			Status: domain.OrderStatusConfirmed,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 1},
			},
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_456").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.OrderStatusCancelled, order.Status)
				return nil
			})

		mockInventoryRepo.EXPECT().
			ReleaseStock(gomock.Any(), "prod_123", 1, "order_456").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.CancelOrder(ctx, "order_456", "Changed mind", "admin_123")
		require.NoError(t, err)
	})

	t.Run("cancel order - order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "nonexistent").
			Return(nil, errors.NotFound("Order"))

		err := service.CancelOrder(ctx, "nonexistent", "reason", "admin_123")
		require.Error(t, err)
	})

	t.Run("cannot cancel shipped order", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusShipped,
		}

		mockOrderRepo.EXPECT().
			GetByID(gomock.Any(), "order_123").
			Return(order, nil)

		err := service.CancelOrder(ctx, "order_123", "Customer requested", "admin_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be canceled")
	})
}

func TestOrderService_RefundOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful full refund", func(t *testing.T) {
		order := &domain.Order{
			ID:          "order_123",
			Status:      domain.OrderStatusDelivered,
			TotalAmount: 100000,
		}

		mockOrderRepo.EXPECT().
			GetByID(ctx, "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.PaymentStatusRefunded, order.PaymentStatus)
				return nil
			})

		err := service.RefundOrder(ctx, "order_123", 100000, "Defective product", "admin_123")

		require.NoError(t, err)
	})

	t.Run("successful partial refund", func(t *testing.T) {
		order := &domain.Order{
			ID:          "order_123",
			Status:      domain.OrderStatusDelivered,
			TotalAmount: 100000,
		}

		mockOrderRepo.EXPECT().
			GetByID(ctx, "order_123").
			Return(order, nil)

		mockOrderRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, order *domain.Order) error {
				assert.Equal(t, domain.PaymentStatusRefunded, order.PaymentStatus)
				return nil
			})

		err := service.RefundOrder(ctx, "order_123", 50000, "Partial refund", "admin_123")

		require.NoError(t, err)
	})

	t.Run("refund order - order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Order"))

		err := service.RefundOrder(ctx, "nonexistent", 50000, "reason", "admin_123")
		require.Error(t, err)
	})

	t.Run("refund amount exceeds total", func(t *testing.T) {
		order := &domain.Order{
			ID:          "order_123",
			Status:      domain.OrderStatusDelivered,
			TotalAmount: 100000,
		}

		mockOrderRepo.EXPECT().
			GetByID(ctx, "order_123").
			Return(order, nil)

		err := service.RefundOrder(ctx, "order_123", 150000, "Too much", "admin_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed")
	})
}

func TestOrderService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful list orders", func(t *testing.T) {
		req := domain.ListOrdersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 20,
			},
		}

		expectedResponse := &domain.ListOrdersResponse{
			Orders: []*domain.Order{
				{ID: "order_1", OrderNumber: "HL20240101001"},
				{ID: "order_2", OrderNumber: "HL20240101002"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockOrderRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Orders, 2)
	})

	t.Run("list orders with status filter", func(t *testing.T) {
		status := domain.OrderStatusPending
		req := domain.ListOrdersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
			Status: &status,
		}

		expectedResponse := &domain.ListOrdersResponse{
			Orders: []*domain.Order{
				{ID: "order_1", Status: domain.OrderStatusPending},
			},
			Pagination: domain.PaginationResponse{
				Limit:   10,
				HasMore: false,
			},
		}

		mockOrderRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Orders, 1)
	})
}

func TestIsValidStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     domain.OrderStatus
		to       domain.OrderStatus
		expected bool
	}{
		{"pending to confirmed", domain.OrderStatusPending, domain.OrderStatusConfirmed, true},
		{"pending to canceled", domain.OrderStatusPending, domain.OrderStatusCancelled, true},
		{"pending to shipped", domain.OrderStatusPending, domain.OrderStatusShipped, false},
		{"confirmed to processing", domain.OrderStatusConfirmed, domain.OrderStatusProcessing, true},
		{"confirmed to canceled", domain.OrderStatusConfirmed, domain.OrderStatusCancelled, true},
		{"processing to shipped", domain.OrderStatusProcessing, domain.OrderStatusShipped, true},
		{"shipped to delivered", domain.OrderStatusShipped, domain.OrderStatusDelivered, true},
		{"shipped to returned", domain.OrderStatusShipped, domain.OrderStatusReturned, true},
		{"delivered to returned", domain.OrderStatusDelivered, domain.OrderStatusReturned, true},
		{"delivered to pending", domain.OrderStatusDelivered, domain.OrderStatusPending, false},
		{"canceled to confirmed", domain.OrderStatusCancelled, domain.OrderStatusConfirmed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidStatusTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}
