// Package service implements the business logic layer
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/errors"
)

// PaymentService implements domain.PaymentService
type PaymentService struct {
	paymentRepo   domain.PaymentRepository
	orderRepo     domain.OrderRepository
	inventoryRepo domain.InventoryRepository
	phonePe       phonepe.Gateway
	publisher     event.EventPublisher
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	orderRepo domain.OrderRepository,
	inventoryRepo domain.InventoryRepository,
	phonePe phonepe.Gateway,
	publisher event.EventPublisher,
) *PaymentService {
	return &PaymentService{
		paymentRepo:   paymentRepo,
		orderRepo:     orderRepo,
		inventoryRepo: inventoryRepo,
		phonePe:       phonePe,
		publisher:     publisher,
	}
}

// InitiatePayment creates a payment record and initiates payment with PhonePe
func (s *PaymentService) InitiatePayment(ctx context.Context, req domain.InitiatePaymentRequest) (*domain.PaymentResponse, error) {
	merchantTxnID := "HC-" + uuid.New().String()
	now := time.Now()

	payment := &domain.Payment{
		ID:                    uuid.New().String(),
		OrderID:               req.OrderID,
		CustomerID:            req.CustomerID,
		Amount:                req.Amount,
		Currency:              "INR",
		Status:                domain.PaymentStatusInitiated,
		Provider:              domain.PaymentProviderPhonePe,
		MerchantTransactionID: merchantTxnID,
		InitiatedAt:           now,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		slog.ErrorContext(ctx, "Failed to create payment record", "error", err)
		return nil, errors.Wrap(err, "Failed to create payment record")
	}

	redirectURL, err := s.phonePe.InitiatePayment(ctx, merchantTxnID, req.CustomerID, req.Amount)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to initiate PhonePe payment", "error", err)
		// Update payment status to FAILED since gateway call failed
		_ = s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed, map[string]interface{}{
			"provider_response": err.Error(),
		})
		return nil, errors.Wrap(err, "Failed to initiate payment with provider")
	}

	return &domain.PaymentResponse{
		PaymentID:     payment.ID,
		RedirectURL:   redirectURL,
		MerchantTxnID: merchantTxnID,
	}, nil
}

// HandleWebhook processes a PhonePe webhook callback
func (s *PaymentService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	// Parse the webhook payload to extract the base64-encoded response
	var webhookPayload phonepe.WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse webhook payload", "error", err)
		return errors.BadRequest("Invalid webhook payload")
	}

	// Verify signature
	if !s.phonePe.VerifyWebhookSignature(webhookPayload.Response, signature) {
		slog.ErrorContext(ctx, "Invalid webhook signature")
		return errors.Unauthorized("Invalid webhook signature")
	}

	// Decode the response
	statusResp, err := s.phonePe.DecodeWebhookResponse(webhookPayload.Response)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to decode webhook response", "error", err)
		return errors.Wrap(err, "Failed to decode webhook response")
	}

	merchantTxnID := statusResp.Data.MerchantTransactionID

	// Find payment by merchant transaction ID
	payment, err := s.paymentRepo.GetByMerchantTxnID(ctx, merchantTxnID)
	if err != nil {
		slog.ErrorContext(ctx, "Payment not found for merchant txn ID", "merchant_txn_id", merchantTxnID, "error", err)
		return errors.NotFound("Payment not found")
	}

	// Idempotency: if already processed (not INITIATED), skip
	if payment.Status != domain.PaymentStatusInitiated {
		slog.InfoContext(ctx, "Payment already processed, skipping", "payment_id", payment.ID, "status", payment.Status)
		return nil
	}

	// Determine payment method from instrument type
	paymentMethod := mapInstrumentType(statusResp.Data.PaymentInstrument.Type)

	switch statusResp.Data.State {
	case "COMPLETED":
		return s.handlePaymentSuccess(ctx, payment, statusResp, paymentMethod)
	case "FAILED":
		return s.handlePaymentFailure(ctx, payment, statusResp, paymentMethod)
	default:
		// PENDING or other states - update status to PENDING
		_ = s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusPending, map[string]interface{}{
			"provider_transaction_id": statusResp.Data.TransactionID,
			"payment_method":          string(paymentMethod),
			"provider_response":       statusResp.Code,
		})
		return nil
	}
}

