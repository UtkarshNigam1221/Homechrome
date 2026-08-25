package handler

import "github.com/handloom/admin/internal/domain"

// Auth request types

// RefreshTokenRequest is the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// PasswordResetEmailRequest is the request body for requesting a password reset
type PasswordResetEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// User request types

// UpdateUserStatusRequest is the request body for updating a user's status
type UpdateUserStatusRequest struct {
	Status domain.UserStatus `json:"status" validate:"required"`
}

// Order request types

// UpdateOrderStatusRequest is the request body for updating order status
type UpdateOrderStatusRequest struct {
	Status domain.OrderStatus `json:"status" validate:"required"`
}

// AddOrderNoteRequest is the request body for adding a note to an order
type AddOrderNoteRequest struct {
	Note       string `json:"note" validate:"required"`
	IsInternal bool   `json:"is_internal"`
}

// UpdateTrackingRequest is the request body for updating tracking info.
// TrackingURL is rendered as a link on the storefront, so it is constrained to
// http/https — `url` alone would accept a javascript: scheme.
type UpdateTrackingRequest struct {
	TrackingNumber string `json:"tracking_number" validate:"required"`
	Carrier        string `json:"carrier"`
	TrackingURL    string `json:"tracking_url" validate:"omitempty,http_url"`
}

// CancelOrderRequest is the request body for canceling an order
type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// Coupon request types

// ValidateCouponRequest is the request body for validating a coupon.
// No product list: coupons discount the cart, not particular items.
type ValidateCouponRequest struct {
	Code              string `json:"code" validate:"required"`
	CartTotal         int64  `json:"cart_total"`
	CustomerID        string `json:"customer_id"`
	HasAutomaticOffer bool   `json:"has_automatic_offer"`
}

// RedeemCouponRequest is the request body for recording a redemption.
type RedeemCouponRequest struct {
	CouponID   string `json:"coupon_id" validate:"required"`
	OrderID    string `json:"order_id" validate:"required"`
	CustomerID string `json:"customer_id"`
	Discount   int64  `json:"discount"`
}
