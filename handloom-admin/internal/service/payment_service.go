// Package service implements the business logic layer
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// PaymentService implements domain.PaymentService
type PaymentService struct {
	paymentRepo   domain.PaymentRepository
	orderRepo     domain.OrderRepository
	inventoryRepo domain.InventoryRepository
	phonePe       *phonepe.Client
	logger        *logger.Logger
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	orderRepo domain.OrderRepository,
	inventoryRepo domain.InventoryRepository,
	phonePe *phonepe.Client,
	logger *logger.Logger,
) *PaymentService {
	return &PaymentService{
		paymentRepo:   paymentRepo,
		orderRepo:     orderRepo,
		inventoryRepo: inventoryRepo,
		phonePe:       phonePe,
		logger:        logger,
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
		s.logger.WithContext(ctx).WithError(err).Error("Failed to create payment record")
		return nil, errors.Wrap(err, "Failed to create payment record")
	}

	redirectURL, err := s.phonePe.InitiatePayment(ctx, merchantTxnID, req.CustomerID, req.Amount)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to initiate PhonePe payment")
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
		s.logger.WithContext(ctx).WithError(err).Error("Failed to parse webhook payload")
		return errors.BadRequest("Invalid webhook payload")
	}

	// Verify signature
	if !s.phonePe.VerifyWebhookSignature(webhookPayload.Response, signature) {
		s.logger.WithContext(ctx).Error("Invalid webhook signature")
		return errors.Unauthorized("Invalid webhook signature")
	}

	// Decode the response
	statusResp, err := s.phonePe.DecodeWebhookResponse(webhookPayload.Response)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to decode webhook response")
		return errors.Wrap(err, "Failed to decode webhook response")
	}

	merchantTxnID := statusResp.Data.MerchantTransactionID

	// Find payment by merchant transaction ID
	payment, err := s.paymentRepo.GetByMerchantTxnID(ctx, merchantTxnID)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Errorf("Payment not found for merchant txn ID: %s", merchantTxnID)
		return errors.NotFound("Payment not found")
	}

	// Idempotency: if already processed (not INITIATED), skip
	if payment.Status != domain.PaymentStatusInitiated {
		s.logger.WithContext(ctx).Infof("Payment %s already processed with status %s, skipping", payment.ID, payment.Status)
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
			"payment_method":         string(paymentMethod),
			"provider_response":      statusResp.Code,
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
		"payment_method":         string(paymentMethod),
		"provider_response":      statusResp.Code,
		"completed_at":           now.Format(time.RFC3339),
	}); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to update payment status to SUCCESS")
		return errors.Wrap(err, "Failed to update payment status")
	}

	// Update order status to CONFIRMED and payment_status to PAID
	if err := s.orderRepo.Update(ctx, s.buildOrderStatusUpdate(ctx, payment.OrderID, domain.OrderStatusConfirmed, domain.PaymentStatusPaid, payment.ID)); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to update order status after successful payment")
		// Non-fatal: payment is already recorded as success
	}

	s.logger.WithContext(ctx).Infof("Payment %s completed successfully for order %s", payment.ID, payment.OrderID)
	return nil
}

// handlePaymentFailure processes a failed payment and releases reserved inventory
func (s *PaymentService) handlePaymentFailure(ctx context.Context, payment *domain.Payment, statusResp *phonepe.StatusResponse, paymentMethod domain.PaymentMethod) error {
	// Update payment status to FAILED
	if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed, map[string]interface{}{
		"provider_transaction_id": statusResp.Data.TransactionID,
		"payment_method":         string(paymentMethod),
		"provider_response":      statusResp.Code,
	}); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to update payment status to FAILED")
		return errors.Wrap(err, "Failed to update payment status")
	}

	// Release reserved inventory for the order
	s.releaseOrderInventory(ctx, payment.OrderID)

	s.logger.WithContext(ctx).Infof("Payment %s failed for order %s", payment.ID, payment.OrderID)
	return nil
}

// releaseOrderInventory retrieves the order and releases reserved inventory for each item
func (s *PaymentService) releaseOrderInventory(ctx context.Context, orderID string) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Errorf("Failed to get order %s for inventory release", orderID)
		return
	}

	for _, item := range order.Items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, orderID)
		if err != nil {
			s.logger.WithContext(ctx).WithError(err).Errorf("Failed to release inventory for product %s in order %s", item.ProductID, orderID)
			// Continue releasing other items even if one fails
		}
	}
}

// buildOrderStatusUpdate fetches the order and sets status fields for update
func (s *PaymentService) buildOrderStatusUpdate(ctx context.Context, orderID string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus, paymentID string) *domain.Order {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Errorf("Failed to get order %s for status update", orderID)
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
