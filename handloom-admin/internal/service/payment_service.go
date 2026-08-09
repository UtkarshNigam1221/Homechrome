// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/telemetry"
)

const (
	merchantTxnPrefix = "HC-"
	defaultCurrency   = "INR"

	// Metadata keys stored alongside payment status updates
	metaProviderTxnID    = "provider_transaction_id"
	metaPaymentMethod    = "payment_method"
	metaCompletedAt      = "completed_at"
	metaProviderResponse = "provider_response"
)

// errPaymentAlreadyProcessed is returned when a webhook arrives for a payment
// that has already transitioned past the allowed statuses (idempotent replay).
var errPaymentAlreadyProcessed = fmt.Errorf("payment already processed")

// PaymentService implements domain.PaymentService
type PaymentService struct {
	paymentRepo   domain.PaymentRepository
	orderRepo     domain.OrderRepository
	inventoryRepo domain.InventoryRepository
	cartService   domain.CartService
	customerRepo  domain.CustomerRepository
	phonePe       phonepe.Gateway
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	orderRepo domain.OrderRepository,
	inventoryRepo domain.InventoryRepository,
	cartService domain.CartService,
	customerRepo domain.CustomerRepository,
	phonePe phonepe.Gateway,
) *PaymentService {
	return &PaymentService{
		paymentRepo:   paymentRepo,
		orderRepo:     orderRepo,
		inventoryRepo: inventoryRepo,
		cartService:   cartService,
		customerRepo:  customerRepo,
		phonePe:       phonePe,
	}
}

// InitiatePayment creates a payment record and initiates payment with PhonePe
func (s *PaymentService) InitiatePayment(ctx context.Context, req domain.InitiatePaymentRequest) (*domain.PaymentResponse, error) {
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "initiate")
	span.SetAttribute("entity.type", "payment")
	span.SetAttribute("order.id", req.OrderID)

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
		createErr := errors.Wrap(err, "Failed to create payment record")
		span.EndWithError(createErr)
		return nil, createErr
	}

	redirectURL, err := s.phonePe.InitiatePayment(ctx, merchantTxnID, req.CustomerID, req.Amount, req.OrderID)
	if err != nil {
		if updateErr := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed, map[string]interface{}{
			metaProviderResponse: err.Error(),
		}); updateErr != nil {
			slog.ErrorContext(ctx, "Failed to update payment status to failed", "payment_id", payment.ID, "error", updateErr)
		}
		providerErr := errors.Wrap(err, "Failed to initiate payment with provider")
		span.EndWithError(providerErr)
		return nil, providerErr
	}

	// Gateway accepted the request and returned a redirect URL — record the
	// funnel event. Only PhonePe exists today; the gateway label is hardcoded
	// to keep cardinality bounded.
	metrics.Record(ctx, "payment_initiated", metrics.L{
		metrics.LabelGateway: gatewayPhonePe, metrics.LabelCountry: middleware.GetCountry(ctx),
	})

	span.SetAttribute("entity.id", payment.ID)
	span.SetAttribute("payment.merchant_txn_id", merchantTxnID)
	span.End()
	return &domain.PaymentResponse{
		PaymentID:     payment.ID,
		RedirectURL:   redirectURL,
		MerchantTxnID: merchantTxnID,
	}, nil
}

// HandlePaymentSuccess processes a successful payment webhook event.
func (s *PaymentService) HandlePaymentSuccess(ctx context.Context, evt domain.PaymentWebhookEvent) error {
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "handle_success")
	span.SetAttribute("entity.type", "payment")
	span.SetAttribute("payment.merchant_txn_id", evt.MerchantTxnID)

	payment, err := s.resolvePayment(ctx, evt.MerchantTxnID, domain.PaymentStatusInitiated, domain.PaymentStatusPending)
	if err != nil {
		if err == errPaymentAlreadyProcessed {
			span.End()
			return nil // Idempotent — not an error
		}
		span.EndWithError(err)
		return err
	}

	if err = s.updatePaymentStatus(ctx, payment.ID, domain.PaymentStatusSuccess, evt, map[string]interface{}{
		metaCompletedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		span.EndWithError(err)
		return err
	}

	// Update order to CONFIRMED + PAID
	order, err := s.updateOrderStatus(ctx, payment.OrderID, domain.OrderStatusConfirmed, domain.PaymentStatusPaid, payment.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update order status after payment success", "order_id", payment.OrderID, "error", err)
		span.EndWithError(err)
		return err
	}

	// Clear cart now that payment is confirmed
	if err := s.cartService.ClearCart(ctx, payment.CustomerID); err != nil {
		slog.ErrorContext(ctx, "Failed to clear cart after payment success", "customer_id", payment.CustomerID, "error", err)
	}

	// Emit purchase-completion signals. These were previously emitted at
	// order-create time in CheckoutService.Initiate, which caused failed-payment
	// orders to inflate KPIs. All four signals now fire here, after the order
	// has been confirmed as paid. resolvePayment already returned
	// errPaymentAlreadyProcessed for webhook replays, so we never double-count.
	{
		// Webhook ctx is server-to-server (no CloudFront / X-Geo / X-Utm
		// headers). All visitor attribution flows through fields the order
		// denormalised at checkout-initiate time. Reuse the order returned by
		// updateOrderStatus above rather than re-reading it.
		city := order.City
		if city == "" {
			city = labelUnknown
		}
		country := order.Country
		if country == "" {
			country = labelUnknown
		}
		device := order.DeviceType
		if device == "" {
			device = labelUnknown
		}
		utmSource := order.UTMSource
		if utmSource == "" {
			utmSource = labelUnknown
		}

		// Shared product-purchase signals (product_purchased, coupon_redeemed,
		// first-vs-repeat). First-touch attribution flows through fields the
		// order denormalised at checkout-initiate time.
		recordPurchaseAnalytics(ctx, s.customerRepo, order, purchaseAttribution{
			country:   country,
			city:      city,
			device:    device,
			utmSource: utmSource,
		})

		// payment_completed carries utm_source (for ROAS) + device_type (per-
		// device funnel completion). Both read from the order — webhook ctx
		// is server-to-server with no browser headers.
		metrics.Record(ctx, "payment_completed", metrics.L{
			metrics.LabelGateway:    gatewayPhonePe,
			metrics.LabelCountry:    country,
			metrics.LabelCity:       city,
			metrics.LabelDeviceType: device,
			metrics.LabelUTMSource:  utmSource,
		})

		if !order.CreatedAt.IsZero() {
			metrics.RecordDuration(ctx, "cart_to_payment_duration", time.Since(order.CreatedAt), nil)
		}
	}

	metrics.Record(ctx, "payment_outcome", metrics.L{
		metrics.LabelGateway: gatewayPhonePe, metrics.LabelOutcome: "success", metrics.LabelCountry: middleware.GetCountry(ctx),
	})

	// RecordOrderPlaced moved to CheckoutService.Initiate (at order creation
	// time) so the metric also fires on the DevClient path where this webhook
	// never runs. Don't emit here.

	span.SetAttribute("entity.id", payment.ID)
	span.SetAttribute("order.id", payment.OrderID)
	slog.InfoContext(ctx, "Payment completed", "payment_id", payment.ID, "order_id", payment.OrderID)
	span.End()
	return nil
}

