package domain

import (
	"time"
)

// ==================== ORDER ENTITY ====================

// Order represents a customer order
type Order struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK     string `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK     string `json:"-" dynamodbav:"GSI2SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	OrderNumber   string `json:"order_number" dynamodbav:"order_number"`
	CustomerID    string `json:"customer_id" dynamodbav:"customer_id"`
	CustomerName  string `json:"customer_name" dynamodbav:"customer_name"`
	CustomerEmail string `json:"customer_email" dynamodbav:"customer_email"`
	CustomerPhone string `json:"customer_phone" dynamodbav:"customer_phone"`

	// Items
	Items     []OrderItem `json:"items" dynamodbav:"items"`
	ItemCount int         `json:"item_count" dynamodbav:"item_count"`

	// Pricing (in paise)
	Subtotal       int64  `json:"subtotal" dynamodbav:"subtotal"`
	DiscountAmount int64  `json:"discount_amount" dynamodbav:"discount_amount"`
	TaxAmount      int64  `json:"tax_amount" dynamodbav:"tax_amount"`
	ShippingAmount int64  `json:"shipping_amount" dynamodbav:"shipping_amount"`
	TotalAmount    int64  `json:"total_amount" dynamodbav:"total_amount"`
	Currency       string `json:"currency" dynamodbav:"currency"`

	// Coupon
	CouponID   *string `json:"coupon_id,omitempty" dynamodbav:"coupon_id,omitempty"`
	CouponCode *string `json:"coupon_code,omitempty" dynamodbav:"coupon_code,omitempty"`

	// Status
	Status        OrderStatus   `json:"status" dynamodbav:"status"`
	PaymentStatus PaymentStatus `json:"payment_status" dynamodbav:"payment_status"`
	PaymentMethod string        `json:"payment_method,omitempty" dynamodbav:"payment_method,omitempty"`
	PaymentID     string        `json:"payment_id,omitempty" dynamodbav:"payment_id,omitempty"`

	// Shipping
	ShippingAddress *Address `json:"shipping_address" dynamodbav:"shipping_address"`
	BillingAddress  *Address `json:"billing_address,omitempty" dynamodbav:"billing_address,omitempty"`
	TrackingNumber  string   `json:"tracking_number,omitempty" dynamodbav:"tracking_number,omitempty"`
	TrackingURL     string   `json:"tracking_url,omitempty" dynamodbav:"tracking_url,omitempty"`
	ShippingCarrier string   `json:"shipping_carrier,omitempty" dynamodbav:"shipping_carrier,omitempty"`

	// Geo — resolved from the checkout request context (CloudFront headers)
	// and persisted here so downstream payment/order-placed metrics can read
	// them without relying on the gateway webhook's request context. No `state`
	// field: the geo pipeline is country/city only (CloudFront does not resolve
	// state, and migration 010 purges any state-labeled metrics).
	Country    string `json:"country,omitempty"     dynamodbav:"country,omitempty"`
	City       string `json:"city,omitempty"        dynamodbav:"city,omitempty"`
	DeviceType string `json:"device_type,omitempty" dynamodbav:"device_type,omitempty"`
	UTMSource  string `json:"utm_source,omitempty"  dynamodbav:"utm_source,omitempty"`

	// Notes
	CustomerNote  string      `json:"customer_note,omitempty" dynamodbav:"customer_note,omitempty"`
	InternalNotes []OrderNote `json:"internal_notes,omitempty" dynamodbav:"internal_notes,omitempty"`

	// Timestamps
	ShippedAt   *time.Time `json:"shipped_at,omitempty" dynamodbav:"shipped_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" dynamodbav:"delivered_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty" dynamodbav:"cancelled_at,omitempty"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Order
func (o *Order) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for Order
func (o *Order) SetKeys() {
	o.PK = "ORDER#" + o.ID
	o.SK = SKMetadata
	o.GSI1PK = "CUSTOMER#" + o.CustomerID
	o.GSI1SK = o.CreatedAt.Format("2006-01-02T15:04:05Z")
	o.GSI2PK = "ORDER#ALL"
	o.GSI2SK = o.CreatedAt.Format("2006-01-02T15:04:05Z")
	o.EntityType = "ORDER"
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID           string `json:"id" dynamodbav:"id"`
	ProductID    string `json:"product_id" dynamodbav:"product_id"`
	ProductName  string `json:"product_name" dynamodbav:"product_name"`
	ProductSKU   string `json:"product_sku" dynamodbav:"product_sku"`
	ProductImage string `json:"product_image,omitempty" dynamodbav:"product_image,omitempty"`
	CategoryID   string `json:"category_id" dynamodbav:"category_id"`
	CategoryName string `json:"category_name" dynamodbav:"category_name"`

	// Custom dimensions (if applicable)
	IsCustomSize bool                   `json:"is_custom_size" dynamodbav:"is_custom_size"`
	Dimensions   *Dimensions            `json:"dimensions,omitempty" dynamodbav:"dimensions,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty" dynamodbav:"attributes,omitempty"`
	QuoteID      *string                `json:"quote_id,omitempty" dynamodbav:"quote_id,omitempty"`

	// Pricing
	UnitPrice  int64 `json:"unit_price" dynamodbav:"unit_price"`
	Quantity   int   `json:"quantity" dynamodbav:"quantity"`
	TotalPrice int64 `json:"total_price" dynamodbav:"total_price"`
}

// OrderNote represents an internal note on an order
type OrderNote struct {
	ID         string    `json:"id" dynamodbav:"id"`
	Note       string    `json:"note" dynamodbav:"note"`
	IsInternal bool      `json:"is_internal" dynamodbav:"is_internal"`
	CreatedBy  string    `json:"created_by" dynamodbav:"created_by"`
	CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
}

// Address represents a shipping or billing address
type Address struct {
	ID           string `json:"id,omitempty" dynamodbav:"id,omitempty"`
	FirstName    string `json:"first_name" dynamodbav:"first_name"`
	LastName     string `json:"last_name" dynamodbav:"last_name"`
	Phone        string `json:"phone" dynamodbav:"phone"`
	AddressLine1 string `json:"address_line1" dynamodbav:"address_line1"`
	AddressLine2 string `json:"address_line2,omitempty" dynamodbav:"address_line2,omitempty"`
	City         string `json:"city" dynamodbav:"city"`
	State        string `json:"state" dynamodbav:"state"`
	PostalCode   string `json:"postal_code" dynamodbav:"postal_code"`
	Country      string `json:"country" dynamodbav:"country"`
	IsDefault    bool   `json:"is_default,omitempty" dynamodbav:"is_default,omitempty"`
}

// OrderStatusHistory tracks status changes
type OrderStatusHistory struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	OrderID    string      `json:"order_id" dynamodbav:"order_id"`
	FromStatus OrderStatus `json:"from_status" dynamodbav:"from_status"`
	ToStatus   OrderStatus `json:"to_status" dynamodbav:"to_status"`
	Reason     string      `json:"reason,omitempty" dynamodbav:"reason,omitempty"`
	CreatedBy  string      `json:"created_by" dynamodbav:"created_by"`
	CreatedAt  time.Time   `json:"created_at" dynamodbav:"created_at"`
}

// TableName returns the DynamoDB table name for OrderStatusHistory
func (h *OrderStatusHistory) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for OrderStatusHistory
func (h *OrderStatusHistory) SetKeys() {
	h.PK = "ORDER#" + h.OrderID
	h.SK = "STATUS#" + h.CreatedAt.Format("2006-01-02T15:04:05.000Z")
	h.EntityType = "ORDER_STATUS_HISTORY"
}

// CustomerStatus defines the status of a customer
type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "ACTIVE"
	CustomerStatusInactive CustomerStatus = "INACTIVE"
	CustomerStatusBlocked  CustomerStatus = "BLOCKED"
)

// Customer represents a customer
type Customer struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK     string `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK     string `json:"-" dynamodbav:"GSI2SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Email         string         `json:"email" dynamodbav:"email"`
	FirstName     string         `json:"first_name" dynamodbav:"first_name"`
	LastName      string         `json:"last_name" dynamodbav:"last_name"`
	Phone         string         `json:"phone" dynamodbav:"phone"`
	PhoneVerified bool           `json:"phone_verified" dynamodbav:"phone_verified"`
	Status        CustomerStatus `json:"status" dynamodbav:"status"`

	// Both maintained by CustomerRepository.RecordPurchase. TotalSpent is gross
	// order value in paise — the stubbed refund path does not adjust it.
	TotalSpent int64 `json:"total_spent" dynamodbav:"total_spent"`
	OrderCount int   `json:"order_count" dynamodbav:"order_count"`

	Addresses []Address `json:"addresses,omitempty" dynamodbav:"addresses,omitempty"`
	Tags      []string  `json:"tags,omitempty" dynamodbav:"tags,omitempty"`
	Notes     string    `json:"notes,omitempty" dynamodbav:"notes,omitempty"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Customer
