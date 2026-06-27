package store

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/response"
)

// eventRecorder maps a validated storefront beacon event to its PG metrics.
// Implemented by service.StoreEventService.
type eventRecorder interface {
	Record(ctx context.Context, evt domain.StoreEvent)
}

// EventsHandler handles frontend tracking event ingestion for the storefront.
type EventsHandler struct {
	validation *middleware.Validation
	events     eventRecorder
}

// NewEventsHandler creates a new EventsHandler.
func NewEventsHandler(validation *middleware.Validation, events eventRecorder) *EventsHandler {
	return &EventsHandler{validation: validation, events: events}
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
// Each valid event is routed to PG metric_counters by the StoreEventService.
// Raw events are no longer persisted (DDB events table dropped in M39).
func (h *EventsHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	batch := middleware.MustGetValidatedBody[domain.StoreEventBatch](ctx)

	// Filter out events older than 24 hours and with unknown event types.
	cutoff := time.Now().Add(-24 * time.Hour)
	valid := make([]domain.StoreEvent, 0, len(batch.Events))
	for _, evt := range batch.Events {
		if evt.Timestamp.After(cutoff) && domain.IsValidStoreEventType(evt.EventType) {
			valid = append(valid, evt)
		}
	}

	if len(valid) == 0 {
		response.JSON(w, http.StatusAccepted, map[string]int{"accepted": 0})
		return
	}

	for _, evt := range valid {
		h.events.Record(ctx, evt)
	}

	response.JSON(w, http.StatusAccepted, map[string]int{"accepted": len(valid)})
}
