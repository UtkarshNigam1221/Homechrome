package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// ManifestService orchestrates manifest creation and pickup scheduling for
// shipments via the carrier-agnostic courier.Gateway.
type ManifestService struct {
	shipmentRepo   domain.ShipmentRepository
	courier        courier.Gateway
	publisher      event.EventPublisher
	pickupLocation string
}

// NewManifestService creates a ManifestService.
func NewManifestService(
	repo domain.ShipmentRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
	pickupLocation string,
) *ManifestService {
	return &ManifestService{
		shipmentRepo:   repo,
		courier:        gw,
		publisher:      pub,
		pickupLocation: pickupLocation,
	}
}

// CreatePerOrderManifest manifests a single shipment immediately (used for
// PRIORITY orders). Creates a one-AWB manifest, schedules the pickup, marks the
// shipment as MANIFESTED, and publishes a shipment.manifested event.
func (m *ManifestService) CreatePerOrderManifest(ctx context.Context, sh *domain.Shipment) error {
	if sh.AWBNumber == "" {
		return errors.Validation("Shipment has no AWB; cannot manifest")
	}
	pickupDate := time.Now().UTC()
	res, err := m.courier.CreateManifest(ctx, []string{sh.AWBNumber}, pickupDate)
	if err != nil {
		return errors.Wrap(err, "Failed to create per-order manifest")
	}
	if err := m.courier.SchedulePickup(ctx, res.ManifestID, m.pickupLocation, pickupDate); err != nil {
		return errors.Wrap(err, "Failed to schedule pickup")
	}
	if err := m.shipmentRepo.UpdateStatus(ctx, sh.OrderID, sh.ID, sh.Priority, domain.ShipmentStatusManifested,
		map[string]interface{}{"manifest_id": res.ManifestID}); err != nil {
		return errors.Wrap(err, "Failed to update shipment status")
	}
	_ = m.publisher.Publish(ctx, event.New(event.ShipmentManifested, map[string]any{
		"shipment_id": sh.ID,
		"order_id":    sh.OrderID,
		"awb":         sh.AWBNumber,
		"manifest_id": res.ManifestID,
	}))
	return nil
}

// RunDailyBatch manifests every NORMAL-priority shipment in CREATED state in a
// single batch, schedules one pickup for the manifest, marks each shipment as
// MANIFESTED, and publishes a shipment.pickup_scheduled event.
func (m *ManifestService) RunDailyBatch(ctx context.Context, pickupDate time.Time) (*domain.BatchResult, error) {
	shipments, err := m.shipmentRepo.QueryByPriorityStatus(ctx, domain.PriorityNormal, domain.ShipmentStatusCreated, 500)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query normal-priority shipments")
	}
	if len(shipments) == 0 {
		return &domain.BatchResult{ShipmentCount: 0}, nil
	}
	awbs := make([]string, 0, len(shipments))
	for _, s := range shipments {
		if s.AWBNumber == "" {
			slog.WarnContext(ctx, "Skipping shipment without AWB", "shipment_id", s.ID)
			continue
		}
		awbs = append(awbs, s.AWBNumber)
	}
	if len(awbs) == 0 {
		return &domain.BatchResult{ShipmentCount: 0}, nil
	}
	res, err := m.courier.CreateManifest(ctx, awbs, pickupDate)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create batch manifest")
	}
	if err := m.courier.SchedulePickup(ctx, res.ManifestID, m.pickupLocation, pickupDate); err != nil {
		return nil, errors.Wrap(err, "Failed to schedule batch pickup")
	}
	// Track per-shipment status update outcomes. Pickup is already scheduled
	// with the carrier, so any DynamoDB failure here is a partial inconsistency
	// that ops must reconcile — surface the failing IDs in the result and log
	// every failure at ERROR.
	marked := make([]string, 0, len(awbs))
	failed := make([]string, 0)
	for _, s := range shipments {
		if s.AWBNumber == "" {
			continue
		}
		if uerr := m.shipmentRepo.UpdateStatus(ctx, s.OrderID, s.ID, s.Priority, domain.ShipmentStatusManifested,
			map[string]interface{}{"manifest_id": res.ManifestID}); uerr != nil {
			slog.ErrorContext(ctx, "Failed to mark shipment manifested",
				"shipment_id", s.ID, "awb", s.AWBNumber, "manifest_id", res.ManifestID, "error", uerr)
			failed = append(failed, s.ID)
			continue
		}
		marked = append(marked, s.ID)
	}
	_ = m.publisher.Publish(ctx, event.New(event.ShipmentPickupScheduled, map[string]any{
		"manifest_id":    res.ManifestID,
		"shipment_count": len(awbs),
		"pickup_date":    pickupDate.Format(time.RFC3339),
	}))
	return &domain.BatchResult{
		ManifestID:        res.ManifestID,
		ShipmentCount:     len(awbs),
		ShipmentMarkedIDs: marked,
		FailedShipmentIDs: failed,
	}, nil
}

// Ensure interface compliance.
var _ domain.ManifestService = (*ManifestService)(nil)
