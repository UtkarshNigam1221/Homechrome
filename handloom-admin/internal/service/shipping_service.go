// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// ShippingService implements domain.ShippingService backed by a courier.Gateway.
type ShippingService struct {
	shipmentRepo   domain.ShipmentRepository
	orderRepo      domain.OrderRepository
	pincodeRepo    domain.PincodeRepository
	courier        courier.Gateway
	publisher      event.EventPublisher
	returnService  domain.ReturnService
	pickupLocation string
}

// NewShippingService creates a new ShippingService.
func NewShippingService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	pincodeRepo domain.PincodeRepository,
	gw courier.Gateway,
	publisher event.EventPublisher,
	returnService domain.ReturnService,
	pickupLocation string,
) *ShippingService {
	return &ShippingService{
		shipmentRepo:   shipmentRepo,
		orderRepo:      orderRepo,
		pincodeRepo:    pincodeRepo,
		courier:        gw,
		publisher:      publisher,
		returnService:  returnService,
		pickupLocation: pickupLocation,
	}
}

// CheckServiceability checks delivery serviceability for a pincode, caching
// results in the pincode repository for 7 days.
func (s *ShippingService) CheckServiceability(ctx context.Context, pincode string, _ int) (*domain.ServiceabilityResult, error) {
	pz, err := s.pincodeRepo.Get(ctx, pincode)
	needFetch := err != nil || pz == nil || time.Since(pz.RefreshedAt) > 7*24*time.Hour
	if needFetch {
		info, fetchErr := s.courier.CheckPincode(ctx, pincode)
		if fetchErr != nil {
			return nil, errors.Wrap(fetchErr, "Failed to check pincode")
		}
		pz = &domain.PincodeZone{
			Pincode:          info.Pincode,
			Zone:             info.Zone,
			City:             info.City,
			State:            info.State,
			Serviceable:      info.Serviceable,
			CODAvailable:     info.CODAvailable,
			PrepaidAvailable: info.PrepaidAvailable,
			RefreshedAt:      time.Now().UTC(),
			TTL:              time.Now().Add(7 * 24 * time.Hour).Unix(),
		}
		if upErr := s.pincodeRepo.Upsert(ctx, pz); upErr != nil {
			slog.WarnContext(ctx, "Failed to cache pincode", "error", upErr)
		}
	}
	return &domain.ServiceabilityResult{
		Serviceable: pz.Serviceable,
		Couriers: []domain.CourierOption{{
			ID:            0,
			Name:          "Delhivery",
			Rate:          0,
			EstimatedDays: 4,
		}},
	}, nil
}

// isTerminalStatus reports whether a shipment status will not change again.
func isTerminalStatus(s domain.ShipmentStatus) bool {
	switch s {
	case domain.ShipmentStatusDelivered, domain.ShipmentStatusRTO, domain.ShipmentStatusReturned:
		return true
	}
	return false
}

// CreateShipment creates a forward shipment via the courier gateway and
// persists the resulting shipment record.
func (s *ShippingService) CreateShipment(ctx context.Context, order *domain.Order, priority domain.ShipmentPriority) (*domain.Shipment, error) {
	if order.ShippingAddress == nil {
		return nil, errors.Validation("Order has no shipping address")
	}
	items := make([]courier.ShipmentItem, 0, len(order.Items))
	for _, it := range order.Items {
		items = append(items, courier.ShipmentItem{
			Name:      it.ProductName,
			SKU:       it.ProductSKU,
			Quantity:  it.Quantity,
			UnitPaise: it.UnitPrice,
		})
	}
	addr := order.ShippingAddress
	mode := courier.PaymentPrepaid
	codAmt := int64(0)
	if order.PaymentMethod == "COD" {
		mode = courier.PaymentCOD
		codAmt = order.TotalAmount
	}
	req := &courier.CreateShipmentRequest{
		OrderID:        order.ID,
		PickupLocation: s.pickupLocation,
		Customer: courier.Address{
			FirstName:    addr.FirstName,
			LastName:     addr.LastName,
			Phone:        addr.Phone,
			Email:        order.CustomerEmail,
			AddressLine1: addr.AddressLine1,
			AddressLine2: addr.AddressLine2,
			City:         addr.City,
			State:        addr.State,
			Pincode:      addr.PostalCode,
			Country:      addr.Country,
		},
		Items:              items,
		PaymentMode:        mode,
		CODAmountPaise:     codAmt,
		WeightGrams:        500,
		LengthCm:           30,
		BreadthCm:          25,
		HeightCm:           5,
		DeclaredValuePaise: order.Subtotal,
	}
	res, err := s.courier.CreateShipment(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create shipment")
	}
	label, lerr := s.courier.GenerateLabel(ctx, res.AWB)
	if lerr != nil {
		slog.WarnContext(ctx, "Failed to fetch label", "error", lerr)
	}
	now := time.Now().UTC()
	sh := &domain.Shipment{
		ID:                 uuid.New().String(),
		OrderID:            order.ID,
		Provider:           "delhivery",
		ProviderShipmentID: res.CarrierShipmentID,
		AWBNumber:          res.AWB,
		CourierName:        "Delhivery",
		Status:             domain.ShipmentStatusCreated,
		Priority:           priority,
		PickupLocation:     s.pickupLocation,
		LabelURL:           label,
		WeightGrams:        req.WeightGrams,
		IsCOD:              mode == courier.PaymentCOD,
		CODAmountPaise:     codAmt,
		ShippedAt:          &now,
	}
	if err := s.shipmentRepo.Create(ctx, sh); err != nil {
		return nil, errors.Wrap(err, "Failed to save shipment")
	}
	_ = s.publisher.Publish(ctx, event.New(event.ShipmentCreated, sh))
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusShipped, "system"); err != nil {
		slog.ErrorContext(ctx, "Failed to update order status to SHIPPED", "order_id", order.ID, "error", err)
	}
	if err := s.orderRepo.UpdateTracking(ctx, order.ID, sh.AWBNumber, sh.CourierName); err != nil {
		slog.ErrorContext(ctx, "Failed to update tracking", "order_id", order.ID, "error", err)
	}
	return sh, nil
}

