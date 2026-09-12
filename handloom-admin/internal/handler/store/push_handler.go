package store

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// PushHandler handles storefront Web Push subscription management.
type PushHandler struct {
	pushService *service.PushService
	validation  *middleware.Validation
}

// NewPushHandler creates a new PushHandler
func NewPushHandler(pushService *service.PushService, validation *middleware.Validation) *PushHandler {
	return &PushHandler{pushService: pushService, validation: validation}
}

// Routes returns the storefront push routes.
//
// These are PUBLIC (no auth middleware) — an anonymous visitor must be able to
// opt in. Every route is scoped to the endpoint in the request body, which the
// caller can only possess if their own browser produced it, so a caller can
// only ever read or change their own device's subscription. Nothing here
// exposes another subscriber's endpoint or keys, and nothing here can send to
// a device other than the one named in the request. Fan-out to all subscribers
// lives on the admin router behind JWT auth, never here.
func (h *PushHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/vapid-key", h.VapidKey)
	r.With(middleware.ValidateJSONTyped[domain.SubscribePushRequest](h.validation)).
		Post("/subscribe", h.Subscribe)
	r.With(middleware.ValidateJSONTyped[domain.UnsubscribePushRequest](h.validation)).
		Post("/unsubscribe", h.Unsubscribe)
	r.With(middleware.ValidateJSONTyped[domain.TestPushRequest](h.validation)).
		Post("/test", h.SendTest)

	return r
}

// VapidKey returns the VAPID application server key.
// GET /api/v1/store/push/vapid-key
func (h *PushHandler) VapidKey(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"public_key": h.pushService.PublicKey(),
	})
}

// Subscribe registers this browser's push endpoint.
// POST /api/v1/store/push/subscribe
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.SubscribePushRequest](r.Context())

	sub, err := h.pushService.Subscribe(r.Context(), *req, r.Header.Get("User-Agent"))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, sub)
}

// Unsubscribe retires this browser's push endpoint.
// POST /api/v1/store/push/unsubscribe
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.UnsubscribePushRequest](r.Context())

	if err := h.pushService.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "unsubscribed"})
}

// SendTest delivers a test notification to the caller's own endpoint.
// POST /api/v1/store/push/test
func (h *PushHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.TestPushRequest](r.Context())

	if err := h.pushService.SendTest(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "sent"})
}
