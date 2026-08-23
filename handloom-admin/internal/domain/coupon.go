package domain

import (
	"context"
	"time"
)

// ==================== COUPON ENTITY ====================

// CouponType defines the type of coupon
type CouponType string

const (
	CouponTypePercentage CouponType = "PERCENTAGE"
	CouponTypeFixed      CouponType = "FIXED"
)

// CouponStatus defines the status of a coupon
type CouponStatus string

const (
	CouponStatusActive   CouponStatus = "ACTIVE"
	CouponStatusInactive CouponStatus = "INACTIVE"
	CouponStatusExpired  CouponStatus = "EXPIRED"
)

// Coupon represents a discount coupon
type Coupon struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Code        string `json:"code" dynamodbav:"code"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description,omitempty" dynamodbav:"description,omitempty"`

	Type  CouponType `json:"type" dynamodbav:"type"`
	Value int64      `json:"value" dynamodbav:"value"` // percentage * 100 or fixed amount in paise

	// Constraints
	MinOrderValue int64 `json:"min_order_value" dynamodbav:"min_order_value"`
	MaxDiscount   int64 `json:"max_discount,omitempty" dynamodbav:"max_discount,omitempty"` // for percentage type

	// Usage limits
	UsageLimit   int `json:"usage_limit" dynamodbav:"usage_limit"`       // 0 = unlimited
	UsagePerUser int `json:"usage_per_user" dynamodbav:"usage_per_user"` // 0 = unlimited
	UsageCount   int `json:"usage_count" dynamodbav:"usage_count"`

	// Applicability
	ApplicableCategories []string `json:"applicable_categories,omitempty" dynamodbav:"applicable_categories,omitempty"`
	ApplicableProducts   []string `json:"applicable_products,omitempty" dynamodbav:"applicable_products,omitempty"`
	ExcludedCategories   []string `json:"excluded_categories,omitempty" dynamodbav:"excluded_categories,omitempty"`
	ExcludedProducts     []string `json:"excluded_products,omitempty" dynamodbav:"excluded_products,omitempty"`

	// Validity
	ValidFrom  time.Time    `json:"valid_from" dynamodbav:"valid_from"`
	ValidUntil time.Time    `json:"valid_until" dynamodbav:"valid_until"`
	Status     CouponStatus `json:"status" dynamodbav:"status"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Coupon
func (c *Coupon) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for Coupon
func (c *Coupon) SetKeys() {
	c.PK = "COUPON#" + c.ID
	c.SK = SKMetadata
	c.GSI1PK = "COUPON_CODE"
	c.GSI1SK = c.Code
	c.EntityType = "COUPON"
}

// CouponUsage tracks coupon usage
type CouponUsage struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	CouponID   string `json:"coupon_id" dynamodbav:"coupon_id"`
	CouponCode string `json:"coupon_code" dynamodbav:"coupon_code"`
	OrderID    string `json:"order_id" dynamodbav:"order_id"`
	CustomerID string `json:"customer_id" dynamodbav:"customer_id"`
	Discount   int64  `json:"discount" dynamodbav:"discount"` // actual discount applied

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

// TableName returns the DynamoDB table name for CouponUsage
func (u *CouponUsage) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for CouponUsage
func (u *CouponUsage) SetKeys() {
	u.PK = "COUPON#" + u.CouponID
	u.SK = "USAGE#" + u.CreatedAt.Format("2006-01-02T15:04:05Z") + "#" + u.OrderID
	u.EntityType = "COUPON_USAGE"
}

// ==================== COUPON REPOSITORY ====================

// CouponRepository defines the interface for coupon data access
type CouponRepository interface {
	// Create creates a new coupon
	Create(ctx context.Context, coupon *Coupon) error

	// GetByID retrieves a coupon by ID
	GetByID(ctx context.Context, id string) (*Coupon, error)

	// GetByCode retrieves a coupon by code
	GetByCode(ctx context.Context, code string) (*Coupon, error)

	// Update updates an existing coupon
	Update(ctx context.Context, coupon *Coupon) error

	// Delete deletes a coupon by ID
	Delete(ctx context.Context, id string) error

	// List retrieves coupons with filters
	List(ctx context.Context, req ListCouponsRequest) (*ListCouponsResponse, error)

	// RecordUsage records coupon usage
	RecordUsage(ctx context.Context, usage *CouponUsage) error

	// GetUserUsageCount gets the number of times a user has used a coupon
	GetUserUsageCount(ctx context.Context, couponID, customerID string) (int, error)
}

// ListCouponsRequest contains parameters for listing coupons
type ListCouponsRequest struct {
	PaginationRequest
	Status   *CouponStatus `json:"status,omitempty"`
	Type     *CouponType   `json:"type,omitempty"`
	Search   string        `json:"search,omitempty"`
	IsActive *bool         `json:"is_active,omitempty"`
}

// ListCouponsResponse contains the list of coupons
type ListCouponsResponse struct {
	Coupons    []*Coupon          `json:"coupons"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== COUPON SERVICE ====================

