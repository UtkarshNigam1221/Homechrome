package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
)

// refundAuditor records who sent money back. Declared here rather than taken as
// the whole audit service: this needs one verb.
//
// Nil in the store Lambda, which settles refunds from webhooks and raises none.
type refundAuditor interface {
	Log(ctx context.Context, action, entityType, entityID, userID string,
		changes []domain.FieldChange, metadata map[string]interface{}) error
}

// refundNotifier tells the customer their money is on its way back. Nil where
// there is no notification service to reach.
type refundNotifier interface {
	SendOrderNotification(ctx context.Context, order *domain.Order,
		trigger domain.NotificationTrigger, createdBy string) error
}

// RefundService owns the refund lifecycle: deriving the amount, moving the
// money, and settling asynchronously.
//
// Its own service rather than more of OrderService, which is already the
// largest here — refunds have a lifecycle of their own, spanning two Lambdas.
type RefundService struct {
	refundRepo    domain.RefundRepository
	orderRepo     domain.OrderRepository
	paymentRepo   domain.PaymentRepository
	inventoryRepo domain.InventoryRepository
	userRepo      domain.UserRepository
	auditor       refundAuditor
	notifier      refundNotifier
	gateway       phonepe.Gateway
}

// NewRefundService creates a new RefundService.
func NewRefundService(
	refundRepo domain.RefundRepository,
	orderRepo domain.OrderRepository,
	paymentRepo domain.PaymentRepository,
	inventoryRepo domain.InventoryRepository,
	userRepo domain.UserRepository,
	auditor refundAuditor,
	notifier refundNotifier,
	gateway phonepe.Gateway,
) *RefundService {
	return &RefundService{
		refundRepo:    refundRepo,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
		auditor:       auditor,
		notifier:      notifier,
		gateway:       gateway,
	}
}

// Create derives the amount, records the refund, then asks the provider for it.
//
// The record is written before the provider is called, deliberately. A refund
// that leaves the building with no local row is unreconcilable; a PENDING row
// whose gateway call never happened is recoverable through RecheckStatus.
func (s *RefundService) Create(ctx context.Context, orderID string, req domain.CreateRefundRequest, createdBy string) (*domain.Refund, error) {
	if !req.Reason.IsValid() {
		return nil, errors.BadRequest("Unknown refund reason")
	}

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, errors.BadRequest("Order has no payment to refund")
	}
	if payment.Status != domain.PaymentStatusPaid && payment.Status != domain.PaymentStatusSuccess &&
		payment.Status != domain.PaymentStatusPartiallyRefunded {
		return nil, errors.BadRequest("Order has not been paid")
	}

	// What every refund that has not failed already claims — settled or still in
	// flight. Bounding on the settled figures alone left a window, between
	// creating a refund and its webhook landing, in which the same units could go
	// back a second time and real money left twice.
	existing, err := s.refundRepo.ListByOrder(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	claimed, claimedAmount := claimedByLive(existing)
	if claimedAmount < payment.RefundAmount {
		// The payment total is authoritative for money; a refund whose settlement
		// half-completed can leave it ahead of the rows.
		claimedAmount = payment.RefundAmount
	}

	breakdown, err := deriveRefundAmount(order, req.Items, claimed, claimedAmount)
	if err != nil {
		return nil, err
	}

	refund := &domain.Refund{
		ID:               "refund_" + uuid.New().String()[:8],
		OrderID:          order.ID,
		PaymentID:        payment.ID,
		CustomerID:       order.CustomerID,
		Amount:           breakdown.Total,
		Status:           domain.RefundStatusPending,
		Reason:           req.Reason,
		Items:            breakdown.Items,
		MerchantRefundID: "mref_" + uuid.New().String(),
		InitiatedAt:      time.Now(),
		CreatedBy:        createdBy,
	}

	if createErr := s.refundRepo.Create(ctx, refund); createErr != nil {
		return nil, createErr
	}

	resp, err := s.gateway.InitiateRefund(ctx, refund.MerchantRefundID, payment.MerchantTransactionID, refund.Amount)
	if err != nil {
		// The provider never accepted it, so no money moves and no stock does
		// either. Recording the failure keeps the row honest rather than leaving
		// it PENDING forever.
		slog.ErrorContext(ctx, "Refund initiation failed", "refund_id", refund.ID, "error", err)
		if settleErr := s.refundRepo.Settle(ctx, refund.ID, domain.RefundStatusFailed, time.Now(),
			"INITIATION_FAILED", err.Error()); settleErr != nil {
			slog.ErrorContext(ctx, "Failed to mark refund failed", "refund_id", refund.ID, "error", settleErr)
		}
		metrics.Record(ctx, "refund_failed", metrics.L{metrics.LabelReason: string(refund.Reason)})
		return nil, errors.Wrap(err, "Failed to initiate refund")
	}

	if err := s.refundRepo.SetProviderRefundID(ctx, refund.ID, resp.RefundID); err != nil {
		// The refund is live at the provider either way; losing the id only
		// costs us the webhook path, and RecheckStatus keys on ours.
		slog.ErrorContext(ctx, "Failed to record provider refund id", "refund_id", refund.ID, "error", err)
	}
	refund.ProviderRefundID = resp.RefundID

	s.applyInventoryEffect(ctx, order, refund)
	s.audit(ctx, refund)

	metrics.Record(ctx, "refund_initiated", metrics.L{
		metrics.LabelReason:  string(refund.Reason),
		metrics.LabelGateway: gatewayPhonePe,
	})
	slog.InfoContext(ctx, "Refund initiated",
		"refund_id", refund.ID, "order_id", order.ID, "amount_paise", refund.Amount)

	return refund, nil
}

