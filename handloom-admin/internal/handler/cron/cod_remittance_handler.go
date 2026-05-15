package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/handloom/admin/internal/domain"
)

// CODRemittanceHandler runs the daily COD remittance pull from the carrier.
type CODRemittanceHandler struct {
	cod domain.CODReconciliationService
}

// NewCODRemittanceHandler constructs a CODRemittanceHandler.
func NewCODRemittanceHandler(c domain.CODReconciliationService) *CODRemittanceHandler {
	return &CODRemittanceHandler{cod: c}
}

// Handle is the Lambda entry point. Triggered by EventBridge daily.
func (h *CODRemittanceHandler) Handle(ctx context.Context) error {
	to := time.Now().UTC()
	from := to.Add(-24 * time.Hour)
	slog.InfoContext(ctx, "Running COD remittance pull", "from", from.Format(time.RFC3339), "to", to.Format(time.RFC3339))
	res, err := h.cod.RunDailyPull(ctx, from, to)
	if err != nil {
		slog.ErrorContext(ctx, "COD remittance pull failed", "error", err)
		return err
	}
	slog.InfoContext(ctx, "COD remittance pull complete",
		"remittances_processed", res.RemittancesProcessed,
		"matched", res.EntriesMatched,
		"unmatched", res.EntriesUnmatched)
	return nil
}
