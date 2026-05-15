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

// Ensure interface compliance.
var _ domain.NDRService = (*NDRService)(nil)