// audit records who sent the money back, with the lines it covered. Logged and
// swallowed on failure: the refund is already live at the provider, and losing
// the trail is not a reason to report the refund as failed.
func (s *RefundService) audit(ctx context.Context, refund *domain.Refund) {
	if s.auditor == nil {
		return
	}

	lines := make([]map[string]interface{}, 0, len(refund.Items))
	for _, item := range refund.Items {
		lines = append(lines, map[string]interface{}{
			"order_item_id": item.OrderItemID,
			"product_id":    item.ProductID,
			"quantity":      item.Quantity,
			"amount_paise":  item.Amount,
			"restock":       item.Restock,
		})
	}

	if err := s.auditor.Log(ctx, "refund.create", "REFUND", refund.ID, refund.CreatedBy, nil,
		map[string]interface{}{
			"order_id":     refund.OrderID,
			"payment_id":   refund.PaymentID,
			"amount_paise": refund.Amount,
			"reason":       string(refund.Reason),
			"items":        lines,
		}); err != nil {
		slog.ErrorContext(ctx, "Failed to audit refund", "refund_id", refund.ID, "error", err)
	}
}

// notifyCustomer tells the customer the money is on its way back. Best-effort
// for the same reason the rest of applyCompletion is: it has already gone.
func (s *RefundService) notifyCustomer(ctx context.Context, order *domain.Order, refund *domain.Refund) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.SendOrderNotification(ctx, order, domain.NotificationTriggerRefund, refund.CreatedBy); err != nil {
		slog.ErrorContext(ctx, "Failed to notify the customer of a refund",
			"refund_id", refund.ID, "error", err)
	}
}

// claimedByLive totals what refunds that have not failed already account for,
// per line and in money. A failed refund returns nothing, so its units are free
// again.
func claimedByLive(refunds []*domain.Refund) (map[string]int, int64) {
	claimed := make(map[string]int)
	var amount int64

	for _, refund := range refunds {
		if refund.Status == domain.RefundStatusFailed {
			continue
		}
		amount += refund.Amount
		for _, item := range refund.Items {
			claimed[item.OrderItemID] += item.Quantity
		}
	}

	return claimed, amount
}

