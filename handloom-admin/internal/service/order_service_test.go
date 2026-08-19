package service

import (
	"context"
	"fmt"
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
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

		// Aggregated per product, so two lines of one product are one movement.
		mockInventoryRepo.EXPECT().
			ReserveOrderStock(gomock.Any(), gomock.Any(), map[string]int{"prod_123": 2}).
			Return(nil)

		// The order total must reach TotalSpent — it is what the admin customer
		// view renders as lifetime spend.
		mockCustomerRepo.EXPECT().
			RecordPurchase(gomock.Any(), "cust_123", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, amountPaise int64) (int64, error) {
				assert.Positive(t, amountPaise)
				return int64(1), nil
			})

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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful get order with details", func(t *testing.T) {
		// The detail view carries what the payment says has actually gone back.
		mockPaymentRepo.EXPECT().GetByOrderID(gomock.Any(), gomock.Any()).
			Return(&domain.Payment{ID: "pay_1"}, nil).AnyTimes()
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
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

		// No items, so the batch call carries an empty map rather than not
		// happening at all — the repository short-circuits it.
		mockInventoryRepo.EXPECT().
			CommitOrderStock(gomock.Any(), "order_123", map[string]int{}).
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
			ReleaseOrderStock(gomock.Any(), "order_123", map[string]int{"prod_123": 2}).
			Return(nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusCancelled, "admin_123")

		require.NoError(t, err)
	})
}

func TestOrderService_UpdateStatus_Inventory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("shipping commits stock for every item", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusConfirmed,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
				{ProductID: "prod_456", Quantity: 1},
			},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_123").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			CommitOrderStock(gomock.Any(), "order_123", map[string]int{"prod_123": 2, "prod_456": 1}).
			Return(nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusShipped, "admin_123")
		require.NoError(t, err)
	})

	t.Run("delivery has no inventory effect", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_789",
			Status: domain.OrderStatusShipped,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_789").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		// No inventory expectation: gomock fails the test on any unexpected call.

		err := service.UpdateStatus(ctx, "order_789", domain.OrderStatusDelivered, "admin_123")
		require.NoError(t, err)
	})

	// The order lines are deliberately not passed: the repository restocks what
	// the order committed, read from its COMMIT ledger rows. A line that never
	// committed was never decremented, so adding it back would inflate stock.
	t.Run("return restocks the order, not its lines", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_ret",
			Status: domain.OrderStatusDelivered,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
			},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_ret").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			RestockOrderStock(gomock.Any(), "order_ret").
			Return(nil)

		err := service.UpdateStatus(ctx, "order_ret", domain.OrderStatusReturned, "admin_123")
		require.NoError(t, err)
	})

	t.Run("a failed restock does not fail the return", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_ret_fail",
			Status: domain.OrderStatusDelivered,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_ret_fail").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			RestockOrderStock(gomock.Any(), "order_ret_fail").
			Return(errors.New(errors.ErrCodeInternal, "boom"))

		err := service.UpdateStatus(ctx, "order_ret_fail", domain.OrderStatusReturned, "admin_123")
		require.NoError(t, err, "inventory failure must not block the status change")
		require.Equal(t, domain.OrderStatusReturned, order.Status)
	})

	t.Run("canceling via status update sets CancelledAt", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_cancel_update",
			Status: domain.OrderStatusConfirmed,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_cancel_update").Return(order, nil)
		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, order *domain.Order) error {
				assert.NotNil(t, order.CancelledAt)
				return nil
			})
		mockInventoryRepo.EXPECT().
			ReleaseOrderStock(gomock.Any(), "order_cancel_update", map[string]int{"prod_123": 2}).
			Return(nil)

		err := service.UpdateStatus(ctx, "order_cancel_update", domain.OrderStatusCancelled, "admin_123")
		require.NoError(t, err)
		assert.NotNil(t, order.CancelledAt)
	})

	t.Run("a failed commit does not fail the shipment", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_fail",
			Status: domain.OrderStatusConfirmed,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_fail").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			CommitOrderStock(gomock.Any(), "order_fail", map[string]int{"prod_123": 2}).
			Return(errors.New(errors.ErrCodeInsufficientStock, "insufficient stock"))

		err := service.UpdateStatus(ctx, "order_fail", domain.OrderStatusShipped, "admin_123")
		require.NoError(t, err, "inventory failure must not block the status change")
		require.Equal(t, domain.OrderStatusShipped, order.Status)
	})

	t.Run("orderRepo.Update failure prevents any inventory mutation", func(t *testing.T) {
		// Regression test: inventory mutations must run AFTER the status is
		// persisted, not before. Registering no CommitOrderStock/RestockOrderStock/
		// ReleaseOrderStock expectation means gomock fails this test the moment an
		// unexpected call happens — which is exactly what the old ordering
		// (mutate inventory, then persist) would have done here despite the
		// persist failing.
		order := &domain.Order{
			ID:     "order_updatefail",
			Status: domain.OrderStatusConfirmed,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_updatefail").Return(order, nil)
		mockOrderRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(errors.New(errors.ErrCodeInternal, "db write failed"))

		err := service.UpdateStatus(ctx, "order_updatefail", domain.OrderStatusShipped, "admin_123")
		require.Error(t, err, "a persist failure must propagate")
	})
}

