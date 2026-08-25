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

func ptr(s string) *string { return &s }

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
	mockCouponService := mocks.NewMockCouponService(ctrl)

	service := NewCheckoutService(
		mockCartService,
		mockOrderRepo,
		mockPaymentService,
		mockInventoryRepo,
		mockCustomerRepo,
		mockCouponService,
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
			ReserveOrderStock(gomock.Any(), gomock.Any(), map[string]int{"prod_123": 2}).
			DoAndReturn(func(_ context.Context, ref string, _ map[string]int) error {
				reservedRef = ref
				return nil
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

// checkoutFixture wires a CheckoutService whose every dependency succeeds, so each test
// only has to state the one thing it is about. The cart is two lines, ₹1,000 and ₹2,000.
type checkoutFixture struct {
	svc        *CheckoutService
	couponSvc  *mocks.MockCouponService
	savedOrder *domain.Order
}

func newCheckoutFixture(t *testing.T) *checkoutFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	cartSvc := mocks.NewMockCartService(ctrl)
	orderRepo := mocks.NewMockOrderRepository(ctrl)
	paymentSvc := mocks.NewMockPaymentService(ctrl)
	inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	couponSvc := mocks.NewMockCouponService(ctrl)

	f := &checkoutFixture{couponSvc: couponSvc}

	customerRepo.EXPECT().GetByID(gomock.Any(), "cust_1").Return(&domain.Customer{
		ID: "cust_1", FirstName: "Test", LastName: "Buyer", Phone: "+919000000000",
		Addresses: []domain.Address{{ID: "addr_1", City: "Mumbai", Country: "India"}},
	}, nil).AnyTimes()

	cartSvc.EXPECT().GetCart(gomock.Any(), "cust_1", false).Return(&domain.CartWithItems{
		Cart: &domain.Cart{Subtotal: 300000},
		Items: []domain.CartItem{
			{ProductID: "p1", ProductName: "A", Quantity: 1, UnitPrice: 100000, TotalPrice: 100000},
			{ProductID: "p2", ProductName: "B", Quantity: 1, UnitPrice: 200000, TotalPrice: 200000},
		},
	}, nil).AnyTimes()

	inventoryRepo.EXPECT().ReserveOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// Capture what was persisted — that is what every assertion below inspects.
	orderRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, o *domain.Order) error {
			f.savedOrder = o
			return nil
		}).AnyTimes()

	paymentSvc.EXPECT().InitiatePayment(gomock.Any(), gomock.Any()).
		Return(&domain.PaymentResponse{
			RedirectURL: "https://pay.example/x", MerchantTxnID: "txn_1",
		}, nil).AnyTimes()

	f.svc = NewCheckoutService(cartSvc, orderRepo, paymentSvc, inventoryRepo, customerRepo, couponSvc)
	return f
}

// A coupon reduces the total, and the reduction lands on the lines — the order-level
// figure is their sum, not the other way round.
func TestCheckoutService_AppliesCoupon(t *testing.T) {
	f := newCheckoutFixture(t)
	f.couponSvc.EXPECT().
		Validate(gomock.Any(), "FESTIVE20", gomock.Any()).
		Return(&domain.CouponValidationResult{
			Valid: true, CouponID: "coupon_1", Code: "FESTIVE20", DiscountAmount: 30000,
		}, nil)

	result, err := f.svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1",
		CouponCode:        ptr("FESTIVE20"),
	})
	require.NoError(t, err)

	order := result.Order
	require.Equal(t, int64(300000), order.Subtotal)
	require.Equal(t, int64(30000), order.DiscountAmount)
	require.Equal(t, int64(270000), order.TotalAmount)
	require.True(t, order.DiscountAllocated, "the lines are authoritative")

	var lineSum int64
	for _, item := range order.Items {
		lineSum += item.DiscountAmount
	}
	require.Equal(t, order.DiscountAmount, lineSum, "line discounts must sum to the order's")

	// Tax is extracted from the discounted total, never added to it.
	require.Equal(t, extractTax(270000), order.TaxAmount)
}

// The cart total the coupon sees must be the cart's, not something the client sent.
func TestCheckoutService_ValidatesAgainstTheServerCartTotal(t *testing.T) {
	f := newCheckoutFixture(t)
	f.couponSvc.EXPECT().
		Validate(gomock.Any(), "FESTIVE20", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, cc domain.CouponContext) (*domain.CouponValidationResult, error) {
			require.Equal(t, int64(300000), cc.CartTotal)
			require.Equal(t, "cust_1", cc.CustomerID)
			require.False(t, cc.HasAutomaticOffer, "Phase 1 has no automatic offers")
			return &domain.CouponValidationResult{Valid: true, CouponID: "c1", Code: "FESTIVE20", DiscountAmount: 1}, nil
		})

	_, err := f.svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1", CouponCode: ptr("FESTIVE20"),
	})
	require.NoError(t, err)
}

