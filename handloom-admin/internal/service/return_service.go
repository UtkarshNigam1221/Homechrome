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
// and persists the return record. Refunds and webhook-driven status updates
// are publish-only stubs in this slice (full implementations land later).
type ReturnService struct {
	orderRepo        domain.OrderRepository
	shipmentRepo     domain.ShipmentRepository
	returnRepo       domain.ReturnRepository
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

// Cancel cancels a return request. Not yet implemented in this slice.
func (s *ReturnService) Cancel(ctx context.Context, returnID string, adminID string) error {
	slog.WarnContext(ctx, "Return cancel not yet implemented", "return_id", returnID, "admin", adminID)
	return errors.NotImplemented("Return cancel not implemented")
}

// ProcessRefund is a Phase 3 stub. PaymentService refund integration, status
// mutation, and refunded_at persistence land in the next slice; until then
// callers receive NotImplemented rather than a silent fake success.
func (s *ReturnService) ProcessRefund(ctx context.Context, returnID string, amountPaise int64, adminID string) error {
	slog.WarnContext(ctx, "Refund processing not implemented",
		"return_id", returnID, "amount_paise", amountPaise, "admin_id", adminID)
	return errors.NotImplemented("Refund processing wired in Phase 3")
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
