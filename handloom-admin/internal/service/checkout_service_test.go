package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
)

func TestFindShippingAddress(t *testing.T) {
	customer := &domain.Customer{
		Addresses: []domain.Address{
			{ID: "addr-1", City: "Bengaluru"},
			{ID: "addr-2", City: "Mumbai"},
		},
	}

	t.Run("found", func(t *testing.T) {
		got, err := findShippingAddress(customer, "addr-2")
		require.NoError(t, err)
		assert.Equal(t, "Mumbai", got.City)
	})

	t.Run("not found returns NotFound", func(t *testing.T) {
		_, err := findShippingAddress(customer, "missing")
		require.Error(t, err)
	})

	t.Run("empty address list", func(t *testing.T) {
		_, err := findShippingAddress(&domain.Customer{}, "addr-1")
		require.Error(t, err)
	})
}

func TestCartItemsToOrderItems(t *testing.T) {
	t.Run("maps fields and assigns unique ids", func(t *testing.T) {
		items := []domain.CartItem{
			{ProductID: "p1", ProductName: "Saree", Quantity: 2, UnitPrice: 100, TotalPrice: 200},
			{ProductID: "p2", ProductName: "Dupatta", Quantity: 1, UnitPrice: 50, TotalPrice: 50},
		}

		got := cartItemsToOrderItems(items)

		require.Len(t, got, 2)
		assert.Equal(t, "p1", got[0].ProductID)
		assert.Equal(t, "Saree", got[0].ProductName)
		assert.Equal(t, 2, got[0].Quantity)
		assert.Equal(t, int64(200), got[0].TotalPrice)

		// Each order item gets a fresh, non-empty, unique ID.
		assert.NotEmpty(t, got[0].ID)
		assert.NotEmpty(t, got[1].ID)
		assert.NotEqual(t, got[0].ID, got[1].ID)
	})

	t.Run("empty input yields empty non-nil slice", func(t *testing.T) {
		got := cartItemsToOrderItems(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestCheckoutService_Initiate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartService := mocks.NewMockCartService(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockPaymentService := mocks.NewMockPaymentService(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)

	service := NewCheckoutService(
		mockCartService,
		mockOrderRepo,
		mockPaymentService,
		mockInventoryRepo,
		mockCustomerRepo,
	)
	ctx := context.Background()

	t.Run("reserves against the order id, not the literal checkout", func(t *testing.T) {
		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "Test",
			LastName:  "Customer",
			Phone:     "+919999900001",
			Addresses: []domain.Address{{ID: "addr_1"}},
		}

		cart := &domain.CartWithItems{
			Cart:  &domain.Cart{Subtotal: 50000},
			Items: []domain.CartItem{{ProductID: "prod_123", Quantity: 2}},
		}

		var reservedRef, createdOrderID string

		mockCustomerRepo.EXPECT().GetByID(gomock.Any(), "cust_123").Return(customer, nil)
		mockCartService.EXPECT().GetCart(gomock.Any(), "cust_123", false).Return(cart, nil)

		mockInventoryRepo.EXPECT().
			ReserveStock(gomock.Any(), "prod_123", 2, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ int, ref string) (*domain.InventoryTransaction, error) {
				reservedRef = ref
				return &domain.InventoryTransaction{}, nil
			})

		mockOrderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				createdOrderID = o.ID
				return nil
			})

		mockPaymentService.EXPECT().
			InitiatePayment(gomock.Any(), gomock.Any()).
			Return(&domain.PaymentResponse{
				PaymentID:     "pay_1",
				RedirectURL:   "https://sandbox.example/pay",
				MerchantTxnID: "txn_1",
			}, nil)

		result, err := service.Initiate(ctx, "cust_123", domain.CheckoutRequest{ShippingAddressID: "addr_1"})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.NotEqual(t, "checkout", reservedRef, "reservation must not use the literal placeholder")
		require.Equal(t, createdOrderID, reservedRef, "reserve and release must share the order id")
	})
}
