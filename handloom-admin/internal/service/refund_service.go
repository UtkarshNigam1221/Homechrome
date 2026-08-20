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

// RefundService owns the refund lifecycle: deriving the amount, moving the money,
// settling asynchronously. Its own service — that lifecycle spans two Lambdas.
type RefundService struct {
	refundRepo    domain.RefundRepository
	orderRepo     domain.OrderRepository
	paymentRepo   domain.PaymentRepository
	inventoryRepo domain.InventoryRepository
	gateway       phonepe.Gateway
}

// NewRefundService creates a new RefundService.
func NewRefundService(
	refundRepo domain.RefundRepository,
	orderRepo domain.OrderRepository,
	paymentRepo domain.PaymentRepository,
	inventoryRepo domain.InventoryRepository,
	gateway phonepe.Gateway,
) *RefundService {
	return &RefundService{
		refundRepo:    refundRepo,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		gateway:       gateway,
	}
}

// Create derives the amount, records the refund, then asks the provider. Record
// first: a refund with no local row is unreconcilable, a stale PENDING one is not.
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

	breakdown, err := deriveRefundAmount(order, req.Items, payment.RefundAmount)
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

	metrics.Record(ctx, "refund_initiated", metrics.L{
		metrics.LabelReason:  string(refund.Reason),
		metrics.LabelGateway: gatewayPhonePe,
	})
	slog.InfoContext(ctx, "Refund initiated",
		"refund_id", refund.ID, "order_id", order.ID, "amount_paise", refund.Amount)

	return refund, nil
}

// applyInventoryEffect moves stock for a refund, only before dispatch: CommitStock
// consumes the reservation at SHIPPED, after which RETURNED owns restocking.
func (s *RefundService) applyInventoryEffect(ctx context.Context, order *domain.Order, refund *domain.Refund) {
	if order.Status == domain.OrderStatusShipped || order.Status == domain.OrderStatusDelivered ||
		order.Status == domain.OrderStatusReturned {
		return
	}

	for _, item := range refund.Items {
		var err error
		reason := reasonWriteOff
		if item.Restock {
			reason = reasonRelease
			// Back on sale: the reservation is released and nothing else moves.
			_, err = s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, order.ID)
		} else {
			// Written off: the goods are not there, so the reservation goes and
			// on-hand falls with it.
			_, err = s.inventoryRepo.WriteOffStock(ctx, item.ProductID, item.Quantity, order.ID)
		}
		if err != nil {
			slog.ErrorContext(ctx, "Failed to move stock for refund",
				keyProductID, item.ProductID, "refund_id", refund.ID, "error", err)
			metrics.Record(ctx, "inventory_mutation_failed",
				metrics.L{metrics.LabelReason: reason})
		}
	}
}

// ListByOrder returns an order's refunds.
func (s *RefundService) ListByOrder(ctx context.Context, orderID string) ([]*domain.Refund, error) {
	return s.refundRepo.ListByOrder(ctx, orderID)
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

	refundedByItem := make(map[string]int, len(refund.Items))
	for _, item := range refund.Items {
		refundedByItem[item.OrderItemID] += item.Quantity
	}
	for i := range order.Items {
		order.Items[i].RefundedQuantity += refundedByItem[order.Items[i].ID]
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
}

var _ domain.RefundService = (*RefundService)(nil)
