package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// PushHandler handles admin-side Web Push broadcast operations.
type PushHandler struct {
	pushService *service.PushService
	validation  *middleware.Validation
}

// NewPushHandler creates a new PushHandler
func NewPushHandler(pushService *service.PushService, validation *middleware.Validation) *PushHandler {
	return &PushHandler{pushService: pushService, validation: validation}
}

// Routes returns the admin push routes; the router mounts them behind the
// ADMIN role, since broadcasting and reading endpoints are operator-only.
func (h *PushHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/subscribers", h.ListSubscribers)
	r.Get("/broadcasts", h.ListBroadcasts)
	r.With(middleware.ValidateJSONTyped[domain.BroadcastPushRequest](h.validation)).
		Post("/broadcast", h.Broadcast)

	return r
}

// ListSubscribers returns a page of push subscriptions.
// GET /admin/push/subscribers?status=ACTIVE
func (h *PushHandler) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	status := domain.PushSubscriptionStatus(r.URL.Query().Get("status"))

	result, err := h.pushService.ListSubscriptions(r.Context(), status, parsePagination(r))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ListBroadcasts returns recent broadcast history.
// GET /admin/push/broadcasts?limit=20
func (h *PushHandler) ListBroadcasts(w http.ResponseWriter, r *http.Request) {
	broadcasts, err := h.pushService.ListBroadcasts(r.Context(), int32(parsePagination(r).Limit))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"broadcasts": broadcasts})
}

// Broadcast sends a notification to every active subscriber.
// POST /admin/push/broadcast
func (h *PushHandler) Broadcast(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.BroadcastPushRequest](r.Context())

	result, err := h.pushService.Broadcast(r.Context(), *req, middleware.GetCreatedBy(r.Context()))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
