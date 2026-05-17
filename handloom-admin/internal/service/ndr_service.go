package service

import (
	"context"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// NDRService handles Non-Delivery Report events from the courier. It auto
// re-attempts delivery until the NDR count reaches maxAttempts, then escalates
// the shipment to the admin queue.
type NDRService struct {
	shipmentRepo domain.ShipmentRepository
	courier      courier.Gateway
	publisher    event.EventPublisher
	maxAttempts  int
}

// NewNDRService creates an NDRService. A non-positive maxAttempts defaults to 3.
func NewNDRService(
	repo domain.ShipmentRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
	maxAttempts int,
) *NDRService {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &NDRService{
		shipmentRepo: repo,
		courier:      gw,
		publisher:    pub,
		maxAttempts:  maxAttempts,
	}
}

// HandleNDREvent records an NDR (failed delivery) and either requests a
// re-attempt from the carrier or escalates to the admin queue.
//
// Semantics with maxAttempts=3 (default): on the 1st NDR (count→1) and 2nd
// NDR (count→2) the shipment is re-attempted; on the 3rd NDR (count→3) it
// escalates. Total maximum delivery attempts = maxAttempts (1 original
// + maxAttempts-1 re-attempts).
func (n *NDRService) HandleNDREvent(ctx context.Context, awb, reason string) error {
	sh, err := n.shipmentRepo.GetByAWB(ctx, awb)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch shipment for NDR")
	}
	nextCount := sh.NDRCount + 1
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"ndr_count":       nextCount,
		"last_ndr_reason": reason,
		"last_ndr_at":     now.Format(time.RFC3339),
	}
	if nextCount >= n.maxAttempts {
		updates["ndr_escalated"] = true
		if err := n.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, domain.ShipmentStatusNDREscalated, updates); err != nil {
			return errors.Wrap(err, "Failed to update escalated shipment")
		}
		_ = n.publisher.Publish(ctx, event.New(event.ShipmentNDREscalated, map[string]any{
			"awb":      awb,
			"order_id": sh.OrderID,
			"reason":   reason,
			"attempts": nextCount,
		}))
		return nil
	}
	if err := n.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, domain.ShipmentStatusNDR, updates); err != nil {
		return errors.Wrap(err, "Failed to update NDR shipment")
	}
	if err := n.courier.ReAttemptDelivery(ctx, awb, courier.NDRActionReAttempt); err != nil {
		return errors.Wrap(err, "Failed to request re-attempt")
	}
	_ = n.publisher.Publish(ctx, event.New(event.ShipmentNDRReattempted, map[string]any{
		"awb":      awb,
		"order_id": sh.OrderID,
		"reason":   reason,
		"attempts": nextCount,
	}))
	return nil
}

// HandleAdminAction dispatches an operator-triggered NDR action against an
// escalated shipment, identified by AWB. REATTEMPT/RTO call the carrier;
// MARK_CONTACTED is a DB-only annotation.
func (n *NDRService) HandleAdminAction(ctx context.Context, awb string, action domain.NDRAdminAction, note, adminID string) error {
	sh, err := n.shipmentRepo.GetByAWB(ctx, awb)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch shipment for NDR action")
	}
	if !sh.NDREscalated && sh.Status != domain.ShipmentStatusNDREscalated && sh.Status != domain.ShipmentStatusNDR {
		return errors.Validation("Shipment is not in an NDR state")
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"last_ndr_action":    string(action),
		"last_ndr_action_at": now.Format(time.RFC3339),
		"last_ndr_action_by": adminID,
	}
	if note != "" {
		updates["last_ndr_action_note"] = note
	}

	switch action {
	case domain.NDRAdminActionReattempt:
		if err := n.courier.ReAttemptDelivery(ctx, awb, courier.NDRActionReAttempt); err != nil {
			return errors.Wrap(err, "Carrier re-attempt failed")
		}
		if err := n.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, domain.ShipmentStatusNDR, updates); err != nil {
			return errors.Wrap(err, "Failed to update shipment after re-attempt")
		}
		_ = n.publisher.Publish(ctx, event.New(event.ShipmentNDRReattempted, map[string]any{
			"awb": awb, "order_id": sh.OrderID, "admin_id": adminID, "source": "admin",
		}))
	case domain.NDRAdminActionRTO:
		if err := n.courier.ReAttemptDelivery(ctx, awb, courier.NDRActionRTO); err != nil {
			return errors.Wrap(err, "Carrier RTO failed")
		}
		if err := n.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, domain.ShipmentStatusRTO, updates); err != nil {
			return errors.Wrap(err, "Failed to update shipment after RTO")
		}
		_ = n.publisher.Publish(ctx, event.New(event.ShipmentRTO, map[string]any{
			"awb": awb, "order_id": sh.OrderID, "admin_id": adminID,
		}))
	case domain.NDRAdminActionMarkContacted:
		updates["ndr_customer_contacted"] = true
		if err := n.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, sh.Status, updates); err != nil {
			return errors.Wrap(err, "Failed to mark customer contacted")
		}
	default:
		return errors.BadRequest("Unsupported NDR action")
	}
	return nil
}

// Ensure interface compliance.
var _ domain.NDRService = (*NDRService)(nil)
