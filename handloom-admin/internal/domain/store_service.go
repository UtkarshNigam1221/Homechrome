package domain

import (
	"context"
	"net/http"
	"time"
)

// SMSGateway defines the SMS sending interface
type SMSGateway interface {
	SendOTP(ctx context.Context, phone, code string) error
}

// CustomerTokenClaims holds validated customer JWT claims
type CustomerTokenClaims struct {
	CustomerID string `json:"customer_id"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
}

// CustomerAuthService defines customer authentication operations
type CustomerAuthService interface {
	SendOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, code string) (*Customer, *TokenPair, bool, error)
	RefreshToken(ctx context.Context, refreshToken string) (*Customer, *TokenPair, error)
	Logout(ctx context.Context, customerID, refreshToken string) error
	ValidateCustomerToken(ctx context.Context, token string) (*CustomerTokenClaims, error)
}

// CartService defines shopping cart operations
type CartService interface {
	GetCart(ctx context.Context, cartOwner string, isGuest bool) (*CartWithItems, error)
	AddItem(ctx context.Context, cartOwner string, isGuest bool, req AddCartItemRequest) (*CartWithItems, error)
	UpdateItemQuantity(ctx context.Context, cartOwner string, isGuest bool, productID string, quantity int) (*CartWithItems, error)
	RemoveItem(ctx context.Context, cartOwner string, isGuest bool, productID string) (*CartWithItems, error)
	ClearCart(ctx context.Context, cartOwner string) error
	MergeGuestCart(ctx context.Context, customerID string, items []AddCartItemRequest) (*CartWithItems, error)
	MergeGuestSession(ctx context.Context, customerID, guestSessionID string) error
}

// CheckoutService defines checkout operations
type CheckoutService interface {
	CheckServiceability(ctx context.Context, customerID, pincode string) (*ServiceabilityResult, error)
	Initiate(ctx context.Context, customerID string, req CheckoutRequest) (*CheckoutResult, error)
	GetPaymentStatus(ctx context.Context, customerID, orderID string) (*PaymentStatusResult, error)
}

// CheckoutRequest contains data for initiating checkout
type CheckoutRequest struct {
	ShippingAddressID string `json:"shipping_address_id" validate:"required"`
	CourierID         *int   `json:"courier_id,omitempty"`
}

// CheckoutResult contains the result of a checkout initiation
type CheckoutResult struct {
	Order         *Order `json:"order"`
	RedirectURL   string `json:"redirect_url"`
	MerchantTxnID string `json:"merchant_txn_id"`
}

// PaymentStatusResult contains the current payment status for an order
type PaymentStatusResult struct {
	PaymentStatus PaymentStatus `json:"payment_status"`
	Order         *Order        `json:"order"`
}

// PaymentWebhookEvent is a provider-agnostic payment webhook event.
// The handler translates provider-specific payloads into this struct.
type PaymentWebhookEvent struct {
	MerchantTxnID string
	TransactionID string
	PaymentMode   string // provider-specific mode, mapped to PaymentMethod by service
}

// PaymentService defines payment operations
type PaymentService interface {
	InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*PaymentResponse, error)
	HandlePaymentSuccess(ctx context.Context, evt PaymentWebhookEvent) error
	HandlePaymentFailure(ctx context.Context, evt PaymentWebhookEvent) error
	HandlePaymentPending(ctx context.Context, evt PaymentWebhookEvent) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
	CheckProviderStatus(ctx context.Context, orderID string) (*ProviderPaymentStatus, error)
	RefundPayment(ctx context.Context, paymentID string, amount int64, reason string) error
}

// ProviderPaymentStatus contains payment status from the payment provider
type ProviderPaymentStatus struct {
	OrderID         string `json:"order_id"`
	MerchantTxnID   string `json:"merchant_txn_id"`
	ProviderOrderID string `json:"provider_order_id"`
	ProviderState   string `json:"provider_state"`
	LocalStatus     string `json:"local_status"`
	Amount          int64  `json:"amount"`
	PaymentMode     string `json:"payment_mode,omitempty"`
	TransactionID   string `json:"transaction_id,omitempty"`
}

// PaymentResponse contains the result of initiating a payment
type PaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	RedirectURL   string `json:"redirect_url"`
	MerchantTxnID string `json:"merchant_txn_id"`
}

// ShippingService defines shipping operations
type ShippingService interface {
	CheckServiceability(ctx context.Context, pincode string, weightGrams int) (*ServiceabilityResult, error)
	CreateShipment(ctx context.Context, order *Order, priority ShipmentPriority) (*Shipment, error)
	TrackShipment(ctx context.Context, orderID string) (*Shipment, error)
	HandleWebhook(ctx context.Context, body []byte, headers http.Header) error
}

// ManifestService defines manifest + pickup orchestration for shipments.
type ManifestService interface {
	CreatePerOrderManifest(ctx context.Context, shipment *Shipment) error
	RunDailyBatch(ctx context.Context, pickupDate time.Time) (*BatchResult, error)
}

// BatchResult summarizes a batch manifest run.
//
// ShipmentCount is the number of shipments attempted in this batch (i.e. the
// number of AWBs handed to the carrier's manifest endpoint). ShipmentMarkedIDs
// lists shipments that were successfully transitioned to MANIFESTED in
// DynamoDB. FailedShipmentIDs lists shipments whose status update failed after
// the manifest was already scheduled with the carrier — these need manual
// reconciliation and must be surfaced to operators.
type BatchResult struct {
	ManifestID        string   `json:"manifest_id"`
	ShipmentCount     int      `json:"shipment_count"`
	ShipmentMarkedIDs []string `json:"shipment_marked_ids"`
	FailedShipmentIDs []string `json:"failed_shipment_ids"`
}

// NDRService handles Non-Delivery Report events: auto re-attempt up to a
// configured limit, then escalate for manual action / RTO.
type NDRService interface {
	HandleNDREvent(ctx context.Context, awb, reason string) error
}

// CODReconciliationService pulls daily COD remittance rows from the carrier
// and reconciles them against shipments / orders.
type CODReconciliationService interface {
	RunDailyPull(ctx context.Context, from, to time.Time) (*PullResult, error)
}

// PullResult summarizes a COD reconciliation run.
type PullResult struct {
	RemittancesProcessed int
	EntriesMatched       int
	EntriesUnmatched     int
}

// RateTableService refreshes the carrier rate matrix and resolves shipping
// charges for a given pincode + weight + payment mode.
type RateTableService interface {
	Refresh(ctx context.Context) (*RefreshResult, error)
	Lookup(ctx context.Context, pincode string, weightGrams int, mode PaymentMode) (int64, error)
}

// RefreshResult summarizes a rate matrix refresh.
type RefreshResult struct {
	RowsUpdated int
	RowsSkipped int
	Errors      []string
}

// ReturnService manages admin-initiated customer returns: creates a reverse
// pickup with the courier, tracks lifecycle status, and orchestrates refunds.
type ReturnService interface {
	Create(ctx context.Context, orderID string, req CreateReturnRequest, adminID string) (*ReturnRequest, error)
	Cancel(ctx context.Context, returnID string, adminID string) error
	ProcessRefund(ctx context.Context, returnID string, amountPaise int64, adminID string) error
	HandleReverseWebhook(ctx context.Context, awb string, status ReturnStatus) error
}

// CreateReturnRequest is the admin-supplied payload for creating a return.
type CreateReturnRequest struct {
	Items  []ReturnItem
	Reason string
}
