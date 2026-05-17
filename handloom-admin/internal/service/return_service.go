package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// ReturnService manages admin-initiated customer returns. It enforces a
// delivery-based return window, creates a reverse pickup with the courier,
// and persists the return record. Cancel currently performs a DB-only state
// transition; the carrier-side reverse pickup must be cancelled manually until
// the gateway exposes a cancel API.
type ReturnService struct {
	orderRepo        domain.OrderRepository
	shipmentRepo     domain.ShipmentRepository
	returnRepo       domain.ReturnRepository
	paymentService   domain.PaymentService
	courier          courier.Gateway
	publisher        event.EventPublisher
	pickupLocation   string
	returnWindowDays int
}

// NewReturnService creates a ReturnService. A non-positive returnWindowDays
// defaults to 7.
func NewReturnService(
	orderRepo domain.OrderRepository,
	shipmentRepo domain.ShipmentRepository,
	returnRepo domain.ReturnRepository,
	paymentService domain.PaymentService,
	gw courier.Gateway,
	pub event.EventPublisher,
	pickupLocation string,
	returnWindowDays int,
) *ReturnService {
	if returnWindowDays <= 0 {
		returnWindowDays = 7
	}
	return &ReturnService{
		orderRepo:        orderRepo,
		shipmentRepo:     shipmentRepo,
		returnRepo:       returnRepo,
		paymentService:   paymentService,
		courier:          gw,
		publisher:        pub,
		pickupLocation:   pickupLocation,
		returnWindowDays: returnWindowDays,
	}
}

// Create initiates a return for a delivered order: validates the return window,
// creates a reverse pickup with the courier, and persists the return record.
func (s *ReturnService) Create(ctx context.Context, orderID string, req domain.CreateReturnRequest, adminID string) (*domain.ReturnRequest, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.OrderStatusDelivered {
		return nil, errors.Validation("Order is not delivered; cannot return")
	}
	if order.DeliveredAt == nil {
		return nil, errors.Validation("Order has no delivery timestamp")
	}
	deadline := order.DeliveredAt.Add(time.Duration(s.returnWindowDays) * 24 * time.Hour)
	if time.Now().After(deadline) {
		return nil, errors.Validation("Return window expired")
	}
	sh, err := s.shipmentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot find forward shipment for return")
	}
	if order.ShippingAddress == nil {
		return nil, errors.Validation("Order has no shipping address; cannot create reverse pickup")
	}
	addr := order.ShippingAddress
	items := make([]courier.ShipmentItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, courier.ShipmentItem{
			Name:      it.SKU,
			SKU:       it.SKU,
			Quantity:  it.Quantity,
			UnitPaise: it.UnitPaise,
		})
	}
	revRes, err := s.courier.CreateReversePickup(ctx, &courier.ReversePickupRequest{
		OriginalOrderID: orderID,
		OriginalAWB:     sh.AWBNumber,
		Customer: courier.Address{
			FirstName:    addr.FirstName,
			LastName:     addr.LastName,
			Phone:        addr.Phone,
			AddressLine1: addr.AddressLine1,
			AddressLine2: addr.AddressLine2,
			City:         addr.City,
			State:        addr.State,
			Pincode:      addr.PostalCode,
			Country:      addr.Country,
		},
		PickupLocation: s.pickupLocation,
		Items:          items,
		Reason:         req.Reason,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create reverse pickup")
	}
	rr := &domain.ReturnRequest{
		ID:                uuid.New().String(),
		OrderID:           orderID,
		ShipmentID:        sh.ID,
		ReverseAWB:        revRes.ReverseAWB,
		ReverseShipmentID: revRes.CarrierShipmentID,
		Reason:            req.Reason,
		Items:             req.Items,
		Status:            domain.ReturnStatusRequested,
		CreatedBy:         adminID,
	}
	if err := s.returnRepo.Create(ctx, rr); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, event.New(event.ReturnRequested, map[string]any{
		"return_id":   rr.ID,
		"order_id":    orderID,
		"reverse_awb": rr.ReverseAWB,
	}))
	return rr, nil
}

// ListByOrder returns every return colocated under the given order.
func (s *ReturnService) ListByOrder(ctx context.Context, orderID string) ([]*domain.ReturnRequest, error) {
	return s.returnRepo.ListByOrder(ctx, orderID)
}

