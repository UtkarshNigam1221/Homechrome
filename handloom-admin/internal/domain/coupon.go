package domain

import (
	"context"
	"strings"
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

// CouponAudience is who a coupon is for. Exactly one applies. A single field rather
// than independent flags, so "first order only AND returning" — which is unsatisfiable —
// cannot be expressed and does not need guarding at every read site.
type CouponAudience string

const (
	AudienceAll              CouponAudience = "ALL"
	AudienceFirstOrder       CouponAudience = "FIRST_ORDER"
	AudienceReturning        CouponAudience = "RETURNING"
	AudienceSpecificCustomer CouponAudience = "SPECIFIC_CUSTOMER"
)

// couponSortTimeLayout is fixed-width on purpose — see SetKeys.
const couponSortTimeLayout = "2006-01-02T15:04:05.000000000Z"

// PublicCouponListTTL is how long a public coupon payload may be cached. ListPublic
// drops coupons expiring inside it, so a cached payload cannot advertise a dead code.
const PublicCouponListTTL = time.Hour

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

	// Audience. Exactly one case applies; CustomerID is set only for SPECIFIC_CUSTOMER.
	Audience   CouponAudience `json:"audience" dynamodbav:"audience"`
	CustomerID string         `json:"customer_id,omitempty" dynamodbav:"customer_id,omitempty"`
	BatchID    string         `json:"batch_id,omitempty" dynamodbav:"batch_id,omitempty"`

	// CombinesWithOffers lets this code apply on a cart already carrying an automatic
	// offer. The zero value refuses, deliberately: buy-2-get-1 is a third off before any
	// code, so stacking should be chosen rather than defaulted into.
	CombinesWithOffers bool `json:"combines_with_offers" dynamodbav:"combines_with_offers"`

	// Validity
	ValidFrom  time.Time    `json:"valid_from" dynamodbav:"valid_from"`
	ValidUntil *time.Time   `json:"valid_until,omitempty" dynamodbav:"valid_until,omitempty"` // nil = open-ended
	Status     CouponStatus `json:"status" dynamodbav:"status"`

	// SearchKey is lower(code + " " + name), written by SetKeys so the list's search
	// filter can run in DynamoDB. Never returned — it is an index, not a field.
	SearchKey string `json:"-" dynamodbav:"search_key"`

	BaseEntity
}

// SetKeys derives every key, including which GSI1 partition this coupon belongs in.
// Deriving it here rather than at call sites means no writer has to remember the rule.
func (c *Coupon) SetKeys() {
	c.PK = "COUPON#" + c.ID
	c.SK = SKMetadata
	if c.Audience == AudienceSpecificCustomer {
		c.GSI1PK = "CUSTOMER_COUPON#" + c.CustomerID
	} else {
		c.GSI1PK = "COUPON#ALL"
	}
	// created_at, so a descending query on GSI1 returns the admin list newest-first
	// and pages in DynamoDB. This key used to encode valid_until for a `GSI1SK >= now`
	// range the Phase 5 wallet was to use — nothing queries it today, and the list
	// that does exist was reading the whole partition on every page. When the wallet
	// arrives it needs its own index; that is a cost paid when it buys something.
	//
	// Fixed-width nanoseconds, not RFC3339Nano: that layout trims trailing zeros, so a
	// whole second renders "12:00:00Z" while a microsecond later renders
	// "12:00:00.000001Z" — and "." sorts before "Z", putting the later coupon first.
	// A constant nine digits keeps lexicographic order equal to chronological order,
	// which is what the descending query and the cursor both depend on.
	c.GSI1SK = c.CreatedAt.UTC().Format(couponSortTimeLayout) + "#" + c.ID
	// A DynamoDB contains() is case-sensitive, so search filters on a lowercased copy.
	c.SearchKey = strings.ToLower(c.Code + " " + c.Name)
	c.EntityType = "COUPON"
}

// TableName returns the DynamoDB table name for Coupon
func (c *Coupon) TableName() string {
	return TableCoupons
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
	return TableCoupons
}

// SetKeys sets the DynamoDB keys for CouponUsage
func (u *CouponUsage) SetKeys() {
	u.PK = "COUPON#" + u.CouponID
	u.SK = "USAGE#" + u.CreatedAt.Format("2006-01-02T15:04:05Z") + "#" + u.OrderID
	u.EntityType = "COUPON_USAGE"
}

// CouponUseCounter is one customer's redemption count for one coupon. Keyed by customer
// so a single query returns every count that customer has, which is what makes the
// wallet affordable — the alternative reads every redemption row of each candidate.
type CouponUseCounter struct {
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	CustomerID string `json:"customer_id" dynamodbav:"customer_id"`
	CouponID   string `json:"coupon_id" dynamodbav:"coupon_id"`
	Count      int    `json:"count" dynamodbav:"count"`
}

// SetKeys sets the DynamoDB keys for CouponUseCounter
func (u *CouponUseCounter) SetKeys(customerID, couponID string) {
	u.PK = "CUSTOMER#" + customerID
	u.SK = "USE#" + couponID
	u.EntityType = "COUPON_USE_COUNTER"
}

// TableName returns the DynamoDB table name for CouponUseCounter
func (u *CouponUseCounter) TableName() string {
	return TableCoupons
}