// A coupon problem must never cost the sale.
func TestCheckoutService_InvalidCouponStillPlacesTheOrder(t *testing.T) {
	f := newCheckoutFixture(t)
	f.couponSvc.EXPECT().
		Validate(gomock.Any(), "EXPIRED", gomock.Any()).
		Return(&domain.CouponValidationResult{
			Valid: false, Code: "EXPIRED", ErrorMessage: "This coupon has expired",
		}, nil)

	result, err := f.svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1",
		CouponCode:        ptr("EXPIRED"),
	})
	require.NoError(t, err, "an expired code must not fail checkout")
	require.Equal(t, int64(0), result.Order.DiscountAmount)
	require.Equal(t, result.Order.Subtotal, result.Order.TotalAmount)
	require.Contains(t, result.CouponNotice, "expired")
}

// A coupon lookup that errors is not the customer's problem either.
func TestCheckoutService_CouponErrorStillPlacesTheOrder(t *testing.T) {
	f := newCheckoutFixture(t)
	f.couponSvc.EXPECT().
		Validate(gomock.Any(), "BOOM", gomock.Any()).
		Return(nil, errors.Internal("dynamo is down"))

	result, err := f.svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1", CouponCode: ptr("BOOM"),
	})
	require.NoError(t, err, "a coupon lookup failure must not cost the sale")
	require.Equal(t, int64(0), result.Order.DiscountAmount)
	require.NotEmpty(t, result.CouponNotice)
}

// No code submitted must behave exactly as it did before coupons existed.
func TestCheckoutService_NoCouponIsUnchanged(t *testing.T) {
	f := newCheckoutFixture(t)
	// No Validate expectation: gomock fails the test if it is called.

	result, err := f.svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1",
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), result.Order.DiscountAmount)
	require.Equal(t, result.Order.Subtotal, result.Order.TotalAmount)
	require.Empty(t, result.CouponNotice)
}

// If allocateDiscount cannot honor the discount, the checkout must abort and release
// whatever stock it already reserved. Writing the order anyway would leave lines that
// disagree with the total — permanently un-refundable, since every later refund reads
// the mismatch and rejects itself. Recipe: a cart subtotal the coupon prices against,
// but zero-priced lines, so allocateDiscount has no line value to carry the discount.
func TestCheckoutService_AllocationFailureReleasesStockAndAborts(t *testing.T) {
	ctrl := gomock.NewController(t)

	cartSvc := mocks.NewMockCartService(ctrl)
	orderRepo := mocks.NewMockOrderRepository(ctrl)
	paymentSvc := mocks.NewMockPaymentService(ctrl)
	inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	couponSvc := mocks.NewMockCouponService(ctrl)

	customerRepo.EXPECT().GetByID(gomock.Any(), "cust_1").Return(&domain.Customer{
		ID: "cust_1", FirstName: "Test", LastName: "Buyer",
		Addresses: []domain.Address{{ID: "addr_1"}},
	}, nil)

	cartSvc.EXPECT().GetCart(gomock.Any(), "cust_1", false).Return(&domain.CartWithItems{
		Cart:  &domain.Cart{Subtotal: 300000},
		Items: []domain.CartItem{{ProductID: "p1", Quantity: 1, UnitPrice: 0, TotalPrice: 0}},
	}, nil)

	couponSvc.EXPECT().Validate(gomock.Any(), "FESTIVE20", gomock.Any()).Return(&domain.CouponValidationResult{
		Valid: true, CouponID: "c1", Code: "FESTIVE20", DiscountAmount: 30000,
	}, nil)

	var reservedOrderID, releasedOrderID string
	inventoryRepo.EXPECT().ReserveOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, orderID string, _ map[string]int) error {
			reservedOrderID = orderID
			return nil
		})
	inventoryRepo.EXPECT().ReleaseOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, orderID string, _ map[string]int) error {
			releasedOrderID = orderID
			return nil
		})
	// No EXPECT on orderRepo.Create or paymentSvc.InitiatePayment: gomock fails the
	// test if either is called — an allocation failure must never reach them.

	svc := NewCheckoutService(cartSvc, orderRepo, paymentSvc, inventoryRepo, customerRepo, couponSvc)

	result, err := svc.Initiate(context.Background(), "cust_1", domain.CheckoutRequest{
		ShippingAddressID: "addr_1", CouponCode: ptr("FESTIVE20"),
	})

	require.Error(t, err, "an allocation failure must abort the checkout, not price around it")
	require.Nil(t, result)
	require.NotEmpty(t, releasedOrderID, "the reservation must be released before aborting")
	require.Equal(t, reservedOrderID, releasedOrderID, "release must free the same reservation that was taken")
}

// The service method itself must read the cart server-side, same as Initiate — there
// is no client-supplied total anywhere to accidentally start trusting.
func TestCheckoutService_PreviewCoupon(t *testing.T) {
	f := newCheckoutFixture(t)
	f.couponSvc.EXPECT().
		Validate(gomock.Any(), "FESTIVE20", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, cc domain.CouponContext) (*domain.CouponValidationResult, error) {
			require.Equal(t, int64(300000), cc.CartTotal)
			require.Equal(t, "cust_1", cc.CustomerID)
			require.False(t, cc.HasAutomaticOffer)
			return &domain.CouponValidationResult{
				Valid: true, CouponID: "c1", Code: "FESTIVE20", DiscountAmount: 30000,
			}, nil
		})

	got, err := f.svc.PreviewCoupon(context.Background(), "cust_1", "FESTIVE20")
	require.NoError(t, err)
	require.True(t, got.Valid)
	require.Equal(t, int64(30000), got.DiscountAmount)
}
