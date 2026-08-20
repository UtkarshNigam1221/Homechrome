package service

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

// fakeRefundGateway stands in for PhonePe. Hand-rolled rather than generated:
// the interface lives in another package and only two calls matter here.
type fakeRefundGateway struct {
	initiateResp *phonepe.RefundResponse
	initiateErr  error
	statusResp   *phonepe.RefundStatusResponse
	statusErr    error

	initiatedAmount int64
	initiatedFor    string
}

func (f *fakeRefundGateway) InitiatePayment(context.Context, string, string, int64, string) (string, error) {
	return "", nil
}
func (f *fakeRefundGateway) CheckPaymentStatus(context.Context, string) (*phonepe.StatusResponse, error) {
	return nil, nil
}
func (f *fakeRefundGateway) VerifyWebhookSignature(string, string, string) bool { return true }

func (f *fakeRefundGateway) InitiateRefund(_ context.Context, _, originalMerchantOrderID string, amount int64) (*phonepe.RefundResponse, error) {
	f.initiatedAmount = amount
	f.initiatedFor = originalMerchantOrderID
	if f.initiateErr != nil {
		return nil, f.initiateErr
	}
	return f.initiateResp, nil
}

func (f *fakeRefundGateway) CheckRefundStatus(context.Context, string) (*phonepe.RefundStatusResponse, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.statusResp, nil
}

type refundHarness struct {
	svc       *RefundService
	refunds   *mocks.MockRefundRepository
	orders    *mocks.MockOrderRepository
	payments  *mocks.MockPaymentRepository
	inventory *mocks.MockInventoryRepository
	users     *mocks.MockUserRepository
	auditor   *fakeAuditor
	notifier  *fakeNotifier
	gateway   *fakeRefundGateway
}

// fakeAuditor and fakeNotifier record the one call each that matters, rather
// than pulling generated mocks in for a single verb.
type fakeAuditor struct {
	action, entityID, userID string
	metadata                 map[string]interface{}
	err                      error
	calls                    int
}

func (f *fakeAuditor) Log(_ context.Context, action, _, entityID, userID string,
	_ []domain.FieldChange, metadata map[string]interface{}) error {
	f.calls++
	f.action, f.entityID, f.userID, f.metadata = action, entityID, userID, metadata
	return f.err
}

type fakeNotifier struct {
	trigger domain.NotificationTrigger
	orderID string
	err     error
	calls   int
}

func (f *fakeNotifier) SendOrderNotification(_ context.Context, order *domain.Order,
	trigger domain.NotificationTrigger, _ string) error {
	f.calls++
	f.trigger = trigger
	if order != nil {
		f.orderID = order.ID
	}
	return f.err
}

func newRefundHarness(t *testing.T) *refundHarness {
	t.Helper()
	ctrl := gomock.NewController(t)

	h := &refundHarness{
		refunds:   mocks.NewMockRefundRepository(ctrl),
		orders:    mocks.NewMockOrderRepository(ctrl),
		payments:  mocks.NewMockPaymentRepository(ctrl),
		inventory: mocks.NewMockInventoryRepository(ctrl),
		users:     mocks.NewMockUserRepository(ctrl),
		auditor:   &fakeAuditor{},
		notifier:  &fakeNotifier{},
		gateway: &fakeRefundGateway{
			initiateResp: &phonepe.RefundResponse{RefundID: "OMR1", State: phonepe.RefundStatePending},
		},
	}
	// Audit and notification are optional collaborators; the paths that assert on
	// them supply their own.
	h.svc = NewRefundService(h.refunds, h.orders, h.payments, h.inventory, h.users,
		h.auditor, h.notifier, h.gateway)

	return h
}

func paidOrder(status domain.OrderStatus) *domain.Order {
	return &domain.Order{
		ID: "order_1", CustomerID: "cust_1", Status: status,
		Items: []domain.OrderItem{
			{ID: "item_a", ProductID: "prod_a", UnitPrice: 10000, Quantity: 2},
		},
		Subtotal: 20000, TotalAmount: 20000,
	}
}

func paidPayment() *domain.Payment {
	return &domain.Payment{
		ID: "pay_1", OrderID: "order_1", Status: domain.PaymentStatusPaid,
		MerchantTransactionID: "txn_1",
	}
}

func oneLine(qty int, restock bool) domain.CreateRefundRequest {
	return domain.CreateRefundRequest{
		Reason: domain.RefundReasonOutOfStock,
		Items:  []domain.CreateRefundItemRequest{{OrderItemID: "item_a", Quantity: qty, Restock: restock}},
	}
}

