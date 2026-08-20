package dto

import "github.com/handloom/admin/internal/domain"

// CreateOrderRequest represents the order creation request.
type CreateOrderRequest struct {
	CustomerID      string          `json:"customer_id" validate:"required"`
	Items           []OrderItemDTO  `json:"items" validate:"required,min=1,dive"`
	ShippingAddress *domain.Address `json:"shipping_address,omitempty"`
	BillingAddress  *domain.Address `json:"billing_address,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	CouponCode      string          `json:"coupon_code,omitempty"`
}

// OrderItemDTO represents an order item in the request.
type OrderItemDTO struct {
	ProductID  string                 `json:"product_id" validate:"required"`
	Quantity   int                    `json:"quantity" validate:"required,gt=0"`
	QuoteID    *string                `json:"quote_id,omitempty"`
	Dimensions *domain.Dimensions     `json:"dimensions,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *CreateOrderRequest) ToDomain() domain.CreateOrderRequest {
	items := make([]domain.OrderItemInput, len(r.Items))
	for i, item := range r.Items {
		items[i] = domain.OrderItemInput{
			ProductID:        item.ProductID,
			Quantity:         item.Quantity,
			QuoteID:          item.QuoteID,
			CustomDimensions: item.Dimensions,
			Attributes:       item.Attributes,
		}
	}

	// Handle shipping address - domain expects non-pointer
	var shippingAddr domain.Address
	if r.ShippingAddress != nil {
		shippingAddr = *r.ShippingAddress
	}

	// Handle coupon code - domain expects pointer
	var couponCode *string
	if r.CouponCode != "" {
		couponCode = &r.CouponCode
	}

	return domain.CreateOrderRequest{
		CustomerID:      r.CustomerID,
		Items:           items,
		ShippingAddress: shippingAddr,
		BillingAddress:  r.BillingAddress,
		Notes:           r.Notes,
		CouponCode:      couponCode,
	}
}

// UpdateOrderStatusRequest represents the status update request.
type UpdateOrderStatusRequest struct {
	Status domain.OrderStatus `json:"status" validate:"required"`
}

// AddOrderNoteRequest represents the add note request.
type AddOrderNoteRequest struct {
	Note       string `json:"note" validate:"required"`
	IsInternal bool   `json:"is_internal"`
}

// UpdateTrackingRequest represents the tracking update request.
type UpdateTrackingRequest struct {
	TrackingNumber string `json:"tracking_number" validate:"required"`
	Carrier        string `json:"carrier,omitempty"`
}

// CancelOrderRequest represents the cancel order request.
type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// CreateCustomerRequest represents the customer creation request.
type CreateCustomerRequest struct {
	FirstName string          `json:"first_name" validate:"required"`
	LastName  string          `json:"last_name" validate:"required"`
	Email     string          `json:"email" validate:"required,email"`
	Phone     string          `json:"phone,omitempty"`
	Address   *domain.Address `json:"address,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	Notes     string          `json:"notes,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *CreateCustomerRequest) ToDomain() domain.CreateCustomerRequest {
	return domain.CreateCustomerRequest{
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Email:     r.Email,
		Phone:     r.Phone,
		Address:   r.Address,
		Tags:      r.Tags,
		Notes:     r.Notes,
	}
}

// UpdateCustomerRequest represents the customer update request.
type UpdateCustomerRequest struct {
	FirstName *string                `json:"first_name,omitempty"`
	LastName  *string                `json:"last_name,omitempty"`
	Phone     *string                `json:"phone,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Notes     *string                `json:"notes,omitempty"`
	Status    *domain.CustomerStatus `json:"status,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *UpdateCustomerRequest) ToDomain() domain.UpdateCustomerRequest {
	req := domain.UpdateCustomerRequest{}
	if r.FirstName != nil {
		req.FirstName = *r.FirstName
	}
	if r.LastName != nil {
		req.LastName = *r.LastName
	}
	if r.Phone != nil {
		req.Phone = *r.Phone
	}
	if r.Notes != nil {
		req.Notes = *r.Notes
	}
	if r.Status != nil {
		req.Status = *r.Status
	}
	req.Tags = r.Tags
	return req
}
