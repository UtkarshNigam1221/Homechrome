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

// CODReconciliationService pulls daily COD remittance rows from the carrier
// and reconciles them against shipments / orders. Matched rows mark the order
// as remitted; unmatched rows are flagged for manual review.
type CODReconciliationService struct {
	shipmentRepo domain.ShipmentRepository
	orderRepo    domain.OrderRepository
	remRepo      domain.CODRemittanceRepository
	courier      courier.Gateway
	publisher    event.EventPublisher
}

// NewCODReconciliationService creates a CODReconciliationService.
func NewCODReconciliationService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	remRepo domain.CODRemittanceRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
) *CODReconciliationService {
	return &CODReconciliationService{
		shipmentRepo: shipmentRepo,
		orderRepo:    orderRepo,
		remRepo:      remRepo,
		courier:      gw,
		publisher:    pub,
	}
}

// RunDailyPull fetches all carrier remittance rows in [from, to], groups them
// by UTR (one payout per UTR), matches each AWB to a shipment/order, and
// persists one CODRemittance per UTR. Per-entry events (matched / unmatched)
// are published fire-and-forget.
func (s *CODReconciliationService) RunDailyPull(ctx context.Context, from, to time.Time) (*domain.PullResult, error) {
	rows, err := s.courier.FetchCODRemittances(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch COD remittances")
	}
	grouped := map[string][]courier.RemittanceRow{}
	for _, r := range rows {
		grouped[r.UTR] = append(grouped[r.UTR], r)
	}
	res := &domain.PullResult{}
	for utr, entries := range grouped {
		s.processUTRGroup(ctx, utr, entries, res)
	}
	return res, nil
}

// processUTRGroup reconciles all entries that share a UTR, persists a single
// CODRemittance row, and updates res counters in place.
//
// Idempotency: total and remittedAt are computed up-front over all entries so
// that the per-entry update sees identical values across re-pulls. The
// CODRemittance ID is derived deterministically from the UTR (SHA1 UUID), so
// re-pulls upsert the same row instead of inserting a duplicate.
func (s *CODReconciliationService) processUTRGroup(ctx context.Context, utr string, entries []courier.RemittanceRow, res *domain.PullResult) {
	// First pass: compute the aggregate values that must stay stable across re-pulls.
	var total int64
	var remittedAt time.Time
	for _, e := range entries {
		total += e.AmountPaise
		if e.RemittedAt.After(remittedAt) {
			remittedAt = e.RemittedAt
		}
	}
	// Second pass: reconcile each entry against shipments/orders.
	codEntries := make([]domain.CODEntry, 0, len(entries))
	allMatched := true
	for _, e := range entries {
		entry := s.reconcileEntry(ctx, utr, e, remittedAt, res)
		if !entry.Matched {
			allMatched = false
		}
		codEntries = append(codEntries, entry)
	}
	status := domain.CODRemittanceStatusReconciled
	if !allMatched {
		status = domain.CODRemittanceStatusUnmatched
	}
	rem := &domain.CODRemittance{
		ID:            uuid.NewSHA1(uuid.Nil, []byte(utr)).String(),
		RemittanceRef: utr,
		AmountPaise:   total,
		RemittedAt:    remittedAt,
		BankRef:       utr,
		Status:        status,
		Entries:       codEntries,
	}
	if err := s.remRepo.Upsert(ctx, rem); err != nil {
		slog.ErrorContext(ctx, "Failed to upsert remittance", "utr", utr, "error", err)
		return
	}
	res.RemittancesProcessed++
}

// reconcileEntry resolves one carrier remittance row to a shipment/order,
// updates the order on match, emits the appropriate event, and bumps the
// matched/unmatched counters on res.
func (s *CODReconciliationService) reconcileEntry(ctx context.Context, utr string, e courier.RemittanceRow, remittedAt time.Time, res *domain.PullResult) domain.CODEntry {
	matched := false
	orderID := ""
	sh, lookupErr := s.shipmentRepo.GetByAWB(ctx, e.AWB)
	if lookupErr == nil && sh != nil {
		matched = true
		orderID = sh.OrderID
		if upErr := s.orderRepo.UpdateCODRemittance(ctx, sh.OrderID, utr, remittedAt); upErr != nil {
			slog.ErrorContext(ctx, "Failed to mark order COD remitted", "order_id", sh.OrderID, "error", upErr)
			matched = false
		}
	}
	if matched {
		res.EntriesMatched++
		_ = s.publisher.Publish(ctx, event.New(event.CODRemitted, map[string]any{
			"awb":          e.AWB,
			"utr":          utr,
			"order_id":     orderID,
			"amount_paise": e.AmountPaise,
		}))
	} else {
		res.EntriesUnmatched++
		_ = s.publisher.Publish(ctx, event.New(event.CODUnmatched, map[string]any{
			"awb":          e.AWB,
			"utr":          utr,
			"amount_paise": e.AmountPaise,
		}))
	}
	return domain.CODEntry{
		AWB:         e.AWB,
		OrderID:     orderID,
		AmountPaise: e.AmountPaise,
		Matched:     matched,
	}
}

// Ensure interface compliance.
var _ domain.CODReconciliationService = (*CODReconciliationService)(nil)
