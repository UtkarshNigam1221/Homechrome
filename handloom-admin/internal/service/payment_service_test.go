package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/mocks"
)

// fakePaymentGateway stands in for PhonePe. Only the status call matters here.
type fakePaymentGateway struct {
	statusResp *phonepe.StatusResponse
	calls      int
}

func (f *fakePaymentGateway) InitiatePayment(context.Context, string, string, int64, string) (string, error) {
	return "", nil
}

func (f *fakePaymentGateway) CheckPaymentStatus(context.Context, string) (*phonepe.StatusResponse, error) {
	f.calls++
	return f.statusResp, nil
}

func (f *fakePaymentGateway) VerifyWebhookSignature(string, string, string) bool { return true }

func (f *fakePaymentGateway) InitiateRefund(context.Context, string, string, int64) (*phonepe.RefundResponse, error) {
	return nil, nil
}

func (f *fakePaymentGateway) CheckRefundStatus(context.Context, string) (*phonepe.RefundStatusResponse, error) {
	return nil, nil
}

type paymentHarness struct {
	svc       *PaymentService
	payments  *mocks.MockPaymentRepository
	orders    *mocks.MockOrderRepository
	inventory *mocks.MockInventoryRepository
	carts     *mocks.MockCartService
	customers *mocks.MockCustomerRepository
	gateway   *fakePaymentGateway
}

func newPaymentHarness(t *testing.T, state string) *paymentHarness {
	t.Helper()
	ctrl := gomock.NewController(t)

	h := &paymentHarness{
		payments:  mocks.NewMockPaymentRepository(ctrl),
		orders:    mocks.NewMockOrderRepository(ctrl),
		inventory: mocks.NewMockInventoryRepository(ctrl),
		carts:     mocks.NewMockCartService(ctrl),
		customers: mocks.NewMockCustomerRepository(ctrl),
		gateway: &fakePaymentGateway{statusResp: &phonepe.StatusResponse{
			OrderID: "OMO1", State: state, Amount: 20000,
			PaymentDetails: []phonepe.PaymentDetail{{TransactionID: "T1", PaymentMode: "UPI_INTENT"}},
		}},
	}
	h.svc = NewPaymentService(h.payments, h.orders, h.inventory, h.carts, h.customers, h.gateway)
	return h
}

func abandonedPayment() *domain.Payment {
	return &domain.Payment{
		ID: "pay_1", OrderID: "order_1", CustomerID: "cust_1",
		Status: domain.PaymentStatusInitiated, MerchantTransactionID: "HC-1",
	}
}

func pendingOrder() *domain.Order {
	return &domain.Order{
		ID: "order_1", CustomerID: "cust_1",
		Status: domain.OrderStatusPending, PaymentStatus: domain.PaymentStatusPending,
		Items: []domain.OrderItem{{ID: "item_a", ProductID: "prod_a", UnitPrice: 10000, Quantity: 2}},
	}
}