// applyInventoryEffect moves stock for a refund, and only while the order still
// holds a reservation to move.
//
// Dispatch is one dividing line: CommitStock consumes the reservation at
// SHIPPED, and RETURNED owns restocking from there, so a refund moving stock too
// would count the same goods back twice. Cancellation is the other: the release
// already happened, the units are back on sale, and a refund that tried to
// release them again would be refused by the outstanding-reservation guard and
// fire inventory_mutation_failed on what is a perfectly ordinary flow.
func (s *RefundService) applyInventoryEffect(ctx context.Context, order *domain.Order, refund *domain.Refund) {
	switch order.Status {
	case domain.OrderStatusShipped, domain.OrderStatusDelivered,
		domain.OrderStatusReturned, domain.OrderStatusCancelled:
		return
	}

	for _, item := range refund.Items {
		var err error
		if item.Restock {
			// Back on sale: the reservation is released and nothing else moves.
			_, err = s.inventoryRepo.ReleaseRefundedStock(ctx, item.ProductID, item.Quantity, order.ID, refund.ID)
		} else {
			// Written off: the goods are not there, so the reservation goes and
			// on-hand falls with it.
			_, err = s.inventoryRepo.WriteOffStock(ctx, item.ProductID, item.Quantity, order.ID, refund.ID)
		}
		if err != nil {
			slog.ErrorContext(ctx, "Failed to move stock for refund",
				keyProductID, item.ProductID, "refund_id", refund.ID, "error", err)
			metrics.Record(ctx, "inventory_mutation_failed",
				metrics.L{metrics.LabelReason: refundInventoryReason(item.Restock)})
		}
	}
}

func refundInventoryReason(restock bool) string {
	if restock {
		return reasonRelease
	}
	return reasonWriteOff
}

// ListByOrder returns an order's refunds, with the admin behind each one named.
func (s *RefundService) ListByOrder(ctx context.Context, orderID string) ([]*domain.Refund, error) {
	refunds, err := s.refundRepo.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(refunds))
	for _, refund := range refunds {
		ids = append(ids, refund.CreatedBy)
	}

	names := resolveActorNames(ctx, s.userRepo, ids)
	for _, refund := range refunds {
		refund.CreatedByName = names[refund.CreatedBy]
	}

	return refunds, nil
}

// HandleRefundCompleted settles a refund from a pg.refund.completed webhook.
func (s *RefundService) HandleRefundCompleted(ctx context.Context, providerRefundID string) error {
	refund, err := s.refundRepo.GetByProviderRefundID(ctx, providerRefundID)
	if err != nil {
		return err
	}
	return s.settle(ctx, refund, domain.RefundStatusCompleted, "", "")
}

// HandleRefundFailed settles a refund from a pg.refund.failed webhook.
func (s *RefundService) HandleRefundFailed(ctx context.Context, providerRefundID, errorCode, detailedErrorCode string) error {
	refund, err := s.refundRepo.GetByProviderRefundID(ctx, providerRefundID)
	if err != nil {
		return err
	}
	return s.settle(ctx, refund, domain.RefundStatusFailed, errorCode, detailedErrorCode)
}

// RecheckStatus asks the provider directly and applies whatever it says.
//
// The escape hatch for a webhook that never arrived, and the only recovery when
// the initiation response was lost so no provider id was ever stored — which is
// why it keys on our merchant refund id.
func (s *RefundService) RecheckStatus(ctx context.Context, refundID string) (*domain.Refund, error) {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if refund.IsTerminal() {
		return refund, nil
	}

	status, err := s.gateway.CheckRefundStatus(ctx, refund.MerchantRefundID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to check refund status")
	}

	// A first sighting of the provider id: the initiation response was lost, and
	// without this the webhook could never find this refund either.
	if refund.ProviderRefundID == "" && status.RefundID != "" {
		if setErr := s.refundRepo.SetProviderRefundID(ctx, refund.ID, status.RefundID); setErr != nil {
			slog.ErrorContext(ctx, "Failed to record provider refund id", "refund_id", refund.ID, "error", setErr)
		}
		refund.ProviderRefundID = status.RefundID
	}

	switch status.State {
	case phonepe.RefundStateCompleted, phonepe.RefundStateConfirmed:
		if err := s.settle(ctx, refund, domain.RefundStatusCompleted, "", ""); err != nil {
			return nil, err
		}
	case phonepe.RefundStateFailed:
		if err := s.settle(ctx, refund, domain.RefundStatusFailed, status.ErrorCode, status.DetailedErrorCode); err != nil {
			return nil, err
		}
	default:
		// Still pending at the provider; nothing to apply.
		return refund, nil
	}

	return s.refundRepo.GetByID(ctx, refund.ID)
}

