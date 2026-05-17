// Package handler implements HTTP handlers
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler/cron"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// ShippingAdminHandler exposes admin-side shipping operations: rate matrix,
// COD remittances, NDR queue, and daily pickup batches.
type ShippingAdminHandler struct {
	rateRepo      domain.ShippingRateRepository
	rateTable     domain.RateTableService
	remRepo       domain.CODRemittanceRepository
	ndr           domain.NDRService
	shipmentRepo  domain.ShipmentRepository
	manifest      domain.ManifestService
	validation    *middleware.Validation
	rateRefreshFn func(ctx context.Context) error
}

// NewShippingAdminHandler creates a new ShippingAdminHandler.
func NewShippingAdminHandler(
	rateRepo domain.ShippingRateRepository,
	rateTable domain.RateTableService,
	remRepo domain.CODRemittanceRepository,
	ndr domain.NDRService,
	shipmentRepo domain.ShipmentRepository,
	manifest domain.ManifestService,
	validation *middleware.Validation,
	rateRefreshFn func(ctx context.Context) error,
) *ShippingAdminHandler {
	return &ShippingAdminHandler{
		rateRepo:      rateRepo,
		rateTable:     rateTable,
		remRepo:       remRepo,
		ndr:           ndr,
		shipmentRepo:  shipmentRepo,
		manifest:      manifest,
		validation:    validation,
		rateRefreshFn: rateRefreshFn,
	}
}

// Routes returns the admin shipping routes.
func (h *ShippingAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/rates", h.listRates)
	r.Patch("/rates/{zone}/{slab}", h.updateRate)
	r.Post("/rates/refresh", h.triggerRateRefresh)
	r.Get("/cod-remittances", h.listRemittances)
	r.Get("/cod-remittances/{id}", h.getRemittance)
	r.Get("/ndr-queue", h.listNDRQueue)
	r.Post("/shipments/{id}/ndr-action", h.ndrAction)
	r.Get("/pickup-batches", h.listBatches)
	r.Post("/pickup-batches/run", h.runBatch)
	return r
}

func (h *ShippingAdminHandler) listRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.rateRepo.ListAll(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, rates)
}

func (h *ShippingAdminHandler) updateRate(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	slabStr := chi.URLParam(r, "slab")
	slab, err := strconv.Atoi(slabStr)
	if err != nil {
		response.Error(w, errors.BadRequest("Invalid weight slab"))
		return
	}
	var body struct {
		PrepaidPaise int64 `json:"prepaid_paise"`
		CODPaise     int64 `json:"cod_paise"`
		RTOPaise     int64 `json:"rto_paise"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, errors.BadRequest("Invalid JSON"))
		return
	}
	if body.PrepaidPaise < 0 || body.CODPaise < 0 || body.RTOPaise < 0 {
		response.Error(w, errors.BadRequest("Rates must be non-negative"))
		return
	}
	rate := &domain.ShippingRate{
		Zone:            zone,
		WeightSlabGrams: slab,
		PrepaidPaise:    body.PrepaidPaise,
		CODPaise:        body.CODPaise,
		RTOPaise:        body.RTOPaise,
		Source:          domain.RateSourceManualOverride,
	}
	if err := h.rateRepo.Upsert(r.Context(), rate); err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, rate)
}

func (h *ShippingAdminHandler) triggerRateRefresh(w http.ResponseWriter, r *http.Request) {
	if h.rateRefreshFn == nil {
		if _, err := h.rateTable.Refresh(r.Context()); err != nil {
			response.Error(w, err)
			return
		}
		response.Success(w, map[string]any{"status": "refreshed_sync"})
		return
	}
	if err := h.rateRefreshFn(r.Context()); err != nil {
		response.Error(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    map[string]any{"status": "refresh_queued"},
	})
}

func (h *ShippingAdminHandler) listRemittances(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(domain.CODRemittanceStatusReconciled)
	}
	rems, err := h.remRepo.ListByStatus(r.Context(), domain.CODRemittanceStatus(status), 100)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, rems)
}

func (h *ShippingAdminHandler) getRemittance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rem, err := h.remRepo.Get(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, rem)
}

func (h *ShippingAdminHandler) listNDRQueue(w http.ResponseWriter, r *http.Request) {
	shipments, err := h.shipmentRepo.QueryByPriorityStatus(r.Context(), domain.PriorityNormal, domain.ShipmentStatusNDREscalated, 100)
	if err != nil {
		response.Error(w, err)
		return
	}
	pri, perr := h.shipmentRepo.QueryByPriorityStatus(r.Context(), domain.PriorityPriority, domain.ShipmentStatusNDREscalated, 100)
	if perr != nil {
		slog.ErrorContext(r.Context(), "Failed to query priority NDR queue", "error", perr)
		response.Error(w, errors.Wrap(perr, "Failed to query NDR queue"))
		return
	}
	shipments = append(shipments, pri...)
	response.Success(w, shipments)
}

// ndrAction dispatches an admin action against an NDR-escalated shipment.
// The URL {id} is the shipment AWB (the NDR queue exposes AWB per row, and
// the courier gateway is keyed on AWB). Supported actions:
//   - REATTEMPT       — request another delivery attempt from the carrier
//   - RTO             — return-to-origin
//   - MARK_CONTACTED  — DB-only flag that the customer was reached
func (h *ShippingAdminHandler) ndrAction(w http.ResponseWriter, r *http.Request) {
	awb := chi.URLParam(r, "id")
	if awb == "" || awb == "undefined" || awb == "null" {
		response.Error(w, errors.BadRequest("AWB is required"))
		return
	}
	var body struct {
		Action string `json:"action"`
		Note   string `json:"note,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, errors.BadRequest("Invalid JSON"))
		return
	}
	action := domain.NDRAdminAction(strings.ToUpper(body.Action))
	switch action {
	case domain.NDRAdminActionReattempt, domain.NDRAdminActionRTO, domain.NDRAdminActionMarkContacted:
	default:
		response.Error(w, errors.BadRequest("Unsupported NDR action"))
		return
	}
	adminID := getUserIDFromContext(r.Context())
	if err := h.ndr.HandleAdminAction(r.Context(), awb, action, body.Note, adminID); err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, map[string]any{"status": "ok", "action": action})
}

func (h *ShippingAdminHandler) listBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.manifest.ListBatches(r.Context(), 100)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, batches)
}

func (h *ShippingAdminHandler) runBatch(w http.ResponseWriter, r *http.Request) {
	pickupDate := cron.NextDayPickupSlotIST(time.Now().UTC())
	res, err := h.manifest.RunDailyBatch(r.Context(), pickupDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, res)
}
