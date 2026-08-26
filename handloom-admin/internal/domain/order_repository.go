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

	// ApplyRefundSettlement writes only the lines' refunded quantities and the payment
	// status, so it cannot revert a status or tracking change made since the read.
	ApplyRefundSettlement(ctx context.Context, id string, items []OrderItem, paymentStatus PaymentStatus) error

	// List retrieves orders with filters
	List(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)

	// GetByCustomer retrieves orders by customer ID
	GetByCustomer(ctx context.Context, customerID string, pagination PaginationRequest) (*ListOrdersResponse, error)

	// UpdateStatus updates order status
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedBy string) error

	// AddNote adds a note to an order
	AddNote(ctx context.Context, id string, note OrderNote) error

	// UpdateTracking updates tracking information
	UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, trackingURL string) error
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

	// RecordPurchase atomically increments the customer's OrderCount by 1 and
	// adds amountPaise to TotalSpent, returning the new count. Uses DynamoDB ADD
	// with ReturnValues=UPDATED_NEW, which initializes the attributes to 0 if
	// absent (so the first-ever increment returns 1). Both counters move in one
	// UpdateItem so they cannot drift apart under concurrent orders.
	RecordPurchase(ctx context.Context, customerID string, amountPaise int64) (int64, error)
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
	Status    CustomerStatus `json:"status,omitempty" validate:"omitempty,oneof=ACTIVE INACTIVE BLOCKED"`
}

// OrderService defines the interface for order operations
type OrderService interface {
	// GetByID retrieves an order by ID
	GetByID(ctx context.Context, id string) (*OrderWithDetails, error)

	// List retrieves orders with filters
	List(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)

	// UpdateStatus updates order status
	UpdateStatus(ctx context.Context, id string, status OrderStatus, updatedBy string) error

	// AddNote adds a note to an order
	AddNote(ctx context.Context, id string, note string, isInternal bool, createdBy string) error

	// UpdateTracking updates tracking information
	UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, trackingURL string, updatedBy string) error

	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, id string, reason string, updatedBy string) error
}

// OrderWithDetails contains an order with related details
type OrderWithDetails struct {
	*Order
	Customer    *Customer          `json:"customer,omitempty"`
	ItemDetails []OrderItemDetails `json:"item_details,omitempty"`

	// RefundedAmount is what the payment says has gone back, and the authority on what
	// is still refundable: a client summing its own rows drifts on a half settlement.
	RefundedAmount int64 `json:"refunded_amount"`
}

// OrderItemDetails contains order item with product details
type OrderItemDetails struct {
	OrderItem
	ProductName   string         `json:"product_name"`
	ProductSKU    string         `json:"product_sku"`
	ProductImages []ProductImage `json:"product_images,omitempty"`
}