func TestRefundService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("derives the amount and writes the record before calling the provider", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)

		// Written first: a refund that reaches the provider with no local row
		// cannot be reconciled.
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, r *domain.Refund) error {
				require.Equal(t, domain.RefundStatusPending, r.Status)
				require.Equal(t, int64(10000), r.Amount)
				require.NotEmpty(t, r.MerchantRefundID)
				return nil
			})
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), "OMR1").Return(nil)
		h.inventory.EXPECT().WriteOffStock(gomock.Any(), "prod_a", 1, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		refund, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")
		require.NoError(t, err)
		require.Equal(t, int64(10000), refund.Amount)
		require.Equal(t, "txn_1", h.gateway.initiatedFor, "the provider needs the original payment's id")
		require.Equal(t, int64(10000), h.gateway.initiatedAmount)
	})

	t.Run("marks the refund failed and moves no stock when the provider is unreachable", func(t *testing.T) {
		h := newRefundHarness(t)
		h.gateway.initiateErr = stderrors.New("connection refused")

		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().Settle(gomock.Any(), gomock.Any(), domain.RefundStatusFailed,
			gomock.Any(), "INITIATION_FAILED", gomock.Any()).Return(nil)
		// No inventory call registered: gomock fails the test if stock moves.

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")
		require.Error(t, err)
	})

	t.Run("releases stock back to sale when the line is marked restock", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		h.inventory.EXPECT().ReleaseStock(gomock.Any(), "prod_a", 1, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, true), "admin_1")
		require.NoError(t, err)
	})

	// After dispatch the reservation is already consumed, and RETURNED owns
	// restocking — a refund moving stock too would count the goods back twice.
	t.Run("moves no stock for an order already dispatched", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusShipped), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, true), "admin_1")
		require.NoError(t, err, "a post-dispatch refund is money only")
	})

	t.Run("refuses a refund on an unpaid order", func(t *testing.T) {
		h := newRefundHarness(t)
		unpaid := paidPayment()
		unpaid.Status = domain.PaymentStatusPending
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusPending), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(unpaid, nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")
		require.Error(t, err)
	})

	t.Run("refuses more than the line has left", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(5, false), "admin_1")
		require.Error(t, err)
	})

	t.Run("refuses a reason outside the bounded set", func(t *testing.T) {
		h := newRefundHarness(t)
		req := oneLine(1, false)
		req.Reason = "BECAUSE"

		_, err := h.svc.Create(ctx, "order_1", req, "admin_1")
		require.Error(t, err, "an unbounded reason would land in a metric label")
	})
}

