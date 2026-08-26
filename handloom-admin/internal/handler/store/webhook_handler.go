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
	refundService   domain.RefundService
	phonePe         phonepe.Gateway
	webhookUsername string
	webhookPassword string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(paymentService domain.PaymentService, refundService domain.RefundService, phonePe phonepe.Gateway, webhookUsername, webhookPassword string) *WebhookHandler {
	return &WebhookHandler{
		paymentService:  paymentService,
		refundService:   refundService,
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

	return r
}

// PhonePeWebhook handles incoming PhonePe Standard Checkout webhook callbacks.
// It reads the raw body, verifies the signature, parses the PhonePe-specific
// payload, and routes to the appropriate domain service method based on event type.
//
// A delivery we cannot accept — bad signature, unparseable body, an event we do
// not handle — is acknowledged 200: redelivering it would change nothing. A
// delivery we accepted but failed to *process* answers 500, so PhonePe retries.
// Acknowledging those was how a paid order could stay PENDING forever: the only
// other thing that settles a payment is the provider re-check.
func (h *WebhookHandler) PhonePeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read PhonePe webhook body", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Verify webhook signature
	if h.webhookUsername != "" && h.webhookPassword != "" {
		authHeader := r.Header.Get("Authorization")
		if !h.phonePe.VerifyWebhookSignature(h.webhookUsername, h.webhookPassword, authHeader) {
			slog.ErrorContext(ctx, "Invalid PhonePe webhook signature")
			response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
			return
		}
	} else {
		slog.WarnContext(ctx, "PhonePe webhook signature verification SKIPPED - credentials not configured")
	}

	// Parse PhonePe-specific payload
	var webhookPayload phonepe.WebhookPayload
	if err := json.Unmarshal(body, &webhookPayload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse PhonePe webhook payload", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
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
	var handleErr error
	switch webhookPayload.Event {
	case "checkout.order.completed":
		handleErr = h.paymentService.HandlePaymentSuccess(ctx, evt)
	case "checkout.order.failed":
		handleErr = h.paymentService.HandlePaymentFailure(ctx, evt)
	case "pg.refund.completed":
		handleErr = h.refundService.HandleRefundCompleted(ctx, webhookPayload.Payload.RefundID)
	case "pg.refund.failed":
		handleErr = h.refundService.HandleRefundFailed(ctx, webhookPayload.Payload.RefundID,
			webhookPayload.Payload.ErrorCode, webhookPayload.Payload.DetailedErrorCode)
	default:
		slog.WarnContext(ctx, "Unhandled PhonePe webhook event", "event", webhookPayload.Event)
	}

	if handleErr != nil {
		// 500, not 200: this delivery was valid and we failed it, so PhonePe
		// redelivering is the only thing that settles the order without a sweep.
		slog.ErrorContext(ctx, "Failed to process PhonePe webhook, asking for redelivery",
			"event", webhookPayload.Event,
			"merchant_order_id", webhookPayload.Payload.MerchantOrderID,
			"error", handleErr)
		response.JSON(w, http.StatusInternalServerError, map[string]string{response.KeyStatus: response.KeyError})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "ok"})
}
