package domain

import "context"

// CartRepository defines cart data access operations
type CartRepository interface {
	GetCart(ctx context.Context, cartPK string) (*CartWithItems, error)
	PutCartItem(ctx context.Context, item *CartItem) error
	UpdateCartItem(ctx context.Context, cartPK, productID string, quantity int, totalPrice int64) error
	DeleteCartItem(ctx context.Context, cartPK, productID string) error
	UpdateCartHeader(ctx context.Context, cart *Cart) error
	ClearCart(ctx context.Context, cartPK string) error
}

// PaymentRepository defines payment data access operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
	UpdateStatus(ctx context.Context, id string, status PaymentStatus, updates map[string]interface{}) error
}

// ShipmentRepository defines shipment data access operations
type ShipmentRepository interface {
	Create(ctx context.Context, shipment *Shipment) error
	GetByOrderID(ctx context.Context, orderID string) (*Shipment, error)
	UpdateStatus(ctx context.Context, orderID, shipmentID string, status ShipmentStatus, updates map[string]interface{}) error
}

// OTPRepository defines OTP data access operations
type OTPRepository interface {
	Store(ctx context.Context, otp *OTP) error
	Get(ctx context.Context, phone string) (*OTP, error)
	IncrementAttempts(ctx context.Context, phone string) error
	Delete(ctx context.Context, phone string) error
}

// CustomerTokenStore defines customer refresh token storage
type CustomerTokenStore interface {
	StoreToken(ctx context.Context, customerID, tokenHash string, ttl int64) error
	ValidateToken(ctx context.Context, customerID, tokenHash string) (bool, error)
	RevokeToken(ctx context.Context, customerID, tokenHash string) error
	RevokeAllTokens(ctx context.Context, customerID string) error
	// Records successorHash and shortens the row's TTL, conditional on no successor:
	// one concurrent refresh wins. claimed=false lost; ErrCodeInvalidToken row gone.
	ClaimRotation(ctx context.Context, customerID, tokenHash, successorHash string, graceTTL int64) (claimed bool, err error)

	// RevokeTokensExpiringBefore deletes every one of the customer's tokens
	// whose TTL is at or before cutoff, leaving tokens with a later TTL
	// untouched. A live session's TTL is always days out, so with a
	// near-future cutoff this reaches only rotation's short-lived grace-window
	// predecessors — never a session on another device.
	RevokeTokensExpiringBefore(ctx context.Context, customerID string, cutoff int64) error
}
