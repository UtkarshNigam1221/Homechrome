package domain

import "time"

// ==================== CART ENTITY ====================

// Cart represents a shopping cart header
type Cart struct {
	ID         string    `json:"id" dynamodbav:"id"`
	PK         string    `json:"-" dynamodbav:"PK"`
	SK         string    `json:"-" dynamodbav:"SK"`
	EntityType string    `json:"-" dynamodbav:"entity_type"`
	CustomerID string    `json:"customer_id,omitempty" dynamodbav:"customer_id,omitempty"`
	SessionID  string    `json:"session_id" dynamodbav:"session_id"`
	ItemCount  int       `json:"item_count" dynamodbav:"item_count"`
	Subtotal   int64     `json:"subtotal" dynamodbav:"subtotal"` // in paise
	Currency   string    `json:"currency" dynamodbav:"currency"`
	UpdatedAt  time.Time `json:"updated_at" dynamodbav:"updated_at"`
	TTL        int64     `json:"-" dynamodbav:"ttl"`
}

// SetKeys sets the DynamoDB keys for Cart
func (c *Cart) SetKeys() {
	if c.CustomerID != "" {
		c.PK = "CART#" + c.CustomerID
	} else {
		c.PK = "CART#" + c.SessionID
	}
	c.SK = SKMetadata
	c.EntityType = "CART"
}

// CartItem represents an item in a shopping cart
type CartItem struct {
	PK           string            `json:"-" dynamodbav:"PK"`
	SK           string            `json:"-" dynamodbav:"SK"`
	EntityType   string            `json:"-" dynamodbav:"entity_type"`
	ProductID    string            `json:"product_id" dynamodbav:"product_id"`
	ProductName  string            `json:"product_name" dynamodbav:"product_name"`
	ProductSKU   string            `json:"product_sku" dynamodbav:"product_sku"`
	ProductImage string            `json:"product_image" dynamodbav:"product_image"`
	CategoryID   string            `json:"category_id" dynamodbav:"category_id"`
	CategoryName string            `json:"category_name" dynamodbav:"category_name"`
	Quantity     int               `json:"quantity" dynamodbav:"quantity"`
	UnitPrice    int64             `json:"unit_price" dynamodbav:"unit_price"`   // in paise
	TotalPrice   int64             `json:"total_price" dynamodbav:"total_price"` // in paise
	IsCustomSize bool              `json:"is_custom_size" dynamodbav:"is_custom_size"`
	Dimensions   *Dimensions       `json:"dimensions,omitempty" dynamodbav:"dimensions,omitempty"`
	QuoteID      *string           `json:"quote_id,omitempty" dynamodbav:"quote_id,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty" dynamodbav:"attributes,omitempty"`
	AddedAt      time.Time         `json:"added_at" dynamodbav:"added_at"`
	TTL          int64             `json:"-" dynamodbav:"ttl"`
}

// SetKeys sets the DynamoDB keys for CartItem
func (ci *CartItem) SetKeys(cartPK string) {
	ci.PK = cartPK
	ci.SK = "ITEM#" + ci.ProductID
	ci.EntityType = "CART_ITEM"
}

// ==================== CART REQUEST/RESPONSE TYPES ====================

// AddCartItemRequest contains data for adding an item to the cart
type AddCartItemRequest struct {
	ProductID  string      `json:"product_id" validate:"required"`
	Quantity   int         `json:"quantity" validate:"required,gt=0"`
	Dimensions *Dimensions `json:"dimensions,omitempty"`
	QuoteID    *string     `json:"quote_id,omitempty"`
}

// UpdateCartItemRequest contains data for updating a cart item quantity
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gte=0"`
}

// MergeCartRequest contains data for merging a guest cart into a customer cart
type MergeCartRequest struct {
	Items []AddCartItemRequest `json:"items" validate:"required"`
}

// CartWithItems represents a cart with all its items
type CartWithItems struct {
	Cart  *Cart      `json:"cart"`
	Items []CartItem `json:"items"`
}
