package domain

import "context"

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
	// OTPSendRateLimitKey is the key SendOTP is limited on, "" when exempt.
	OTPSendRateLimitKey(phone string) string
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
	Initiate(ctx context.Context, customerID string, req CheckoutRequest) (*CheckoutResult, error)
	GetPaymentStatus(ctx context.Context, customerID, orderID string) (*PaymentStatusResult, error)
}

// CheckoutRequest contains data for initiating checkout
type CheckoutRequest struct {
	ShippingAddressID string `json:"shipping_address_id" validate:"required"`
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
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
	CheckProviderStatus(ctx context.Context, orderID string) (*ProviderPaymentStatus, error)
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