// HandlePaymentFailure processes a failed payment webhook event.
func (s *PaymentService) HandlePaymentFailure(ctx context.Context, evt domain.PaymentWebhookEvent) error {
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "handle_failure")
	span.SetAttribute("entity.type", "payment")
	span.SetAttribute("payment.merchant_txn_id", evt.MerchantTxnID)

	payment, err := s.resolvePayment(ctx, evt.MerchantTxnID, domain.PaymentStatusInitiated, domain.PaymentStatusPending)
	if err != nil {
		if err == errPaymentAlreadyProcessed {
			span.End()
			return nil // Idempotent — not an error
		}
		span.EndWithError(err)
		return err
	}

	if err := s.updatePaymentStatus(ctx, payment.ID, domain.PaymentStatusFailed, evt, nil); err != nil {
		span.EndWithError(err)
		return err
	}

	span.SetAttribute("entity.id", payment.ID)
	span.SetAttribute("order.id", payment.OrderID)
	metrics.Record(ctx, "payment_outcome", metrics.L{
		metrics.LabelGateway: gatewayPhonePe, metrics.LabelOutcome: "failed", metrics.LabelCountry: middleware.GetCountry(ctx),
	})
	s.releaseOrderInventory(ctx, payment.OrderID)

	slog.InfoContext(ctx, "Payment failed", "payment_id", payment.ID, "order_id", payment.OrderID)
	span.End()
	return nil
}

// HandlePaymentPending processes a pending payment webhook event.
// This occurs when PhonePe has received the payment attempt but it's still
// being processed (e.g., UPI mandate pending, bank processing).
func (s *PaymentService) HandlePaymentPending(ctx context.Context, evt domain.PaymentWebhookEvent) error {
	payment, err := s.resolvePayment(ctx, evt.MerchantTxnID, domain.PaymentStatusInitiated)
	if err != nil {
		if err == errPaymentAlreadyProcessed {
			return nil // Idempotent — not an error
		}
		return err
	}

	if err := s.updatePaymentStatus(ctx, payment.ID, domain.PaymentStatusPending, evt, nil); err != nil {
		return err
	}

	// Update order payment status to PENDING (order status stays PENDING)
	if _, err := s.updateOrderStatus(ctx, payment.OrderID, domain.OrderStatusPending, domain.PaymentStatusPending, payment.ID); err != nil {
		slog.ErrorContext(ctx, "Failed to update order status for pending payment", "order_id", payment.OrderID, "error", err)
		return err
	}

	metrics.Record(ctx, "payment_outcome", metrics.L{
		metrics.LabelGateway: gatewayPhonePe, metrics.LabelOutcome: "pending", metrics.LabelCountry: middleware.GetCountry(ctx),
	})
	slog.InfoContext(ctx, "Payment pending", "payment_id", payment.ID, "order_id", payment.OrderID)
	return nil
}

// --- internal helpers ---

// resolvePayment looks up a payment and checks idempotency.
// Returns (nil, nil) if the payment is not in one of the allowed statuses.
func (s *PaymentService) resolvePayment(ctx context.Context, merchantTxnID string, allowedStatuses ...domain.PaymentStatus) (*domain.Payment, error) {
	payment, err := s.paymentRepo.GetByMerchantTxnID(ctx, merchantTxnID)
	if err != nil {
		return nil, errors.NotFound("Payment not found")
	}
	for _, status := range allowedStatuses {
		if payment.Status == status {
			return payment, nil
		}
	}
	slog.InfoContext(ctx, "Payment already processed, skipping", "payment_id", payment.ID, "status", payment.Status)
	return nil, errPaymentAlreadyProcessed
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
// updateOrderStatus fetches the order, applies the new status fields, persists
// it, and returns the updated order so callers can reuse it without a second read.
func (s *PaymentService) updateOrderStatus(ctx context.Context, orderID string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus, paymentID string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get order for status update")
	}

	order.Status = orderStatus
	order.PaymentStatus = paymentStatus
	order.PaymentID = paymentID

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, errors.Wrap(err, "Failed to update order status")
	}
	return order, nil
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
			slog.ErrorContext(ctx, "Failed to release inventory", keyProductID, item.ProductID, "order_id", orderID, "error", err)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: "release"})
		}
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