func (c *Customer) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for Customer
func (c *Customer) SetKeys() {
	c.PK = "CUSTOMER#" + c.ID
	c.SK = SKMetadata
	c.GSI1PK = "CUSTOMER_EMAIL"
	if c.Email != "" {
		c.GSI1SK = c.Email
	} else {
		c.GSI1SK = "NONE#" + c.ID
	}
	c.GSI2PK = "CUSTOMER#ALL"
	c.GSI2SK = c.CreatedAt.Format("2006-01-02T15:04:05Z")
	c.EntityType = "CUSTOMER"
}

// ==================== ORDER/CUSTOMER INDEX ENTITIES ====================

// OrderNumberIndex is a lookup item for finding orders by order number
type OrderNumberIndex struct {
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`
	OrderID    string `json:"order_id" dynamodbav:"order_id"`
}

// SetKeys sets the DynamoDB keys for OrderNumberIndex
func (o *OrderNumberIndex) SetKeys(orderNumber string) {
	o.PK = "ORDER_NUMBER#" + orderNumber
	o.SK = SKMetadata
	o.EntityType = "ORDER_NUMBER_INDEX"
}

// CustomerPhoneIndex is a lookup item for finding customers by phone (uniqueness guard)
type CustomerPhoneIndex struct {
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`
	CustomerID string `json:"customer_id" dynamodbav:"customer_id"`
}

// SetKeys sets the DynamoDB keys for CustomerPhoneIndex
func (c *CustomerPhoneIndex) SetKeys(phone string) {
	c.PK = "CUSTOMER_PHONE#" + phone
	c.SK = SKMetadata
	c.EntityType = "CUSTOMER_PHONE_INDEX"
}
