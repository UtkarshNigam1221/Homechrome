package cron

import (
	"context"
	"log/slog"

	"github.com/handloom/admin/internal/domain"
)

// RateRefreshHandler refreshes the carrier rate matrix.
type RateRefreshHandler struct {
	rateTable domain.RateTableService
}

// NewRateRefreshHandler constructs a RateRefreshHandler.
func NewRateRefreshHandler(r domain.RateTableService) *RateRefreshHandler {
	return &RateRefreshHandler{rateTable: r}
}

// Handle is the Lambda entry point. Triggered by EventBridge (and on-demand
// via async Lambda invoke from the admin handler).
func (h *RateRefreshHandler) Handle(ctx context.Context) error {
	slog.InfoContext(ctx, "Running rate matrix refresh")
	res, err := h.rateTable.Refresh(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Rate refresh failed", "error", err)
		return err
	}
	slog.InfoContext(ctx, "Rate refresh complete",
		"rows_updated", res.RowsUpdated,
		"rows_skipped", res.RowsSkipped)
	return nil
}