// TestReleaseFailureReason pins the inventory_mutation_failed reason label
// chosen for a ReleaseOrderStock failure: insufficient-stock (the reservation was
// already zeroed by an earlier release, e.g. a failed-payment rollback) maps
// to "release_unreserved" so it doesn't get counted as a real leak signal.
// Any other error keeps the default "release" reason.
func TestReleaseFailureReason(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "insufficient stock maps to release_unreserved",
			err:      errors.New(errors.ErrCodeInsufficientStock, "insufficient stock"),
			expected: reasonReleaseUnreserved,
		},
		{
			name:     "other AppError keeps release",
			err:      errors.New(errors.ErrCodeNotFound, "Inventory not found"),
			expected: reasonRelease,
		},
		{
			name:     "non-AppError keeps release",
			err:      fmt.Errorf("connection reset"),
			expected: reasonRelease,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, releaseFailureReason(tt.err))
		})
	}
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("add internal note", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			AddNote(ctx, "order_123", gomock.Any()).
			DoAndReturn(func(ctx context.Context, id string, note domain.OrderNote) error {
				assert.Equal(t, "Test note", note.Note)
				assert.Equal(t, "admin_123", note.CreatedBy)
				// The admin note list highlights internal notes, so the flag has
				// to survive the trip rather than being dropped on the floor.
				assert.True(t, note.IsInternal)
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("successful tracking update", func(t *testing.T) {
		// The courier URL must reach the repository — the storefront renders it
		// as the "Track on courier website" link.
		mockOrderRepo.EXPECT().
			UpdateTracking(ctx, "order_123", "TRACK123456", "BlueDart", "https://bluedart.example/t/TRACK123456").
			Return(nil)

		err := service.UpdateTracking(
			ctx, "order_123", "TRACK123456", "BlueDart",
			"https://bluedart.example/t/TRACK123456", "admin_123",
		)

		require.NoError(t, err)
	})

	t.Run("tracking update - order not found", func(t *testing.T) {
		mockOrderRepo.EXPECT().
			UpdateTracking(ctx, "nonexistent", "TRACK123", "BlueDart", "").
			Return(errors.NotFound("Order"))

		err := service.UpdateTracking(ctx, "nonexistent", "TRACK123", "BlueDart", "", "admin_123")
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
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
			ReleaseOrderStock(gomock.Any(), "order_123", map[string]int{"prod_123": 2, "prod_456": 1}).
			Return(nil)

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
			ReleaseOrderStock(gomock.Any(), "order_456", map[string]int{"prod_123": 1}).
			Return(nil)

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

// TestOrderService_CancelOrder_Inventory covers the PROCESSING widening: the
// validTransitions table already allowed PROCESSING -> CANCELLED, but
// CancelOrder rejected it. These subtests pin CancelOrder's accepted statuses
// to match that table, and confirm SHIPPED (post-commit) still stays out of
// reach.
func TestOrderService_CancelOrder_Inventory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("canceling a processing order releases stock", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_proc",
			Status: domain.OrderStatusProcessing,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 3}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_proc").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			ReleaseOrderStock(gomock.Any(), "order_proc", map[string]int{"prod_123": 3}).
			Return(nil)

		err := service.CancelOrder(ctx, "order_proc", "out of stock", "admin_123")
		require.NoError(t, err)
	})

	t.Run("canceling a shipped order is rejected", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_ship",
			Status: domain.OrderStatusShipped,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 1}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_ship").Return(order, nil)

		err := service.CancelOrder(ctx, "order_ship", "too late", "admin_123")
		require.Error(t, err)
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
	mockPaymentRepo := mocks.NewMockPaymentRepository(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPaymentRepo,
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