// CouponContext is everything a coupon needs to know about a cart. Deliberately not a
// line list: coupons carry no item scoping, so this is a total, an identity and one flag.
type CouponContext struct {
	CartTotal         int64 // after any automatic offer
	CustomerID        string
	HasAutomaticOffer bool // a price campaign or buy-N-get-M applied to this cart
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

	// ListPublic returns coupons safe to advertise: ACTIVE, audience ALL, and valid
	// past the cache window. Not paginated — a banner's worth is small by design.
	ListPublic(ctx context.Context) ([]*Coupon, error)

	// RecordUsage records coupon usage
	RecordUsage(ctx context.Context, usage *CouponUsage) error

	// IncrementUsage atomically claims one redemption. Returns false when exhausted.
	IncrementUsage(ctx context.Context, couponID string) (bool, error)

	// GetCustomerUsage returns how many times this customer has redeemed this coupon.
	GetCustomerUsage(ctx context.Context, customerID, couponID string) (int, error)

	// GetCustomerUsageAll returns every per-coupon count this customer holds, keyed by
	// coupon id. One query, so pricing M candidates costs one read rather than M.
	GetCustomerUsageAll(ctx context.Context, customerID string) (map[string]int, error)

	// IncrementCustomerUsage claims one of this customer's allowance for this coupon,
	// under the same kind of condition IncrementUsage uses for the global limit.
	// Returns false when the allowance is already spent. A limit of 0 is unlimited.
	IncrementCustomerUsage(ctx context.Context, customerID, couponID string, limit int) (claimed bool, err error)
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

// CouponOffer is one coupon priced against a specific cart. Reason carries the
// customer-facing message when Eligible is false, so a picker can say why.
type CouponOffer struct {
	Coupon         *Coupon `json:"-"`
	Eligible       bool    `json:"eligible"`
	DiscountAmount int64   `json:"discount_amount"`
	Reason         string  `json:"reason,omitempty"`
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

	// Validate checks a coupon against a cart and returns the discount it would give.
	Validate(ctx context.Context, code string, cc CouponContext) (*CouponValidationResult, error)

	// ListPublic returns the coupons safe to advertise, with no cart context.
	ListPublic(ctx context.Context) ([]*Coupon, error)

	// ListForCart prices every public coupon against one cart. Eligible offers come
	// first, best saving down; ineligible ones follow, each carrying its reason.
	ListForCart(ctx context.Context, cc CouponContext) ([]*CouponOffer, error)

	// Redeem records one redemption of a paid order. Reports claimed=false when the
	// coupon's usage limit is already exhausted, which is an outcome, not an error.
	Redeem(ctx context.Context, couponID, orderID, customerID string, discount int64) (claimed bool, err error)
}

// CreateCouponRequest contains data for creating a coupon
type CreateCouponRequest struct {
	Code               string         `json:"code" validate:"required"`
	Name               string         `json:"name" validate:"required"`
	Description        string         `json:"description,omitempty"`
	Type               CouponType     `json:"type" validate:"required"`
	Value              int64          `json:"value" validate:"required,gt=0,coupon_value"`
	MinOrderValue      int64          `json:"min_order_value"`
	MaxDiscount        int64          `json:"max_discount,omitempty"`
	UsageLimit         int            `json:"usage_limit"`
	UsagePerUser       int            `json:"usage_per_user"`
	Audience           CouponAudience `json:"audience" validate:"required,oneof=ALL FIRST_ORDER RETURNING SPECIFIC_CUSTOMER"`
	CustomerID         string         `json:"customer_id,omitempty" validate:"required_if=Audience SPECIFIC_CUSTOMER"`
	CombinesWithOffers bool           `json:"combines_with_offers"`
	ValidFrom          time.Time      `json:"valid_from" validate:"required"`
	ValidUntil         *time.Time     `json:"valid_until,omitempty"` // nil = open-ended
}

// UpdateCouponRequest contains data for updating a coupon
type UpdateCouponRequest struct {
	Name               *string       `json:"name,omitempty"`
	Description        *string       `json:"description,omitempty"`
	MinOrderValue      *int64        `json:"min_order_value,omitempty"`
	MaxDiscount        *int64        `json:"max_discount,omitempty"`
	UsageLimit         *int          `json:"usage_limit,omitempty"`
	UsagePerUser       *int          `json:"usage_per_user,omitempty"`
	CombinesWithOffers *bool         `json:"combines_with_offers,omitempty"`
	ValidFrom          *time.Time    `json:"valid_from,omitempty"`
	ValidUntil         *time.Time    `json:"valid_until,omitempty"`
	ClearValidUntil    bool          `json:"clear_valid_until,omitempty"` // explicit: make it open-ended
	Status             *CouponStatus `json:"status,omitempty"`
}

// CouponValidationResult contains the result of coupon validation
type CouponValidationResult struct {
	Valid          bool   `json:"valid"`
	CouponID       string `json:"coupon_id,omitempty"`
	Code           string `json:"code"`
	DiscountAmount int64  `json:"discount_amount,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`

	// Notice explains a discount that was applied but reduced — a code worth more than
	// the order still has to leave something payable. Set only on a valid result;
	// ErrorMessage carries the reason a coupon was refused outright.
	Notice string `json:"notice,omitempty"`

	// Outcome is the metric label for a rejection ("invalid", "expired",
	// "limit_reached"). Internal — never serialized to a customer.
	Outcome string `json:"-"`
}
