// Package cron contains scheduled Lambda handlers.
package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/handloom/admin/internal/domain"
)

// PickupBatchHandler runs the daily NORMAL-priority manifest + pickup batch.
type PickupBatchHandler struct {
	manifest domain.ManifestService
}

// NewPickupBatchHandler constructs a PickupBatchHandler.
func NewPickupBatchHandler(m domain.ManifestService) *PickupBatchHandler {
	return &PickupBatchHandler{manifest: m}
}

// Handle is the Lambda entry point. Triggered by EventBridge daily.
func (h *PickupBatchHandler) Handle(ctx context.Context) error {
	pickupDate := NextDayPickupSlotIST(time.Now().UTC())
	slog.InfoContext(ctx, "Running daily pickup batch", "pickup_date", pickupDate.Format(time.RFC3339))
	res, err := h.manifest.RunDailyBatch(ctx, pickupDate)
	if err != nil {
		slog.ErrorContext(ctx, "Pickup batch failed", "error", err)
		return err
	}
	slog.InfoContext(ctx, "Pickup batch complete",
		"manifest_id", res.ManifestID,
		"shipment_count", res.ShipmentCount,
		"marked", len(res.ShipmentMarkedIDs),
		"failed", len(res.FailedShipmentIDs))
	return nil
}
