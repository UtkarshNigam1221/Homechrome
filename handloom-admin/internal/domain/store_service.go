package domain

import "context"

// CustomerAuthService defines customer authentication operations
type CustomerAuthService interface {
	SendOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, code string) (*Customer, *TokenPair, bool, error)
	RefreshToken(ctx context.Context, refreshToken string) (*Customer, *TokenPair, error)
	Logout(ctx context.Context, customerID, refreshToken string) error
}

// CartService defines shopping cart operations
type CartService interface {
	GetCart(ctx context.Context, customerID string) (*CartWithItems, error)
	AddItem(ctx context.Context, customerID string, req AddCartItemRequest) (*CartWithItems, error)
	UpdateItemQuantity(ctx context.Context, customerID, productID string, quantity int) (*CartWithItems, error)
	RemoveItem(ctx context.Context, customerID, productID string) (*CartWithItems, error)
	ClearCart(ctx context.Context, customerID string) error
	MergeGuestCart(ctx context.Context, customerID string, items []AddCartItemRequest) (*CartWithItems, error)
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

// PaymentService defines payment operations
type PaymentService interface {
	InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*PaymentResponse, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
	RefundPayment(ctx context.Context, paymentID string, amount int64, reason string) error
}

// PaymentResponse contains the result of initiating a payment
type PaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	RedirectURL   string `json:"redirect_url"`
	MerchantTxnID string `json:"merchant_txn_id"`
}

// ShippingService defines shipping operations
type ShippingService interface {
	CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightGrams int) (*ServiceabilityResult, error)
	CreateShipment(ctx context.Context, order *Order) (*Shipment, error)
	TrackShipment(ctx context.Context, orderID string) (*Shipment, error)
	HandleWebhook(ctx context.Context, payload []byte, token string) error
}
