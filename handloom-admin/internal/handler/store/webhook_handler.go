// Package store implements public storefront HTTP handlers.
package store

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
)

// WebhookHandler handles incoming webhook callbacks from external providers.
type WebhookHandler struct {
	paymentService domain.PaymentService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(paymentService domain.PaymentService) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
	}
}

// Routes returns the webhook routes. These routes are unauthenticated;
// verification is performed via provider-specific signature checks.
func (h *WebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/phonepe", h.PhonePeWebhook)
	r.Post("/shiprocket", h.ShiprocketWebhook)

	return r
}

// PhonePeWebhook handles incoming PhonePe payment callbacks.
// It reads the raw request body and X-VERIFY header, delegates to
// the payment service for signature verification and processing,
// and always returns 200 OK so PhonePe does not retry indefinitely.
func (h *WebhookHandler) PhonePeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read PhonePe webhook body", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{"status": "error"})
		return
	}
	defer func() { _ = r.Body.Close() }()

	xVerifyHeader := r.Header.Get("X-VERIFY")

	if err := h.paymentService.HandleWebhook(ctx, body, xVerifyHeader); err != nil {
		slog.ErrorContext(ctx, "failed to process PhonePe webhook", "error", err)
		// Still return 200 to acknowledge receipt; PhonePe requires a quick 200 response
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ShiprocketWebhook is a placeholder for Shiprocket shipping callbacks.
// It acknowledges receipt and returns 200 OK.
func (h *WebhookHandler) ShiprocketWebhook(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