func TestRefundService_Settlement(t *testing.T) {
	ctx := context.Background()

	pendingRefund := func() *domain.Refund {
		return &domain.Refund{
			ID: "refund_1", OrderID: "order_1", PaymentID: "pay_1",
			Amount: 10000, Status: domain.RefundStatusPending,
			Reason: domain.RefundReasonOutOfStock,
			Items:  []domain.RefundItem{{OrderItemID: "item_a", ProductID: "prod_a", Quantity: 1, Amount: 10000}},
		}
	}

	t.Run("a completed refund updates the payment, the line and the order", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(pendingRefund(), nil)
		h.refunds.EXPECT().Settle(gomock.Any(), "refund_1", domain.RefundStatusCompleted,
			gomock.Any(), "", "").Return(nil)
		h.payments.EXPECT().AddRefundAmount(gomock.Any(), "pay_1", int64(10000)).Return(int64(10000), nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		// The settled total is recomputed from the order's refunds, not incremented.
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return([]*domain.Refund{pendingRefund()}, nil)

		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				require.Equal(t, 1, o.Items[0].RefundedQuantity)
				// Half the order refunded, so partial — and the rest still ships.
				require.Equal(t, domain.PaymentStatusPartiallyRefunded, o.PaymentStatus)
				require.Equal(t, domain.OrderStatusConfirmed, o.Status, "fulfillment status must not move")
				return nil
			})
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1",
			domain.PaymentStatusPartiallyRefunded, gomock.Any()).Return(nil)

		require.NoError(t, h.svc.HandleRefundCompleted(ctx, "OMR1"))
	})

	t.Run("clearing the order total marks it fully refunded", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(pendingRefund(), nil)
		h.refunds.EXPECT().Settle(gomock.Any(), gomock.Any(), domain.RefundStatusCompleted,
			gomock.Any(), "", "").Return(nil)
		h.payments.EXPECT().AddRefundAmount(gomock.Any(), "pay_1", int64(10000)).Return(int64(20000), nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		// The settled total is recomputed from the order's refunds, not incremented.
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return([]*domain.Refund{pendingRefund()}, nil)

		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				require.Equal(t, domain.PaymentStatusRefunded, o.PaymentStatus)
				return nil
			})
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1",
			domain.PaymentStatusRefunded, gomock.Any()).Return(nil)

		require.NoError(t, h.svc.HandleRefundCompleted(ctx, "OMR1"))
	})

	// PhonePe retries webhooks. The loser of the conditional update must apply
	// nothing, not fail the delivery.
	t.Run("a redelivered webhook applies nothing", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(pendingRefund(), nil)
		h.refunds.EXPECT().Settle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New(errors.ErrCodeConflict, "Refund is no longer pending"))
		// No payment or order call registered: gomock fails if anything applies.

		require.NoError(t, h.svc.HandleRefundCompleted(ctx, "OMR1"),
			"losing the race is the normal outcome of a retry, not an error")
	})

	t.Run("an already-terminal refund is left alone", func(t *testing.T) {
		h := newRefundHarness(t)
		settled := pendingRefund()
		settled.Status = domain.RefundStatusCompleted
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(settled, nil)

		require.NoError(t, h.svc.HandleRefundCompleted(ctx, "OMR1"))
	})

	t.Run("a failed refund touches neither order nor payment", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(pendingRefund(), nil)
		h.refunds.EXPECT().Settle(gomock.Any(), "refund_1", domain.RefundStatusFailed,
			gomock.Any(), "REFUND_FAILED", "INSUFFICIENT_BALANCE").Return(nil)

		require.NoError(t, h.svc.HandleRefundFailed(ctx, "OMR1", "REFUND_FAILED", "INSUFFICIENT_BALANCE"))
	})
}

