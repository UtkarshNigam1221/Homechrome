// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/errors"
)

const (
	merchantTxnPrefix = "HC-"
	defaultCurrency   = "INR"

	// Metadata keys stored alongside payment status updates
	metaProviderTxnID  = "provider_transaction_id"
	metaPaymentMethod  = "payment_method"
	metaCompletedAt    = "completed_at"
	metaProviderResponse = "provider_response"
)

// PaymentService implements domain.PaymentService
type PaymentService struct {
	paymentRepo   domain.PaymentRepository
	orderRepo     domain.OrderRepository
	inventoryRepo domain.InventoryRepository
	cartService   domain.CartService
	phonePe       phonepe.Gateway
	publisher     event.EventPublisher
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	orderRepo domain.OrderRepository,
	inventoryRepo domain.InventoryRepository,
	cartService domain.CartService,
	phonePe phonepe.Gateway,
	publisher event.EventPublisher,
) *PaymentService {
	return &PaymentService{
		paymentRepo:   paymentRepo,
		orderRepo:     orderRepo,
		inventoryRepo: inventoryRepo,
		cartService:   cartService,
		phonePe:       phonePe,
		publisher:     publisher,
	}
}

// InitiatePayment creates a payment record and initiates payment with PhonePe
func (s *PaymentService) InitiatePayment(ctx context.Context, req domain.InitiatePaymentRequest) (*domain.PaymentResponse, error) {
	merchantTxnID := merchantTxnPrefix + uuid.New().String()

	payment := &domain.Payment{
		ID:                    uuid.New().String(),
		OrderID:               req.OrderID,
		CustomerID:            req.CustomerID,
		Amount:                req.Amount,
		Currency:              defaultCurrency,
		Status:                domain.PaymentStatusInitiated,
		Provider:              domain.PaymentProviderPhonePe,
		MerchantTransactionID: merchantTxnID,
		InitiatedAt:           time.Now(),
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, errors.Wrap(err, "Failed to create payment record")
	}

	redirectURL, err := s.phonePe.InitiatePayment(ctx, merchantTxnID, req.CustomerID, req.Amount, req.OrderID)
	if err != nil {
		_ = s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed, map[string]interface{}{
			metaProviderResponse: err.Error(),
		})
		return nil, errors.Wrap(err, "Failed to initiate payment with provider")
	}

	return &domain.PaymentResponse{
		PaymentID:     payment.ID,
		RedirectURL:   redirectURL,
		MerchantTxnID: merchantTxnID,
	}, nil
}

