package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/metrics"
)

// capturingPublisher keeps every flushed event so a test can total one metric.
type capturingPublisher struct{ events []metrics.Event }

func (p *capturingPublisher) Publish(_ context.Context, evs []metrics.Event) error {
	p.events = append(p.events, evs...)
	return nil
}

// captureMetrics runs fn against a context carrying a metrics buffer and returns
// the summed sum_value per metric name. Count-only metrics total 0.
func captureMetrics(t *testing.T, fn func(ctx context.Context)) map[string]int64 {
	t.Helper()
	pub := &capturingPublisher{}
	metrics.SetDefault(pub)
	t.Cleanup(func() { metrics.SetDefault(metrics.NoopPublisher{}) })

	ctx := metrics.WithBuffer(context.Background())
	fn(ctx)
	require.NoError(t, metrics.Flush(ctx))

	totals := make(map[string]int64)
	for _, ev := range pub.events {
		totals[ev.Metric] += ev.SumValue
	}
	return totals
}

// orders_value is what the admin Geography dashboard renders as revenue. An
// order that is only placed has not paid for anything, so a checkout the
// customer abandons at the PhonePe page must contribute nothing to it.
func TestOrdersValueCountsPaidOrdersOnly(t *testing.T) {
	t.Run("checkout initiate books no revenue", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		carts := mocks.NewMockCartService(ctrl)
		orders := mocks.NewMockOrderRepository(ctrl)
		payments := mocks.NewMockPaymentService(ctrl)
		inventory := mocks.NewMockInventoryRepository(ctrl)
		customers := mocks.NewMockCustomerRepository(ctrl)

		svc := NewCheckoutService(carts, orders, payments, inventory, customers)

		customers.EXPECT().GetByID(gomock.Any(), "cust_1").Return(&domain.Customer{
			ID: "cust_1", FirstName: "Test", LastName: "Customer",
			Addresses: []domain.Address{{ID: "addr_1"}},
		}, nil)
		carts.EXPECT().GetCart(gomock.Any(), "cust_1", false).Return(&domain.CartWithItems{
			Cart:  &domain.Cart{Subtotal: 50000},
			Items: []domain.CartItem{{ProductID: "prod_a", Quantity: 2}},
		}, nil)
		inventory.EXPECT().ReserveOrderStock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		orders.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		payments.EXPECT().InitiatePayment(gomock.Any(), gomock.Any()).
			Return(&domain.PaymentResponse{PaymentID: "pay_1", MerchantTxnID: "HC-1"}, nil)

		totals := captureMetrics(t, func(ctx context.Context) {
			_, err := svc.Initiate(ctx, "cust_1", domain.CheckoutRequest{ShippingAddressID: "addr_1"})
			require.NoError(t, err)
		})

		require.Zero(t, totals["orders_value"],
			"an unpaid order must not be counted as revenue")
	})

	t.Run("payment success books the order total", func(t *testing.T) {
		h := newPaymentHarness(t, phonepe.PaymentStateCompleted)

		paidOrder := pendingOrder()
		paidOrder.TotalAmount = 20000
		paidOrder.Country, paidOrder.City = "IN", "Jaipur"

		h.payments.EXPECT().GetByMerchantTxnID(gomock.Any(), "HC-1").Return(abandonedPayment(), nil)
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1", domain.PaymentStatusSuccess, gomock.Any()).Return(nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder, nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		h.carts.EXPECT().ClearCart(gomock.Any(), "cust_1").Return(nil)
		h.customers.EXPECT().RecordPurchase(gomock.Any(), "cust_1", gomock.Any()).Return(int64(1), nil)

		totals := captureMetrics(t, func(ctx context.Context) {
			require.NoError(t, h.svc.HandlePaymentSuccess(ctx, domain.PaymentWebhookEvent{
				MerchantTxnID: "HC-1", TransactionID: "T1", PaymentMode: "UPI_INTENT",
			}))
		})

		require.Equal(t, int64(20000), totals["orders_value"],
			"a paid order must book its total exactly once")
	})
}