func TestCheckProviderStatus(t *testing.T) {
	ctx := context.Background()

	// The reported bug: the customer walked away from the PhonePe page, so no
	// webhook ever came and nothing but this call can settle the payment.
	t.Run("applies a provider failure to the payment and the order", func(t *testing.T) {
		h := newPaymentHarness(t, phonepe.PaymentStateFailed)

		failed := abandonedPayment()
		failed.Status = domain.PaymentStatusFailed

		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(abandonedPayment(), nil)
		h.payments.EXPECT().GetByMerchantTxnID(gomock.Any(), "HC-1").Return(abandonedPayment(), nil)
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1", domain.PaymentStatusFailed, gomock.Any()).Return(nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(pendingOrder(), nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				require.Equal(t, domain.PaymentStatusFailed, o.PaymentStatus)
				// Canceling is the admin's call, so the order itself does not move.
				require.Equal(t, domain.OrderStatusPending, o.Status)
				return nil
			})
		h.inventory.EXPECT().ReleaseOrderStock(gomock.Any(), "order_1", map[string]int{"prod_a": 2}).Return(nil)
		// Re-read so the response carries the settled status, not the stale one.
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(failed, nil)
		// The sync then reads back what the handler wrote and finds nothing to do.
		failedOrder := pendingOrder()
		failedOrder.PaymentStatus = domain.PaymentStatusFailed
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(failedOrder, nil)

		result, err := h.svc.CheckProviderStatus(ctx, "order_1")
		require.NoError(t, err)
		require.Equal(t, phonepe.PaymentStateFailed, result.ProviderState)
		require.Equal(t, string(domain.PaymentStatusFailed), result.LocalStatus)
	})

	t.Run("applies a provider success by confirming the order", func(t *testing.T) {
		h := newPaymentHarness(t, phonepe.PaymentStateCompleted)

		paid := abandonedPayment()
		paid.Status = domain.PaymentStatusSuccess

		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(abandonedPayment(), nil)
		h.payments.EXPECT().GetByMerchantTxnID(gomock.Any(), "HC-1").Return(abandonedPayment(), nil)
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1", domain.PaymentStatusSuccess, gomock.Any()).Return(nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(pendingOrder(), nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				require.Equal(t, domain.OrderStatusConfirmed, o.Status)
				require.Equal(t, domain.PaymentStatusPaid, o.PaymentStatus)
				return nil
			})
		h.carts.EXPECT().ClearCart(gomock.Any(), "cust_1").Return(nil)
		h.customers.EXPECT().GetByID(gomock.Any(), "cust_1").Return(&domain.Customer{ID: "cust_1"}, nil).AnyTimes()
		h.customers.EXPECT().RecordPurchase(gomock.Any(), "cust_1", gomock.Any()).Return(int64(1), nil).AnyTimes()
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paid, nil)
		paidOrder := pendingOrder()
		paidOrder.Status, paidOrder.PaymentStatus = domain.OrderStatusConfirmed, domain.PaymentStatusPaid
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder, nil)

		result, err := h.svc.CheckProviderStatus(ctx, "order_1")
		require.NoError(t, err)
		require.Equal(t, string(domain.PaymentStatusSuccess), result.LocalStatus)
	})

	// Re-checking a settled payment must not release stock or move money twice.
	t.Run("writes nothing when payment and order already agree", func(t *testing.T) {
		h := newPaymentHarness(t, phonepe.PaymentStateFailed)

		failed := abandonedPayment()
		failed.Status = domain.PaymentStatusFailed
		agreed := pendingOrder()
		agreed.PaymentStatus = domain.PaymentStatusFailed

		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(failed, nil).Times(2)
		h.payments.EXPECT().GetByMerchantTxnID(gomock.Any(), "HC-1").Return(failed, nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(agreed, nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
		h.inventory.EXPECT().ReleaseOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		result, err := h.svc.CheckProviderStatus(ctx, "order_1")
		require.NoError(t, err)
		require.Equal(t, string(domain.PaymentStatusFailed), result.LocalStatus)
	})

	// The reported regression: the pre-fix webhook settled the payment and never
	// touched the order, so the handlers now short-circuit as already-processed and
	// the order was left reading PENDING however many times an admin re-checked.
	t.Run("repairs an order the old failure path left behind", func(t *testing.T) {
		h := newPaymentHarness(t, phonepe.PaymentStateFailed)

		failed := abandonedPayment()
		failed.Status = domain.PaymentStatusFailed

		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(failed, nil).Times(2)
		h.payments.EXPECT().GetByMerchantTxnID(gomock.Any(), "HC-1").Return(failed, nil)
		// Order still carries the stale copy.
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(pendingOrder(), nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				require.Equal(t, domain.PaymentStatusFailed, o.PaymentStatus)
				// Still not a cancellation: that stays the admin's call.
				require.Equal(t, domain.OrderStatusPending, o.Status)
				return nil
			})
		// The handler already released the stock when it first settled the payment.
		h.inventory.EXPECT().ReleaseOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		result, err := h.svc.CheckProviderStatus(ctx, "order_1")
		require.NoError(t, err)
		require.Equal(t, string(domain.PaymentStatusFailed), result.LocalStatus)
	})

	// A payment the customer may still be paying must stay untouched.
	t.Run("leaves a still-pending provider state alone", func(t *testing.T) {
		h := newPaymentHarness(t, "PENDING")

		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(abandonedPayment(), nil).Times(2)

		result, err := h.svc.CheckProviderStatus(ctx, "order_1")
		require.NoError(t, err)
		require.Equal(t, string(domain.PaymentStatusInitiated), result.LocalStatus)
	})
}
