// Package store implements public storefront HTTP handlers.
package store

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/response"
)

// WebhookHandler handles incoming webhook callbacks from external providers.
type WebhookHandler struct {
	paymentService  domain.PaymentService
	phonePe         phonepe.Gateway
	webhookUsername string
	webhookPassword string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(paymentService domain.PaymentService, phonePe phonepe.Gateway, webhookUsername, webhookPassword string) *WebhookHandler {
	return &WebhookHandler{
		paymentService:  paymentService,
		phonePe:         phonePe,
		webhookUsername: webhookUsername,
		webhookPassword: webhookPassword,
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

// PhonePeWebhook handles incoming PhonePe Standard Checkout webhook callbacks.
// It reads the raw body, verifies the signature, parses the PhonePe-specific
// payload, and routes to the appropriate domain service method based on event type.
// Always returns 200 OK to acknowledge receipt.
func (h *WebhookHandler) PhonePeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read PhonePe webhook body", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{"status": "error"})
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Verify webhook signature
	if h.webhookUsername != "" && h.webhookPassword != "" {
		authHeader := r.Header.Get("Authorization")
		if !h.phonePe.VerifyWebhookSignature(h.webhookUsername, h.webhookPassword, authHeader) {
			slog.ErrorContext(ctx, "Invalid PhonePe webhook signature")
			response.JSON(w, http.StatusOK, map[string]string{"status": "error"})
			return
		}
	} else {
		slog.WarnContext(ctx, "PhonePe webhook signature verification SKIPPED - credentials not configured")
	}

	// Parse PhonePe-specific payload
	var webhookPayload phonepe.WebhookPayload
	if err := json.Unmarshal(body, &webhookPayload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse PhonePe webhook payload", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{"status": "error"})
		return
	}

	slog.InfoContext(ctx, "Received PhonePe webhook",
		"event", webhookPayload.Event,
		"merchant_order_id", webhookPayload.Payload.MerchantOrderID,
		"state", webhookPayload.Payload.State,
	)

	// Map PhonePe event to domain payment event
	evt := domain.PaymentWebhookEvent{
		MerchantTxnID: webhookPayload.Payload.MerchantOrderID,
	}
	if len(webhookPayload.Payload.PaymentDetails) > 0 {
		detail := webhookPayload.Payload.PaymentDetails[0]
		evt.TransactionID = detail.TransactionID
		evt.PaymentMode = detail.PaymentMode
	}

	// Route on PhonePe event type
	switch webhookPayload.Event {
	case "checkout.order.completed":
		if err := h.paymentService.HandlePaymentSuccess(ctx, evt); err != nil {
			slog.ErrorContext(ctx, "failed to handle payment success", "error", err)
		}
	case "checkout.order.failed":
		if err := h.paymentService.HandlePaymentFailure(ctx, evt); err != nil {
			slog.ErrorContext(ctx, "failed to handle payment failure", "error", err)
		}
	case "checkout.order.pending":
		if err := h.paymentService.HandlePaymentPending(ctx, evt); err != nil {
			slog.ErrorContext(ctx, "failed to handle payment pending", "error", err)
		}
	default:
		slog.WarnContext(ctx, "Unhandled PhonePe webhook event", "event", webhookPayload.Event)
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ShiprocketWebhook is a placeholder for Shiprocket shipping callbacks.
// It acknowledges receipt and returns 200 OK.
func (h *WebhookHandler) ShiprocketWebhook(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
