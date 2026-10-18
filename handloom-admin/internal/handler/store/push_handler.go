package store

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

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

// Routes returns the public storefront push routes. Each is scoped to the
// endpoint in its own body, so a caller can only touch their own device.
func (h *PushHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// The key is a public constant read on every page load, so it sits outside
	// the limiter; the routes that do outbound work stay behind it.
	r.Get("/vapid-key", h.VapidKey)
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(20, time.Minute))
		r.With(middleware.ValidateJSONTyped[domain.SubscribePushRequest](h.validation)).
			Post("/subscribe", h.Subscribe)
		r.With(middleware.ValidateJSONTyped[domain.PushEndpointRequest](h.validation)).
			Post("/unsubscribe", h.Unsubscribe)
		r.With(middleware.ValidateJSONTyped[domain.PushEndpointRequest](h.validation)).
			Post("/test", h.SendTest)
		r.With(middleware.ValidateJSONTyped[domain.LinkPushRequest](h.validation)).
			Post("/link", h.Link)
		r.With(middleware.ValidateJSONTyped[domain.LinkPushRequest](h.validation)).
			Post("/unlink", h.Unlink)
	})

	return r
}

// VapidKey returns the VAPID application server key.
// GET /api/v1/store/push/vapid-key
func (h *PushHandler) VapidKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
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
	req := middleware.MustGetValidatedBody[domain.PushEndpointRequest](r.Context())

	if err := h.pushService.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "unsubscribed"})
}

// SendTest delivers a test notification to the caller's own endpoint.
// POST /api/v1/store/push/test
func (h *PushHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.PushEndpointRequest](r.Context())

	if err := h.pushService.SendTest(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "sent"})
}

// Link attaches this browser's subscription to the signed-in shopper.
// POST /api/v1/store/push/link
func (h *PushHandler) Link(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.LinkPushRequest](r.Context())
	if err := h.pushService.LinkCustomer(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"linked": true})
}

// Unlink detaches this browser's subscription on sign-out.
// POST /api/v1/store/push/unlink
func (h *PushHandler) Unlink(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.LinkPushRequest](r.Context())
	if err := h.pushService.UnlinkCustomer(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"unlinked": true})
}