func TestRefundService_RecheckStatus(t *testing.T) {
	ctx := context.Background()

	// The recovery path: initiation returned nothing, so no provider id was ever
	// stored and the webhook could never find this refund.
	t.Run("records a provider id first seen at re-check", func(t *testing.T) {
		h := newRefundHarness(t)
		h.gateway.statusResp = &phonepe.RefundStatusResponse{
			RefundID: "OMR_late", State: phonepe.RefundStateCompleted,
		}

		refund := &domain.Refund{
			ID: "refund_1", OrderID: "order_1", PaymentID: "pay_1", Amount: 10000,
			Status: domain.RefundStatusPending, MerchantRefundID: "mref_1",
			Items: []domain.RefundItem{{OrderItemID: "item_a", ProductID: "prod_a", Quantity: 1}},
		}

		h.refunds.EXPECT().GetByID(gomock.Any(), "refund_1").Return(refund, nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), "refund_1", "OMR_late").Return(nil)
		h.refunds.EXPECT().Settle(gomock.Any(), "refund_1", domain.RefundStatusCompleted,
			gomock.Any(), "", "").Return(nil)
		h.payments.EXPECT().AddRefundAmount(gomock.Any(), "pay_1", int64(10000)).Return(int64(10000), nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return([]*domain.Refund{refund}, nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1", gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().GetByID(gomock.Any(), "refund_1").Return(refund, nil)

		_, err := h.svc.RecheckStatus(ctx, "refund_1")
		require.NoError(t, err)
	})

	t.Run("leaves a refund the provider still calls pending", func(t *testing.T) {
		h := newRefundHarness(t)
		h.gateway.statusResp = &phonepe.RefundStatusResponse{
			RefundID: "OMR1", State: phonepe.RefundStatePending,
		}
		h.refunds.EXPECT().GetByID(gomock.Any(), "refund_1").Return(&domain.Refund{
			ID: "refund_1", Status: domain.RefundStatusPending,
			MerchantRefundID: "mref_1", ProviderRefundID: "OMR1",
		}, nil)

		got, err := h.svc.RecheckStatus(ctx, "refund_1")
		require.NoError(t, err)
		require.Equal(t, domain.RefundStatusPending, got.Status)
	})

	t.Run("does not re-ask about a refund already settled", func(t *testing.T) {
		h := newRefundHarness(t)
		h.gateway.statusErr = stderrors.New("should not be called")
		h.refunds.EXPECT().GetByID(gomock.Any(), "refund_1").Return(&domain.Refund{
			ID: "refund_1", Status: domain.RefundStatusCompleted,
		}, nil)

		got, err := h.svc.RecheckStatus(ctx, "refund_1")
		require.NoError(t, err)
		require.Equal(t, domain.RefundStatusCompleted, got.Status)
	})
}

// A refund is a money movement, so the record has to say who raised it. created_by
// holds an opaque user id, which is accurate and useless to whoever reads the
// list back.
func TestRefundService_ListByOrder_ResolvesActorNames(t *testing.T) {
	ctx := context.Background()

	refundsBy := func(ids ...string) []*domain.Refund {
		out := make([]*domain.Refund, 0, len(ids))
		for i, id := range ids {
			out = append(out, &domain.Refund{ID: "refund_" + id, CreatedBy: id, Amount: int64(i + 1)})
		}
		return out
	}

	t.Run("names the admin behind each refund", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(refundsBy("usr_1"), nil)
		h.users.EXPECT().GetByID(gomock.Any(), "usr_1").
			Return(&domain.User{ID: "usr_1", FirstName: "Asha", LastName: "Rao"}, nil)

		got, err := h.svc.ListByOrder(ctx, "order_1")

		require.NoError(t, err)
		require.Equal(t, "Asha Rao", got[0].CreatedByName)
	})

	// One lookup per distinct admin, not per row: a run of refunds on one order
	// is usually one person working through it.
	t.Run("looks each admin up once", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").
			Return(refundsBy("usr_1", "usr_1", "usr_1"), nil)
		h.users.EXPECT().GetByID(gomock.Any(), "usr_1").
			Return(&domain.User{ID: "usr_1", FirstName: "Asha", LastName: "Rao"}, nil).Times(1)

		got, err := h.svc.ListByOrder(ctx, "order_1")

		require.NoError(t, err)
		for _, r := range got {
			require.Equal(t, "Asha Rao", r.CreatedByName)
		}
	})

	t.Run("falls back to the email when a user has no name", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(refundsBy("usr_2"), nil)
		h.users.EXPECT().GetByID(gomock.Any(), "usr_2").
			Return(&domain.User{ID: "usr_2", Email: "ops@handloom.com"}, nil)

		got, err := h.svc.ListByOrder(ctx, "order_1")

		require.NoError(t, err)
		require.Equal(t, "ops@handloom.com", got[0].CreatedByName)
	})

	// The refunds are worth showing without the names attached.
	t.Run("still returns the refunds when the directory fails", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(refundsBy("usr_3"), nil)
		h.users.EXPECT().GetByID(gomock.Any(), "usr_3").Return(nil, stderrors.New("dynamo down"))

		got, err := h.svc.ListByOrder(ctx, "order_1")

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Empty(t, got[0].CreatedByName)
	})

	// Webhook-driven settlement has no admin behind it, so there is nothing to
	// look up and no reason to spend a read finding that out.
	t.Run("looks nothing up for a refund with no actor", func(t *testing.T) {
		h := newRefundHarness(t)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").
			Return([]*domain.Refund{{ID: "refund_x"}}, nil)

		got, err := h.svc.ListByOrder(ctx, "order_1")

		require.NoError(t, err)
		require.Empty(t, got[0].CreatedByName)
	})
}

// A refund is PENDING from creation until the provider's webhook lands. Bounding
// the next one on settled figures alone meant the same units could go back
// twice inside that window, and real money left twice.
func TestRefundService_Create_CountsRefundsStillInFlight(t *testing.T) {
	ctx := context.Background()

	inFlight := func(qty int, amount int64) []*domain.Refund {
		return []*domain.Refund{{
			ID: "refund_prior", Status: domain.RefundStatusPending, Amount: amount,
			Items: []domain.RefundItem{{OrderItemID: "item_a", Quantity: qty, Amount: amount}},
		}}
	}

	t.Run("refuses units a pending refund already claimed", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		// The order has 2 units; one is already going back and has not settled.
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(inFlight(2, 20000), nil)
		// No Create, no gateway call: gomock fails the test if money moves.

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")

		require.Error(t, err)
	})

	t.Run("allows what a pending refund leaves", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(inFlight(1, 10000), nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		h.inventory.EXPECT().WriteOffStock(gomock.Any(), "prod_a", 1, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		refund, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")

		require.NoError(t, err)
		require.Equal(t, int64(10000), refund.Amount, "the remaining unit, and the order is now clear")
	})

	// A failed refund returned nothing, so its units are free again.
	t.Run("frees the units of a refund that failed", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return([]*domain.Refund{{
			ID: "refund_dead", Status: domain.RefundStatusFailed, Amount: 20000,
			Items: []domain.RefundItem{{OrderItemID: "item_a", Quantity: 2, Amount: 20000}},
		}}, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		h.inventory.EXPECT().WriteOffStock(gomock.Any(), "prod_a", 2, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(2, false), "admin_1")

		require.NoError(t, err)
	})
}