// settle applies a terminal outcome exactly once.
//
// The repository's conditional update is the gate: PhonePe retries webhooks and
// Lambda can process two deliveries at once, so of several attempts exactly one
// gets past Settle. Everything after it — the payment total, the item
// quantities, the order status — runs only for that one.
func (s *RefundService) settle(ctx context.Context, refund *domain.Refund, status domain.RefundStatus, errorCode, detailedErrorCode string) error {
	// A cheap early out. Not the guard: the guard is Settle itself.
	if refund.IsTerminal() {
		return nil
	}

	completedAt := time.Now()
	if err := s.refundRepo.Settle(ctx, refund.ID, status, completedAt, errorCode, detailedErrorCode); err != nil {
		if appErr, ok := errors.AsAppError(err); ok && appErr.Code == errors.ErrCodeConflict {
			// Another delivery won. Not an error — the refund is settled.
			slog.InfoContext(ctx, "Refund already settled", "refund_id", refund.ID)
			return nil
		}
		return err
	}

	if status == domain.RefundStatusFailed {
		// Order and payment are untouched: nothing went back. The inventory
		// effect from creation is deliberately not reversed — see the design's
		// known gaps.
		metrics.Record(ctx, "refund_failed", metrics.L{metrics.LabelReason: string(refund.Reason)})
		slog.WarnContext(ctx, "Refund failed at provider",
			"refund_id", refund.ID, "error_code", errorCode, "detailed_error_code", detailedErrorCode)
		return nil
	}

	s.applyCompletion(ctx, refund, completedAt)

	metrics.Record(ctx, "refund_completed", metrics.L{metrics.LabelReason: string(refund.Reason)})
	slog.InfoContext(ctx, "Refund completed",
		"refund_id", refund.ID, "order_id", refund.OrderID, "amount_paise", refund.Amount)
	return nil
}

// applyCompletion records a completed refund against the payment and the order.
// Failures here are logged rather than returned: the money has already gone
// back, so refusing the webhook would only have PhonePe redeliver it into a
// refund that is now terminal and would be skipped.
func (s *RefundService) applyCompletion(ctx context.Context, refund *domain.Refund, completedAt time.Time) {
	refundedTotal, err := s.paymentRepo.AddRefundAmount(ctx, refund.PaymentID, refund.Amount)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to record refunded total", "refund_id", refund.ID, "error", err)
		return
	}

	order, err := s.orderRepo.GetByID(ctx, refund.OrderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load order for refund", "refund_id", refund.ID, "error", err)
		return
	}

	// Derived from every completed refund rather than incremented by this one.
	// Two refunds settling at once both read-modify-write this order, and an
	// increment loses one of them; recomputing the total makes the write
	// idempotent and self-healing instead.
	settled, err := s.refundRepo.ListByOrder(ctx, refund.OrderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to total refunded quantities", "refund_id", refund.ID, "error", err)
		return
	}
	refundedByItem := make(map[string]int)
	for _, r := range settled {
		if r.Status != domain.RefundStatusCompleted && r.ID != refund.ID {
			continue
		}
		for _, item := range r.Items {
			refundedByItem[item.OrderItemID] += item.Quantity
		}
	}
	for i := range order.Items {
		order.Items[i].RefundedQuantity = refundedByItem[order.Items[i].ID]
	}

	// Fulfillment status is left alone: whatever was not refunded still ships.
	if refundedTotal >= order.TotalAmount {
		order.PaymentStatus = domain.PaymentStatusRefunded
	} else {
		order.PaymentStatus = domain.PaymentStatusPartiallyRefunded
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		slog.ErrorContext(ctx, "Failed to update order after refund", "refund_id", refund.ID, "error", err)
		return
	}

	paymentStatus := domain.PaymentStatusPartiallyRefunded
	updates := map[string]interface{}{}
	if refundedTotal >= order.TotalAmount {
		paymentStatus = domain.PaymentStatusRefunded
		updates["refunded_at"] = completedAt
	}
	if err := s.paymentRepo.UpdateStatus(ctx, refund.PaymentID, paymentStatus, updates); err != nil {
		slog.ErrorContext(ctx, "Failed to update payment after refund", "refund_id", refund.ID, "error", err)
	}

	s.notifyCustomer(ctx, order, refund)
}

var _ domain.RefundService = (*RefundService)(nil)
