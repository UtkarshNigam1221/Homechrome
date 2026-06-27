package domain

import (
	"context"
)

//go:generate mockgen -source=order_repository.go -destination=../mocks/order_repository_mock.go -package=mocks

// OrderRepository defines the interface for order data access
type OrderRepository interface {
	// Create creates a new order
	Create(ctx context.Context, order *Order) error

	// GetByID retrieves an order by ID
	GetByID(ctx context.Context, id string) (*Order, error)

	// GetByOrderNumber retrieves an order by order number
	GetByOrderNumber(ctx context.Context, orderNumber string) (*Order, error)

	// Update updates an existing order
	Update(ctx context.Context, order *Order) error

	// List retrieves orders with filters
	List(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)

	// GetByCustomer retrieves orders by customer ID
	GetByCustomer(ctx context.Context, customerID string, pagination PaginationRequest) (*ListOrdersResponse, error)

	// UpdateStatus updates order status
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedBy string) error

	// AddNote adds a note to an order
	AddNote(ctx context.Context, id string, note OrderNote) error

	// UpdateTracking updates tracking information
	UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string) error
}

// ListOrdersRequest contains parameters for listing orders
type ListOrdersRequest struct {
	PaginationRequest
	Status        *OrderStatus   `json:"status,omitempty"`
	PaymentStatus *PaymentStatus `json:"payment_status,omitempty"`
	CustomerID    *string        `json:"customer_id,omitempty"`
	Search        string         `json:"search,omitempty"`
	StartDate     *string        `json:"start_date,omitempty"`
	EndDate       *string        `json:"end_date,omitempty"`
}

// ListOrdersResponse contains the list of orders
type ListOrdersResponse struct {
	Orders     []*Order           `json:"orders"`
	Pagination PaginationResponse `json:"pagination"`
}

// CustomerRepository defines the interface for customer data access
type CustomerRepository interface {
	// Create creates a new customer
	Create(ctx context.Context, customer *Customer) error

	// GetByID retrieves a customer by ID
	GetByID(ctx context.Context, id string) (*Customer, error)

	// GetByEmail retrieves a customer by email
	GetByEmail(ctx context.Context, email string) (*Customer, error)

	// GetByPhone retrieves a customer by phone number
	GetByPhone(ctx context.Context, phone string) (*Customer, error)

	// Update updates an existing customer
	Update(ctx context.Context, customer *Customer) error

	// Delete deletes a customer by ID
	Delete(ctx context.Context, id string) error

	// List retrieves customers with filters
	List(ctx context.Context, req ListCustomersRequest) (*ListCustomersResponse, error)

	// Search searches customers by query
	Search(ctx context.Context, query string, pagination PaginationRequest) (*ListCustomersResponse, error)

	// IncrementOrderCount atomically increments the customer's OrderCount by 1
	// and returns the new count. Uses DynamoDB ADD with ReturnValues=UPDATED_NEW,
	// which initializes the attribute to 0 if absent (so first-ever increment returns 1).
	IncrementOrderCount(ctx context.Context, customerID string) (int64, error)
}

// ListCustomersRequest contains parameters for listing customers
type ListCustomersRequest struct {
	Status     CustomerStatus    `json:"status,omitempty"`
	Search     string            `json:"search,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Pagination PaginationRequest `json:"pagination"`
}

// ListCustomersResponse contains the list of customers
type ListCustomersResponse struct {
	Customers  []*Customer        `json:"customers"`
	Pagination PaginationResponse `json:"pagination"`
}

// CreateCustomerRequest represents a request to create a customer
type CreateCustomerRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Phone     string   `json:"phone,omitempty"`
	FirstName string   `json:"first_name" validate:"required"`
	LastName  string   `json:"last_name" validate:"required"`
	Tags      []string `json:"tags,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	Address   *Address `json:"address,omitempty"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	FirstName string         `json:"first_name,omitempty"`
	LastName  string         `json:"last_name,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Notes     string         `json:"notes,omitempty"`
	Status    CustomerStatus `json:"status,omitempty"`
}

// OrderService defines the interface for order operations
type OrderService interface {
	// Create creates a new order
	Create(ctx context.Context, req CreateOrderRequest, createdBy string) (*Order, error)

	// GetByID retrieves an order by ID
	GetByID(ctx context.Context, id string) (*OrderWithDetails, error)

	// List retrieves orders with filters
	List(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)

	// UpdateStatus updates order status
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedBy string) error

	// AddNote adds a note to an order
	AddNote(ctx context.Context, id string, note string, isInternal bool, createdBy string) error

	// UpdateTracking updates tracking information
	UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, updatedBy string) error

	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, id string, reason string, updatedBy string) error

	// RefundOrder initiates a refund for an order
	RefundOrder(ctx context.Context, id string, amount int64, reason string, updatedBy string) error
}

// CreateOrderRequest contains data for creating an order
type CreateOrderRequest struct {
	CustomerID      string           `json:"customer_id" validate:"required"`
	Items           []OrderItemInput `json:"items" validate:"required,min=1"`
	ShippingAddress Address          `json:"shipping_address" validate:"required"`
	BillingAddress  *Address         `json:"billing_address,omitempty"`
	Notes           string           `json:"notes,omitempty"`
	CouponCode      *string          `json:"coupon_code,omitempty"`
}

// OrderItemInput represents an item in an order creation request
type OrderItemInput struct {
	ProductID        string                 `json:"product_id" validate:"required"`
	Quantity         int                    `json:"quantity" validate:"required,gt=0"`
	CustomDimensions *Dimensions            `json:"custom_dimensions,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	QuoteID          *string                `json:"quote_id,omitempty"`
}

// OrderWithDetails contains an order with related details
type OrderWithDetails struct {
	*Order
	Customer    *Customer          `json:"customer,omitempty"`
	ItemDetails []OrderItemDetails `json:"item_details,omitempty"`
}

// OrderItemDetails contains order item with product details
type OrderItemDetails struct {
	OrderItem
	ProductName   string         `json:"product_name"`
	ProductSKU    string         `json:"product_sku"`
	ProductImages []ProductImage `json:"product_images,omitempty"`
}