// A refund is an admin sending money out. The trail has to say who, how much and
// against which lines — and the customer has to be told it is coming back.
func TestRefundService_AuditAndNotify(t *testing.T) {
	ctx := context.Background()

	t.Run("records the refund against the admin who raised it", func(t *testing.T) {
		h := newRefundHarness(t)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		h.inventory.EXPECT().WriteOffStock(gomock.Any(), "prod_a", 1, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		refund, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")

		require.NoError(t, err)
		require.Equal(t, 1, h.auditor.calls)
		require.Equal(t, "refund.create", h.auditor.action)
		require.Equal(t, refund.ID, h.auditor.entityID)
		require.Equal(t, "admin_1", h.auditor.userID)
		require.Equal(t, int64(10000), h.auditor.metadata["amount_paise"])
		require.Len(t, h.auditor.metadata["items"], 1, "the trail names the lines, not just the total")
	})

	// The money is already on its way; a lost trail is not a reason to report it
	// as failed.
	t.Run("a failed audit does not fail the refund", func(t *testing.T) {
		h := newRefundHarness(t)
		h.auditor.err = stderrors.New("audit table down")
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.payments.EXPECT().GetByOrderID(gomock.Any(), "order_1").Return(paidPayment(), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return(nil, nil)
		h.refunds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		h.refunds.EXPECT().SetProviderRefundID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		h.inventory.EXPECT().WriteOffStock(gomock.Any(), "prod_a", 1, "order_1").
			Return(&domain.InventoryTransaction{}, nil)

		_, err := h.svc.Create(ctx, "order_1", oneLine(1, false), "admin_1")

		require.NoError(t, err)
	})

	t.Run("tells the customer once the money has gone back", func(t *testing.T) {
		h := newRefundHarness(t)
		refund := &domain.Refund{
			ID: "refund_1", OrderID: "order_1", PaymentID: "pay_1", Amount: 10000,
			Status: domain.RefundStatusPending,
			Items:  []domain.RefundItem{{OrderItemID: "item_a", Quantity: 1, Amount: 10000}},
		}
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(refund, nil)
		h.refunds.EXPECT().Settle(gomock.Any(), "refund_1", domain.RefundStatusCompleted,
			gomock.Any(), "", "").Return(nil)
		h.payments.EXPECT().AddRefundAmount(gomock.Any(), "pay_1", int64(10000)).Return(int64(10000), nil)
		h.orders.EXPECT().GetByID(gomock.Any(), "order_1").Return(paidOrder(domain.OrderStatusConfirmed), nil)
		h.refunds.EXPECT().ListByOrder(gomock.Any(), "order_1").Return([]*domain.Refund{refund}, nil)
		h.orders.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		h.payments.EXPECT().UpdateStatus(gomock.Any(), "pay_1", gomock.Any(), gomock.Any()).Return(nil)

		require.NoError(t, h.svc.HandleRefundCompleted(ctx, "OMR1"))

		require.Equal(t, 1, h.notifier.calls)
		require.Equal(t, domain.NotificationTriggerRefund, h.notifier.trigger)
		require.Equal(t, "order_1", h.notifier.orderID)
	})

	// A refund that failed at the provider sent nothing back.
	t.Run("says nothing to the customer when the refund failed", func(t *testing.T) {
		h := newRefundHarness(t)
		refund := &domain.Refund{ID: "refund_1", OrderID: "order_1", PaymentID: "pay_1",
			Amount: 10000, Status: domain.RefundStatusPending}
		h.refunds.EXPECT().GetByProviderRefundID(gomock.Any(), "OMR1").Return(refund, nil)
		h.refunds.EXPECT().Settle(gomock.Any(), "refund_1", domain.RefundStatusFailed,
			gomock.Any(), "E", "D").Return(nil)

		require.NoError(t, h.svc.HandleRefundFailed(ctx, "OMR1", "E", "D"))

		require.Zero(t, h.notifier.calls)
	})
}
