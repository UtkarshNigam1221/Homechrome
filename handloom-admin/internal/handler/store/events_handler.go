package store

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// EventsHandler handles frontend tracking event ingestion for the storefront.
type EventsHandler struct {
	eventsRepo domain.EventsRepository
	validation *middleware.Validation
	logger     *logger.Logger
}

// NewEventsHandler creates a new EventsHandler.
func NewEventsHandler(
	eventsRepo domain.EventsRepository,
	validation *middleware.Validation,
	logger *logger.Logger,
) *EventsHandler {
	return &EventsHandler{
		eventsRepo: eventsRepo,
		validation: validation,
		logger:     logger,
	}
}

// Routes returns the events routes.
// This endpoint is PUBLIC (no auth middleware) — tracking works for anonymous visitors.
func (h *EventsHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.StoreEventBatch](h.validation)).
		Post("/", h.IngestEvents)

	return r
}

// IngestEvents handles POST / — accepts a batch of frontend tracking events.
func (h *EventsHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	batch := middleware.MustGetValidatedBody[domain.StoreEventBatch](ctx)

	// Filter out events older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	valid := make([]domain.StoreEvent, 0, len(batch.Events))
	for _, evt := range batch.Events {
		if evt.Timestamp.After(cutoff) {
			valid = append(valid, evt)
		}
	}

	if len(valid) == 0 {
		response.JSON(w, http.StatusAccepted, map[string]int{"accepted": 0})
		return
	}

	// Write raw events (best-effort, don't fail the request)
	if err := h.eventsRepo.BatchWriteEvents(ctx, valid); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("failed to write raw events")
	}

	// TODO: Phase 5 — update live counters on DASHBOARD#CURRENT

	response.JSON(w, http.StatusAccepted, map[string]int{"accepted": len(valid)})
}