// TrackShipment returns the latest shipment status for an order, refreshing
// from the courier gateway when the cached record is stale or non-terminal.
func (s *ShippingService) TrackShipment(ctx context.Context, orderID string) (*domain.Shipment, error) {
	sh, err := s.shipmentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if sh.AWBNumber == "" {
		return sh, nil
	}
	if time.Since(sh.UpdatedAt) < 30*time.Minute && isTerminalStatus(sh.Status) {
		return sh, nil
	}
	info, err := s.courier.TrackByAWB(ctx, sh.AWBNumber)
	if err != nil {
		slog.WarnContext(ctx, "Live track failed; returning cached", "awb", sh.AWBNumber, "error", err)
		return sh, nil
	}
	newStatus := courier.ToShipmentStatus(info.Status)
	if newStatus != sh.Status {
		updates := map[string]interface{}{}
		if newStatus == domain.ShipmentStatusDelivered {
			updates["delivered_at"] = time.Now().UTC().Format(time.RFC3339)
		}
		if upErr := s.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, newStatus, updates); upErr != nil {
			slog.ErrorContext(ctx, "Failed to update shipment", "error", upErr)
		}
		sh.Status = newStatus
		if newStatus == domain.ShipmentStatusDelivered {
			if upErr := s.orderRepo.UpdateStatus(ctx, sh.OrderID, domain.OrderStatusDelivered, "system"); upErr != nil {
				slog.ErrorContext(ctx, "Failed to update order to DELIVERED", "order_id", sh.OrderID, "error", upErr)
			}
		}
	}
	return sh, nil
}

// HandleWebhook processes an incoming courier webhook: verifies the signature,
// parses the event, looks up the shipment by AWB, and updates its status.
func (s *ShippingService) HandleWebhook(ctx context.Context, body []byte, headers http.Header) error {
	if err := s.courier.VerifyWebhookSignature(headers, body); err != nil {
		// VerifyWebhookSignature returns a typed AppError (ErrCodeUnauthorized).
		// Propagate as-is so the handler responds with HTTP 401 instead of 500.
		return err
	}
	ev, err := s.courier.ParseWebhook(body)
	if err != nil {
		return errors.Wrap(err, "Failed to parse webhook")
	}
	// Reverse pickups have their own lifecycle (return request → reverse shipment).
	// Hand them off to ReturnService so the forward shipment lookup below does not
	// try to map a reverse AWB onto a forward shipment.
	if ev.IsReverse && s.returnService != nil {
		return s.returnService.HandleReverseWebhook(ctx, ev.AWB, courier.ToReturnStatus(ev.Status))
	}
	sh, err := s.shipmentRepo.GetByAWB(ctx, ev.AWB)
	if err != nil {
		// Acknowledge unknown AWBs so the courier does not retry indefinitely.
		slog.WarnContext(ctx, "Webhook for unknown AWB", "awb", ev.AWB, "error", err)
		return nil //nolint:nilerr // intentional: swallow lookup error to ACK webhook
	}
	newStatus := courier.ToShipmentStatus(ev.Status)
	if newStatus == sh.Status {
		return nil
	}
	if ev.Status == courier.EventNDR {
		_ = s.publisher.Publish(ctx, event.New(event.ShipmentUpdated, map[string]any{
			"awb":    ev.AWB,
			"status": "NDR",
			"reason": ev.NDRReason,
		}))
	}
	updates := map[string]interface{}{}
	if newStatus == domain.ShipmentStatusDelivered {
		updates["delivered_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	if err := s.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, newStatus, updates); err != nil {
		return errors.Wrap(err, "Failed to update shipment from webhook")
	}
	switch newStatus {
	case domain.ShipmentStatusDelivered:
		if upErr := s.orderRepo.UpdateStatus(ctx, sh.OrderID, domain.OrderStatusDelivered, "system"); upErr != nil {
			slog.ErrorContext(ctx, "Failed to update order to DELIVERED", "order_id", sh.OrderID, "error", upErr)
		}
	case domain.ShipmentStatusRTO:
		if upErr := s.orderRepo.UpdateStatus(ctx, sh.OrderID, domain.OrderStatusReturned, "system"); upErr != nil {
			slog.ErrorContext(ctx, "Failed to update order to RETURNED", "order_id", sh.OrderID, "error", upErr)
		}
	}
	return nil
}

// Ensure interface compliance
var _ domain.ShippingService = (*ShippingService)(nil)