// Cancel cancels a return request. Only REQUESTED returns can be cancelled.
// DB-only transition: the carrier-side reverse pickup (if already scheduled)
// must be cancelled manually with Delhivery until courier.Gateway exposes a
// CancelReversePickup method. A WARN log is emitted so operators notice.
func (s *ReturnService) Cancel(ctx context.Context, returnID string, adminID string) error {
	rr, err := s.returnRepo.GetByReturnID(ctx, returnID)
	if err != nil {
		return err
	}
	if rr.Status != domain.ReturnStatusRequested {
		return errors.Validation("Only REQUESTED returns can be cancelled")
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"cancelled_at": now.Format(time.RFC3339),
		"cancelled_by": adminID,
	}
	if err := s.returnRepo.UpdateStatus(ctx, rr.OrderID, rr.ID, domain.ReturnStatusCancelled, updates); err != nil {
		return errors.Wrap(err, "Failed to cancel return")
	}
	if rr.ReverseAWB != "" {
		slog.WarnContext(ctx, "Return cancelled in DB; reverse pickup must be cancelled manually with carrier",
			"return_id", rr.ID, "reverse_awb", rr.ReverseAWB, "admin_id", adminID)
	}
	_ = s.publisher.Publish(ctx, event.New(event.ReturnCancelled, map[string]any{
		"return_id":   rr.ID,
		"order_id":    rr.OrderID,
		"reverse_awb": rr.ReverseAWB,
		"admin_id":    adminID,
	}))
	return nil
}

// ProcessRefund triggers a PSP refund for a return that has been received,
// then persists the REFUNDED status with amount + timestamp.
//
// Idempotency: the return ID is passed to PaymentService.RefundPayment as the
// idempotency key, so retries after a PSP-success / local-write-failure path
// re-sync state without double-charging the customer. ProcessRefund itself is
// safe to retry: REFUNDED returns short-circuit, and RECEIVED returns with a
// prior PSP attempt fall through to a state re-sync.
func (s *ReturnService) ProcessRefund(ctx context.Context, returnID string, amountPaise int64, adminID string) error {
	if amountPaise <= 0 {
		return errors.BadRequest("Refund amount must be positive")
	}
	rr, err := s.returnRepo.GetByReturnID(ctx, returnID)
	if err != nil {
		return err
	}
	if rr.Status == domain.ReturnStatusRefunded {
		return nil // Idempotent — already settled.
	}
	if rr.Status != domain.ReturnStatusReceived {
		return errors.Validation("Return must be RECEIVED before refund")
	}
	payment, err := s.paymentService.GetByOrderID(ctx, rr.OrderID)
	if err != nil {
		return errors.Wrap(err, "Failed to load payment for refund")
	}
	if payment == nil {
		return errors.Validation("No payment found for order")
	}
	if amountPaise > payment.Amount {
		return errors.Validation("Refund amount exceeds original payment")
	}
	if err := s.paymentService.RefundPayment(ctx, payment.ID, amountPaise, "Return refund: "+rr.ID, rr.ID); err != nil {
		return errors.Wrap(err, "Failed to refund payment")
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"refund_amount_paise": amountPaise,
		"refunded_at":         now.Format(time.RFC3339),
		"refunded_by":         adminID,
	}
	if err := s.returnRepo.UpdateStatus(ctx, rr.OrderID, rr.ID, domain.ReturnStatusRefunded, updates); err != nil {
		slog.ErrorContext(ctx, "Refund succeeded but return status update failed — retry will re-sync",
			"return_id", rr.ID, "payment_id", payment.ID, "error", err)
		return errors.Wrap(err, "Refund processed but status update failed")
	}
	_ = s.publisher.Publish(ctx, event.New(event.ReturnRefunded, map[string]any{
		"return_id":    rr.ID,
		"order_id":     rr.OrderID,
		"payment_id":   payment.ID,
		"amount_paise": amountPaise,
		"admin_id":     adminID,
	}))
	return nil
}

// HandleReverseWebhook is the entry point for courier reverse-pickup status
// callbacks. It receives events forwarded from ShippingService.HandleWebhook when
// the parsed event is flagged as a reverse shipment.
//
// TODO(phase-3): look up the return by reverse AWB and persist the new status.
// Requires a ReturnRepository.GetByReverseAWB method (not yet exposed). For now
// we log and return nil so the carrier acks the webhook.
func (s *ReturnService) HandleReverseWebhook(ctx context.Context, awb string, status domain.ReturnStatus) error {
	slog.InfoContext(ctx, "Reverse webhook received", "reverse_awb", awb, "status", status)
	return nil
}

// Ensure interface compliance.
var _ domain.ReturnService = (*ReturnService)(nil)
