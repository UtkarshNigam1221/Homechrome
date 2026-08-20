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

// refundAuditor records who sent money back. One verb rather than the whole audit
// service. Nil in the store Lambda, which settles refunds but raises none.
type refundAuditor interface {
	Log(ctx context.Context, action, entityType, entityID, userID string,
		changes []domain.FieldChange, metadata map[string]interface{}) error
}

// refundNotifier tells the customer their money is on its way back. Nil where there
// is no notification service to reach.
type refundNotifier interface {
	SendOrderNotification(ctx context.Context, order *domain.Order,
		trigger domain.NotificationTrigger, createdBy string) error
}

// RefundService owns the refund lifecycle: deriving the amount, moving the money,
// settling asynchronously. Its own service — that lifecycle spans two Lambdas.
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

// Create derives the amount, records the refund, then asks the provider. Record
// first: a refund with no local row is unreconcilable, a stale PENDING one is not.
func (s *RefundService) Create(ctx context.Context, orderID string, req domain.CreateRefundRequest, createdBy string) (*domain.Refund, error) {
	if !req.Reason.IsValid() {
		return nil, errors.BadRequest("Unknown refund reason")
	}

	order, payment, breakdown, err := s.price(ctx, orderID, req.Items)
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
		// Provider never accepted it, so no money and no stock moved. Recording the
		// failure beats leaving the row PENDING forever.
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

// audit records who sent the money back, with the lines it covered. Swallowed on
// failure: the refund is already live, and losing the trail does not make it failed.
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

// claimedByLive totals what refunds that have not failed already account for, per line
// and in money. A failed refund returns nothing, so its units are free again.
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

// price is everything Create and Preview agree on: the order, the payment that can
// be refunded, and what the requested lines are worth against what is already claimed.
func (s *RefundService) price(ctx context.Context, orderID string, items []domain.CreateRefundItemRequest) (*domain.Order, *domain.Payment, *refundBreakdown, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, nil, nil, err
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, nil, errors.BadRequest("Order has no payment to refund")
	}
	if payment.Status != domain.PaymentStatusPaid && payment.Status != domain.PaymentStatusSuccess &&
		payment.Status != domain.PaymentStatusPartiallyRefunded {
		return nil, nil, nil, errors.BadRequest("Order has not been paid")
	}

	// What every non-failed refund claims, settled or in flight. Best-effort, not a
	// proof: this reads a GSI, so two creates inside the lag can miss each other.
	existing, err := s.refundRepo.ListByOrder(ctx, order.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	claimed, claimedAmount := claimedByLive(existing)
	if claimedAmount < payment.RefundAmount {
		// The payment total is authoritative for money; a refund whose settlement
		// half-completed can leave it ahead of the rows.
		claimedAmount = payment.RefundAmount
	}

	breakdown, err := deriveRefundAmount(order, items, claimed, claimedAmount)
	if err != nil {
		return nil, nil, nil, err
	}
	return order, payment, breakdown, nil
}

// Preview prices a refund without raising one. The screen an admin authorizes from
// shows this, so it must be the same derivation Create uses — hence the shared price.
func (s *RefundService) Preview(ctx context.Context, orderID string, req domain.PreviewRefundRequest) (*domain.RefundPreview, error) {
	_, _, breakdown, err := s.price(ctx, orderID, req.Items)
	if err != nil {
		return nil, err
	}

	return &domain.RefundPreview{
		Total:   breakdown.Total,
		IsFinal: breakdown.IsFinal,
		Lines:   breakdown.Items,
		Breakdown: domain.RefundPreviewBreakdown{
			LineValue: breakdown.LineValue,
			Discount:  breakdown.Discount,
			Tax:       breakdown.Tax,
			Shipping:  breakdown.Shipping,
		},
	}, nil
}

// applyInventoryEffect moves stock for a refund, and only while the order still holds a
// reservation to move: dispatch consumes it, and a cancellation already released it.
func (s *RefundService) applyInventoryEffect(ctx context.Context, order *domain.Order, refund *domain.Refund) {
	switch order.Status {
	case domain.OrderStatusShipped, domain.OrderStatusDelivered,
		domain.OrderStatusReturned, domain.OrderStatusCancelled:
		return
	}

	for group, quantity := range refundQuantities(refund.Items) {
		var err error
		reason := reasonWriteOff
		if group.restock {
			reason = reasonRelease
			// Back on sale: the reservation is released and nothing else moves.
			_, err = s.inventoryRepo.ReleaseStock(ctx, group.productID, quantity, order.ID)
		} else {
			// Written off: the goods are not there, so the reservation goes and
			// on-hand falls with it.
			_, err = s.inventoryRepo.WriteOffStock(ctx, group.productID, quantity, order.ID)
		}
		if err != nil {
			slog.ErrorContext(ctx, "Failed to move stock for refund",
				keyProductID, group.productID, "refund_id", refund.ID, "error", err)
			metrics.Record(ctx, "inventory_mutation_failed",
				metrics.L{metrics.LabelReason: reason})
		}
	}
}

// movementGroup is one stock movement's worth of a refund: restock picks the type,
// so two lines of one product only merge when they are going the same way.
type movementGroup struct {
	productID string
	restock   bool
}

// refundQuantities aggregates a refund's lines by product, like orderQuantities does
// for an order: movements are idempotent per product, so two lines would dedup to one.
func refundQuantities(items []domain.RefundItem) map[movementGroup]int {
	quantities := make(map[movementGroup]int, len(items))
	for _, item := range items {
		quantities[movementGroup{item.ProductID, item.Restock}] += item.Quantity
	}
	return quantities
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

// RecheckStatus asks the provider directly and applies what it says. Keys on our
// merchant refund id, so it works even when no provider id was ever stored.
func (s *RefundService) RecheckStatus(ctx context.Context, orderID, refundID string) (*domain.Refund, error) {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if refund.OrderID != orderID {
		// The route nests the refund under an order; honor that rather than
		// letting any refund be re-checked through any order's URL.
		return nil, errors.NotFound("Refund not found on this order")
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

// settle applies a terminal outcome exactly once. Settle's conditional update is
// the gate: of several concurrent deliveries, only one gets past it.
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
		// Nothing went back, so order and payment are untouched. The creation-time
		// inventory effect is deliberately not reversed — see the design's known gaps.
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

// applyCompletion records a completed refund against the payment and order. Failures
// are logged, not returned: the money is gone, and a redelivery would be skipped.
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

	// Derived from every completed refund, not incremented by this one: two settling at
	// once both read-modify-write the order, and an increment loses one of them.
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
	order.PaymentStatus = domain.PaymentStatusPartiallyRefunded
	updates := map[string]interface{}{}
	if refundedTotal >= order.TotalAmount {
		order.PaymentStatus = domain.PaymentStatusRefunded
		updates["refunded_at"] = completedAt
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		slog.ErrorContext(ctx, "Failed to update order after refund", "refund_id", refund.ID, "error", err)
		return
	}

	if err := s.paymentRepo.UpdateStatus(ctx, refund.PaymentID, order.PaymentStatus, updates); err != nil {
		slog.ErrorContext(ctx, "Failed to update payment after refund", "refund_id", refund.ID, "error", err)
	}

	s.notifyCustomer(ctx, order, refund)
}

var _ domain.RefundService = (*RefundService)(nil)