// HandlePaymentSuccess processes a successful payment webhook event.
func (s *PaymentService) HandlePaymentSuccess(ctx context.Context, evt domain.PaymentWebhookEvent) error {
	payment, err := s.resolvePayment(ctx, evt.MerchantTxnID)
	if err != nil || payment == nil {
		return err
	}

	if err := s.updatePaymentStatus(ctx, payment.ID, domain.PaymentStatusSuccess, evt, map[string]interface{}{
		metaCompletedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	// Update order to CONFIRMED + PAID
	s.updateOrderStatus(ctx, payment.OrderID, domain.OrderStatusConfirmed, domain.PaymentStatusPaid, payment.ID)

	// Clear cart now that payment is confirmed
	if err := s.cartService.ClearCart(ctx, payment.CustomerID); err != nil {
		slog.ErrorContext(ctx, "Failed to clear cart after payment success", "customer_id", payment.CustomerID, "error", err)
	}

	s.publishPaymentEvent(ctx, event.PaymentReceived, payment)
	slog.InfoContext(ctx, "Payment completed", "payment_id", payment.ID, "order_id", payment.OrderID)
	return nil
}

// HandlePaymentFailure processes a failed payment webhook event.
func (s *PaymentService) HandlePaymentFailure(ctx context.Context, evt domain.PaymentWebhookEvent) error {
	payment, err := s.resolvePayment(ctx, evt.MerchantTxnID)
	if err != nil || payment == nil {
		return err
	}

	if err := s.updatePaymentStatus(ctx, payment.ID, domain.PaymentStatusFailed, evt, nil); err != nil {
		return err
	}

	s.releaseOrderInventory(ctx, payment.OrderID)

	s.publishPaymentEvent(ctx, event.PaymentFailed, payment)
	slog.InfoContext(ctx, "Payment failed", "payment_id", payment.ID, "order_id", payment.OrderID)
	return nil
}

// --- internal helpers ---

// resolvePayment looks up a payment and checks idempotency.
// Returns (nil, nil) if already processed.
func (s *PaymentService) resolvePayment(ctx context.Context, merchantTxnID string) (*domain.Payment, error) {
	payment, err := s.paymentRepo.GetByMerchantTxnID(ctx, merchantTxnID)
	if err != nil {
		return nil, errors.NotFound("Payment not found")
	}
	if payment.Status != domain.PaymentStatusInitiated {
		slog.InfoContext(ctx, "Payment already processed, skipping", "payment_id", payment.ID, "status", payment.Status)
		return nil, nil
	}
	return payment, nil
}

// updatePaymentStatus updates the payment record with provider details.
func (s *PaymentService) updatePaymentStatus(ctx context.Context, paymentID string, status domain.PaymentStatus, evt domain.PaymentWebhookEvent, extra map[string]interface{}) error {
	meta := map[string]interface{}{
		metaProviderTxnID: evt.TransactionID,
		metaPaymentMethod: string(mapPaymentMode(evt.PaymentMode)),
	}
	for k, v := range extra {
		meta[k] = v
	}
	if err := s.paymentRepo.UpdateStatus(ctx, paymentID, status, meta); err != nil {
		return errors.Wrap(err, "Failed to update payment status")
	}
	return nil
}

// updateOrderStatus fetches the order and updates its status fields.
func (s *PaymentService) updateOrderStatus(ctx context.Context, orderID string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus, paymentID string) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get order for status update", "order_id", orderID, "error", err)
		return
	}

	order.Status = orderStatus
	order.PaymentStatus = paymentStatus
	order.PaymentID = paymentID

	if err := s.orderRepo.Update(ctx, order); err != nil {
		slog.ErrorContext(ctx, "Failed to update order status", "order_id", orderID, "error", err)
	}
}

// releaseOrderInventory releases reserved stock for each item in an order.
func (s *PaymentService) releaseOrderInventory(ctx context.Context, orderID string) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get order for inventory release", "order_id", orderID, "error", err)
		return
	}
	for _, item := range order.Items {
		if _, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, orderID); err != nil {
			slog.ErrorContext(ctx, "Failed to release inventory", "product_id", item.ProductID, "order_id", orderID, "error", err)
		}
	}
}

// publishPaymentEvent publishes a payment event (fire-and-forget).
func (s *PaymentService) publishPaymentEvent(ctx context.Context, eventType event.EventType, payment *domain.Payment) {
	if err := s.publisher.Publish(ctx, event.New(eventType, map[string]interface{}{
		"payment_id": payment.ID,
		"order_id":   payment.OrderID,
		"amount":     payment.Amount,
	})); err != nil {
		slog.ErrorContext(ctx, "Failed to publish payment event", "event", eventType, "error", err)
	}
}

// CheckProviderStatus fetches the payment status directly from PhonePe for a given order
func (s *PaymentService) CheckProviderStatus(ctx context.Context, orderID string) (*domain.ProviderPaymentStatus, error) {
	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, errors.Wrap(err, "Payment not found for order")
	}

	statusResp, err := s.phonePe.CheckPaymentStatus(ctx, payment.MerchantTransactionID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to check provider payment status")
	}

	result := &domain.ProviderPaymentStatus{
		OrderID:         orderID,
		MerchantTxnID:   payment.MerchantTransactionID,
		ProviderOrderID: statusResp.OrderID,
		ProviderState:   statusResp.State,
		LocalStatus:     string(payment.Status),
		Amount:          statusResp.Amount,
	}

	if len(statusResp.PaymentDetails) > 0 {
		result.PaymentMode = statusResp.PaymentDetails[0].PaymentMode
		result.TransactionID = statusResp.PaymentDetails[0].TransactionID
	}

	return result, nil
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

// mapPaymentMode maps provider payment mode strings to domain PaymentMethod
func mapPaymentMode(mode string) domain.PaymentMethod {
	switch mode {
	case "UPI_INTENT", "UPI_COLLECT", "UPI_QR":
		return domain.PaymentMethodUPI
	case "CARD":
		return domain.PaymentMethodCard
	case "NET_BANKING":
		return domain.PaymentMethodNetBanking
	default:
		return domain.PaymentMethodUPI
	}
}

var _ domain.PaymentService = (*PaymentService)(nil)