// handlePaymentSuccess processes a successful payment
func (s *PaymentService) handlePaymentSuccess(ctx context.Context, payment *domain.Payment, statusResp *phonepe.StatusResponse, paymentMethod domain.PaymentMethod) error {
	now := time.Now()

	// Update payment status to SUCCESS
	if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusSuccess, map[string]interface{}{
		"provider_transaction_id": statusResp.Data.TransactionID,
		"payment_method":          string(paymentMethod),
		"provider_response":       statusResp.Code,
		"completed_at":            now.Format(time.RFC3339),
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to update payment status to SUCCESS", "error", err)
		return errors.Wrap(err, "Failed to update payment status")
	}

	// Update order status to CONFIRMED and payment_status to PAID
	if err := s.orderRepo.Update(ctx, s.buildOrderStatusUpdate(ctx, payment.OrderID, domain.OrderStatusConfirmed, domain.PaymentStatusPaid, payment.ID)); err != nil {
		slog.ErrorContext(ctx, "Failed to update order status after successful payment", "error", err)
		// Non-fatal: payment is already recorded as success
	}

	if pubErr := s.publisher.Publish(ctx, event.New(event.PaymentReceived, map[string]interface{}{
		"payment_id": payment.ID,
		"order_id":   payment.OrderID,
		"amount":     payment.Amount,
	})); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish payment.received event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Payment completed successfully", "payment_id", payment.ID, "order_id", payment.OrderID)
	return nil
}

// handlePaymentFailure processes a failed payment and releases reserved inventory
func (s *PaymentService) handlePaymentFailure(ctx context.Context, payment *domain.Payment, statusResp *phonepe.StatusResponse, paymentMethod domain.PaymentMethod) error {
	// Update payment status to FAILED
	if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed, map[string]interface{}{
		"provider_transaction_id": statusResp.Data.TransactionID,
		"payment_method":          string(paymentMethod),
		"provider_response":       statusResp.Code,
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to update payment status to FAILED", "error", err)
		return errors.Wrap(err, "Failed to update payment status")
	}

	// Release reserved inventory for the order
	s.releaseOrderInventory(ctx, payment.OrderID)

	if pubErr := s.publisher.Publish(ctx, event.New(event.PaymentFailed, map[string]interface{}{
		"payment_id": payment.ID,
		"order_id":   payment.OrderID,
		"amount":     payment.Amount,
	})); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish payment.failed event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Payment failed", "payment_id", payment.ID, "order_id", payment.OrderID)
	return nil
}

// releaseOrderInventory retrieves the order and releases reserved inventory for each item
func (s *PaymentService) releaseOrderInventory(ctx context.Context, orderID string) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get order for inventory release", "order_id", orderID, "error", err)
		return
	}

	for _, item := range order.Items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, orderID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to release inventory", "product_id", item.ProductID, "order_id", orderID, "error", err)
			// Continue releasing other items even if one fails
		}
	}
}

// buildOrderStatusUpdate fetches the order and sets status fields for update
func (s *PaymentService) buildOrderStatusUpdate(ctx context.Context, orderID string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus, paymentID string) *domain.Order {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get order for status update", "order_id", orderID, "error", err)
		// Return a minimal order for the update attempt
		return &domain.Order{
			ID:            orderID,
			Status:        orderStatus,
			PaymentStatus: paymentStatus,
			PaymentID:     paymentID,
		}
	}

	order.Status = orderStatus
	order.PaymentStatus = paymentStatus
	order.PaymentID = paymentID
	return order
}

// GetByOrderID retrieves payment by order ID
func (s *PaymentService) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return s.paymentRepo.GetByOrderID(ctx, orderID)
}

// GetByMerchantTxnID retrieves payment by merchant transaction ID
func (s *PaymentService) GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*domain.Payment, error) {
	return s.paymentRepo.GetByMerchantTxnID(ctx, merchantTxnID)
}

// RefundPayment is a placeholder for refund functionality
func (s *PaymentService) RefundPayment(_ context.Context, _ string, _ int64, _ string) error {
	return errors.New(errors.ErrCodeInternal, "Refund functionality is not implemented yet")
}

// mapInstrumentType maps PhonePe instrument type to domain PaymentMethod
func mapInstrumentType(instrumentType string) domain.PaymentMethod {
	switch instrumentType {
	case "UPI":
		return domain.PaymentMethodUPI
	case "CARD":
		return domain.PaymentMethodCard
	case "NET_BANKING":
		return domain.PaymentMethodNetBanking
	case "WALLET":
		return domain.PaymentMethodWallet
	default:
		return domain.PaymentMethodUPI
	}
}

// Ensure interface compliance
var _ domain.PaymentService = (*PaymentService)(nil)