// CouponService defines the interface for coupon operations
type CouponService interface {
	// Create creates a new coupon
	Create(ctx context.Context, req CreateCouponRequest, createdBy string) (*Coupon, error)

	// GetByID retrieves a coupon by ID
	GetByID(ctx context.Context, id string) (*Coupon, error)

	// GetByCode retrieves a coupon by code
	GetByCode(ctx context.Context, code string) (*Coupon, error)

	// Update updates an existing coupon
	Update(ctx context.Context, id string, req UpdateCouponRequest, updatedBy string) (*Coupon, error)

	// Delete deletes a coupon by ID
	Delete(ctx context.Context, id string) error

	// List retrieves coupons with filters
	List(ctx context.Context, req ListCouponsRequest) (*ListCouponsResponse, error)

	// Validate validates a coupon for an order
	Validate(ctx context.Context, code string, orderTotal int64, customerID string, lines []CouponLine) (*CouponValidationResult, error)

	// Apply applies a coupon to an order
	Apply(ctx context.Context, couponID string, orderID string, customerID string, discount int64) error
}

// CreateCouponRequest contains data for creating a coupon
type CreateCouponRequest struct {
	Code                 string     `json:"code" validate:"required"`
	Name                 string     `json:"name" validate:"required"`
	Description          string     `json:"description,omitempty"`
	Type                 CouponType `json:"type" validate:"required"`
	Value                int64      `json:"value" validate:"required,gt=0"`
	MinOrderValue        int64      `json:"min_order_value"`
	MaxDiscount          int64      `json:"max_discount,omitempty"`
	UsageLimit           int        `json:"usage_limit"`
	UsagePerUser         int        `json:"usage_per_user"`
	ApplicableCategories []string   `json:"applicable_categories,omitempty"`
	ApplicableProducts   []string   `json:"applicable_products,omitempty"`
	ExcludedCategories   []string   `json:"excluded_categories,omitempty"`
	ExcludedProducts     []string   `json:"excluded_products,omitempty"`
	ValidFrom            time.Time  `json:"valid_from" validate:"required"`
	ValidUntil           time.Time  `json:"valid_until" validate:"required"`
}

// UpdateCouponRequest contains data for updating a coupon
type UpdateCouponRequest struct {
	Name                 *string       `json:"name,omitempty"`
	Description          *string       `json:"description,omitempty"`
	MinOrderValue        *int64        `json:"min_order_value,omitempty"`
	MaxDiscount          *int64        `json:"max_discount,omitempty"`
	UsageLimit           *int          `json:"usage_limit,omitempty"`
	UsagePerUser         *int          `json:"usage_per_user,omitempty"`
	ApplicableCategories []string      `json:"applicable_categories,omitempty"`
	ApplicableProducts   []string      `json:"applicable_products,omitempty"`
	ExcludedCategories   []string      `json:"excluded_categories,omitempty"`
	ExcludedProducts     []string      `json:"excluded_products,omitempty"`
	ValidFrom            *time.Time    `json:"valid_from,omitempty"`
	ValidUntil           *time.Time    `json:"valid_until,omitempty"`
	Status               *CouponStatus `json:"status,omitempty"`
}

// CouponLine is what a coupon needs to know about one order line to scope itself.
// Carries the category so validation needs no product read.
type CouponLine struct {
	ProductID  string `json:"product_id"`
	CategoryID string `json:"category_id"`
}

// CouponValidationResult contains the result of coupon validation
type CouponValidationResult struct {
	Valid          bool   `json:"valid"`
	CouponID       string `json:"coupon_id,omitempty"`
	Code           string `json:"code"`
	DiscountAmount int64  `json:"discount_amount,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}
