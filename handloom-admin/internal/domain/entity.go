// Package domain contains all business entities and interfaces
package domain

import (
	"time"
)

// ==================== ENUMS & CONSTANTS ====================

// UserRole defines the role of a user
type UserRole string

const (
	UserRoleAdmin    UserRole = "ADMIN"
	UserRoleOperator UserRole = "OPERATOR"
)

// UserStatus defines the status of a user
type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusPending  UserStatus = "PENDING"
)

// CategoryStatus defines the status of a category
type CategoryStatus string

const (
	CategoryStatusActive   CategoryStatus = "ACTIVE"
	CategoryStatusInactive CategoryStatus = "INACTIVE"
)

// ProductStatus defines the status of a product
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
	ProductStatusDraft    ProductStatus = "DRAFT"
)

// AttributeType defines the type of a category attribute
type AttributeType string

const (
	AttributeTypeSelect      AttributeType = "SELECT"
	AttributeTypeMultiSelect AttributeType = "MULTI_SELECT"
	AttributeTypeText        AttributeType = "TEXT"
	AttributeTypeNumber      AttributeType = "NUMBER"
	AttributeTypeBoolean     AttributeType = "BOOLEAN"
)

// PricingRuleScope defines where a pricing rule applies
type PricingRuleScope string

const (
	PricingRuleScopeGlobal   PricingRuleScope = "GLOBAL"
	PricingRuleScopeCategory PricingRuleScope = "CATEGORY"
	PricingRuleScopeProduct  PricingRuleScope = "PRODUCT"
	PricingRuleScopeMaterial PricingRuleScope = "MATERIAL"
)

// PricingType defines how price is calculated
type PricingType string

const (
	PricingTypeAreaBased   PricingType = "AREA_BASED"
	PricingTypeLengthBased PricingType = "LENGTH_BASED"
	PricingTypeFixed       PricingType = "FIXED"
	PricingTypeTiered      PricingType = "TIERED"
	PricingTypeFormula     PricingType = "FORMULA"
)

// PricingUnit defines the unit for pricing
type PricingUnit string

const (
	PricingUnitSqInch  PricingUnit = "SQ_INCH"
	PricingUnitSqFoot  PricingUnit = "SQ_FOOT"
	PricingUnitSqCm    PricingUnit = "SQ_CM"
	PricingUnitSqMeter PricingUnit = "SQ_METER"
	PricingUnitInch    PricingUnit = "INCH"
	PricingUnitCm      PricingUnit = "CM"
	PricingUnitFoot    PricingUnit = "FOOT"
	PricingUnitMeter   PricingUnit = "METER"
)

// SurchargeType defines the type of surcharge
type SurchargeType string

const (
	SurchargeTypeFixed      SurchargeType = "FIXED"
	SurchargeTypePercentage SurchargeType = "PERCENTAGE"
)

// OrderStatus defines the status of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusConfirmed  OrderStatus = "CONFIRMED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusDelivered  OrderStatus = "DELIVERED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
	OrderStatusReturned   OrderStatus = "RETURNED"
)

// PaymentStatus defines the status of payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusPaid      PaymentStatus = "PAID"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
	PaymentStatusInitiated PaymentStatus = "INITIATED"
	PaymentStatusSuccess   PaymentStatus = "SUCCESS"
)

// OrphanReservation is stock a reserve took and nothing gave back. No order
// transition frees it, so the units stay unsellable until someone intervenes.
type OrphanReservation struct {
	ProductID   string    `json:"product_id" db:"product_id"`
	ProductName string    `json:"product_name" db:"product_name"`
	SKU         string    `json:"sku" db:"sku"`
	OrderID     string    `json:"order_id" db:"order_id"`
	Quantity    int       `json:"quantity" db:"quantity"`
	ReservedAt  time.Time `json:"reserved_at" db:"reserved_at"`
}

// InventoryTransactionType defines the type of inventory transaction
type InventoryTransactionType string

const (
	InventoryTransactionTypeAdd     InventoryTransactionType = "ADD"
	InventoryTransactionTypeRemove  InventoryTransactionType = "REMOVE"
	InventoryTransactionTypeReserve InventoryTransactionType = "RESERVE"
	InventoryTransactionTypeRelease InventoryTransactionType = "RELEASE"
	InventoryTransactionTypeAdjust  InventoryTransactionType = "ADJUST"
	InventoryTransactionTypeCommit  InventoryTransactionType = "COMMIT"
	InventoryTransactionTypeReturn  InventoryTransactionType = "RETURN"
)

// ==================== BASE ENTITY ====================

// BaseEntity contains common fields for all entities
type BaseEntity struct {
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at" db:"updated_at"`
	CreatedBy string    `json:"created_by,omitempty" dynamodbav:"created_by,omitempty" db:"created_by"`
	UpdatedBy string    `json:"updated_by,omitempty" dynamodbav:"updated_by,omitempty" db:"updated_by"`
}

// ==================== USER ENTITY ====================

// User represents an admin or operator user
type User struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Email        string     `json:"email" dynamodbav:"email"`
	PasswordHash string     `json:"-" dynamodbav:"password_hash"`
	FirstName    string     `json:"first_name" dynamodbav:"first_name"`
	LastName     string     `json:"last_name" dynamodbav:"last_name"`
	Phone        string     `json:"phone,omitempty" dynamodbav:"phone,omitempty"`
	Role         UserRole   `json:"role" dynamodbav:"role"`
	Permissions  []string   `json:"permissions" dynamodbav:"permissions"`
	Status       UserStatus `json:"status" dynamodbav:"status"`

	LastLoginAt *time.Time `json:"last_login_at,omitempty" dynamodbav:"last_login_at,omitempty"`
	BaseEntity
}

// TableName returns the DynamoDB table name for User
func (u *User) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for User
func (u *User) SetKeys() {
	u.PK = "USER#" + u.ID
	u.SK = SKMetadata
	u.GSI1PK = "USER_EMAIL"
	u.GSI1SK = u.Email
	u.EntityType = "USER"
}

// ==================== CATEGORY ENTITY ====================

// Category represents a flat product category
type Category struct {
	ID string `json:"id" db:"id"`

	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	Description string `json:"description,omitempty" db:"description"`
	ImageURL    string `json:"image_url,omitempty" db:"image_url"`

	// Attributes (loaded from category_attributes table)
	OwnAttributes []CategoryAttribute `json:"own_attributes,omitempty"`

	Status CategoryStatus `json:"status" db:"status"`

	// Counts
	ProductCount int `json:"product_count" db:"product_count"`

	BaseEntity
}

// NewCategory creates a Category from a CreateCategoryRequest.
func NewCategory(req CreateCategoryRequest, id, slug, createdBy string) *Category {
	c := &Category{
		ID:            id,
		Name:          req.Name,
		Slug:          slug,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		OwnAttributes: req.OwnAttributes,
		Status:        CategoryStatusActive,
	}
	c.CreatedBy = createdBy
	return c
}

// ApplyUpdate applies non-nil fields from an UpdateCategoryRequest to the category.
func (c *Category) ApplyUpdate(req UpdateCategoryRequest, slug string) {
	if req.Name != nil {
		c.Name = *req.Name
		c.Slug = slug
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.Status != nil {
		c.Status = *req.Status
	}
	if req.OwnAttributes != nil {
		c.OwnAttributes = req.OwnAttributes
	}
}

// CategoryAttribute defines an attribute for a category
type CategoryAttribute struct {
	Name         string            `json:"name" db:"name" validate:"required"`
	Label        string            `json:"label" db:"label"`
	Type         AttributeType     `json:"type" db:"type"`
	Required     bool              `json:"required" db:"required"`
	Searchable   bool              `json:"searchable" db:"searchable"` // Index for search queries AND show in filter UI
	DisplayOrder int               `json:"display_order" db:"display_order"`
	Options      []AttributeOption `json:"options,omitempty"`
}

// AttributeOption defines an option for SELECT/MULTI_SELECT attributes
type AttributeOption struct {
	Value     string `json:"value" db:"value"`
	Label     string `json:"label" db:"label"`
	Surcharge int64  `json:"surcharge,omitempty" db:"surcharge"` // in paise
}

// DimensionConfig defines custom dimension constraints for a category
type DimensionConfig struct {
	LengthEnabled bool    `json:"length_enabled" dynamodbav:"length_enabled"`
	LengthMin     float64 `json:"length_min,omitempty" dynamodbav:"length_min,omitempty"`
	LengthMax     float64 `json:"length_max,omitempty" dynamodbav:"length_max,omitempty"`
	LengthStep    float64 `json:"length_step,omitempty" dynamodbav:"length_step,omitempty"`
	LengthUnit    string  `json:"length_unit,omitempty" dynamodbav:"length_unit,omitempty"`

	WidthEnabled bool    `json:"width_enabled" dynamodbav:"width_enabled"`
	WidthMin     float64 `json:"width_min,omitempty" dynamodbav:"width_min,omitempty"`
	WidthMax     float64 `json:"width_max,omitempty" dynamodbav:"width_max,omitempty"`
	WidthStep    float64 `json:"width_step,omitempty" dynamodbav:"width_step,omitempty"`
	WidthUnit    string  `json:"width_unit,omitempty" dynamodbav:"width_unit,omitempty"`

	HeightEnabled bool    `json:"height_enabled" dynamodbav:"height_enabled"`
	HeightMin     float64 `json:"height_min,omitempty" dynamodbav:"height_min,omitempty"`
	HeightMax     float64 `json:"height_max,omitempty" dynamodbav:"height_max,omitempty"`
	HeightStep    float64 `json:"height_step,omitempty" dynamodbav:"height_step,omitempty"`
	HeightUnit    string  `json:"height_unit,omitempty" dynamodbav:"height_unit,omitempty"`

	PricingModel string `json:"pricing_model" dynamodbav:"pricing_model"` // AREA_BASED, LENGTH_BASED, VOLUME_BASED
}

// ProductImage represents an image for a product
type ProductImage struct {
	URL       string `json:"url" db:"url"`
	AltText   string `json:"alt_text,omitempty" db:"alt_text"`
	IsPrimary bool   `json:"is_primary" db:"is_primary"`
	SortOrder int    `json:"sort_order" db:"sort_order"`
}

// ==================== PRODUCT ENTITY ====================

// Product represents a handloom product
type Product struct {
	ID string `json:"id" db:"id"`

	// Basic Info
	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	SKU         string `json:"sku" db:"sku"`
	Description string `json:"description,omitempty" db:"description"`

	// Relations
	CategoryID string `json:"category_id" db:"category_id"`

	// Pricing (in paise)
	BasePrice    int64  `json:"base_price" db:"base_price"`
	SellingPrice int64  `json:"selling_price" db:"selling_price"`
	CostPrice    int64  `json:"cost_price,omitempty" db:"cost_price"`
	Currency     string `json:"currency" db:"currency"`

	// Dimensions (flattened to dim_length/dim_width/dim_height/dim_unit columns)
	Dimensions *Dimensions `json:"dimensions,omitempty"`
	Weight     int         `json:"weight,omitempty" db:"weight"` // in grams

	// Custom Dimension Support
	AllowCustomDimensions bool    `json:"allow_custom_dimensions" db:"allow_custom_dimensions"`
	PricingRuleID         *string `json:"pricing_rule_id,omitempty" db:"pricing_rule_id"`

	// Attributes (flexible storage for category-specific attributes; stored in product_attribute_values table)
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Common Attributes (indexed for filtering; stored in product_attribute_values table)
	Material  string `json:"material,omitempty"`
	Color     string `json:"color,omitempty"`
	WeaveType string `json:"weave_type,omitempty"`

	// Provenance (stored in product_attribute_values table)
	Origin    string `json:"origin,omitempty"`
	CraftType string `json:"craft_type,omitempty"`

	// Media (stored in product_images table)
	Images []ProductImage `json:"images,omitempty"`

	// Video (stored on products row; one optional video per product)
	VideoURL       string `json:"video_url,omitempty" db:"video_url"`
	VideoPosterURL string `json:"video_poster_url,omitempty" db:"video_poster_url"`

	// Tags & SEO
	Tags []string `json:"tags,omitempty" db:"tags"`

	// Inventory (LEFT JOINed from inventory table during List/Get queries)
	Quantity          int `json:"quantity" db:"inv_quantity"`
	ReservedQty       int `json:"reserved_qty" db:"inv_reserved_qty"`
	AvailableQty      int `json:"available_qty" db:"inv_available_qty"`
	LowStockThreshold int `json:"low_stock_threshold" db:"inv_low_stock_threshold"`

	Status    ProductStatus `json:"status" db:"status"`
	SortOrder int           `json:"sort_order" db:"sort_order"`

	// Embedding is the dense semantic vector (768-dim) used by hybrid search.
	// Nil when not yet embedded; backfill populates lazily.
	Embedding []float32 `json:"-"`

	// EmbeddingUpdatedAt records when the embedding was last written.
	// Nil paired with non-nil Embedding should not occur.
	EmbeddingUpdatedAt *time.Time `json:"-" db:"embedding_updated_at"`

	BaseEntity
}

// NewProduct creates a Product from a CreateProductRequest.
func NewProduct(req CreateProductRequest, id, slug, createdBy string) *Product {
	p := &Product{
		ID:                    id,
		Name:                  req.Name,
		Slug:                  slug,
		SKU:                   req.SKU,
		Description:           req.Description,
		CategoryID:            req.CategoryID,
		BasePrice:             req.BasePrice,
		SellingPrice:          req.SellingPrice,
		CostPrice:             req.CostPrice,
		Currency:              "INR",
		Dimensions:            req.Dimensions,
		Weight:                req.Weight,
		AllowCustomDimensions: req.AllowCustomDimensions,
		PricingRuleID:         req.PricingRuleID,
		Attributes:            req.Attributes,
		Material:              req.Material,
		Color:                 req.Color,
		WeaveType:             req.WeaveType,
		Origin:                req.Origin,
		CraftType:             req.CraftType,
		Images:                req.Images,
		VideoURL:              req.VideoURL,
		VideoPosterURL:        req.VideoPosterURL,
		Tags:                  emptyIfNil(req.Tags),
		Quantity:              req.InitialStock,
		AvailableQty:          req.InitialStock,
		LowStockThreshold:     req.LowStockThreshold,
		Status:                productStatusOrDefault(req.Status),
	}
	p.CreatedBy = createdBy
	return p
}

// emptyIfNil returns s if non-nil, otherwise an empty slice.
func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// productStatusOrDefault returns the given status or DRAFT if nil.
func productStatusOrDefault(s *ProductStatus) ProductStatus {
	if s != nil {
		return *s
	}
	return ProductStatusDraft
}

// ApplyUpdate applies non-nil fields from an UpdateProductRequest to the product.
func (p *Product) ApplyUpdate(req UpdateProductRequest) {
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.BasePrice != nil {
		p.BasePrice = *req.BasePrice
	}
	if req.SellingPrice != nil {
		p.SellingPrice = *req.SellingPrice
	}
	if req.CostPrice != nil {
		p.CostPrice = *req.CostPrice
	}
	if req.Dimensions != nil {
		p.Dimensions = req.Dimensions
	}
	if req.Weight != nil {
		p.Weight = *req.Weight
	}
	if req.AllowCustomDimensions != nil {
		p.AllowCustomDimensions = *req.AllowCustomDimensions
	}
	if req.PricingRuleID != nil {
		p.PricingRuleID = req.PricingRuleID
	}
	if req.Attributes != nil {
		p.Attributes = req.Attributes
	}
	if req.Material != nil {
		p.Material = *req.Material
	}
	if req.Color != nil {
		p.Color = *req.Color
	}
	if req.WeaveType != nil {
		p.WeaveType = *req.WeaveType
	}
	if req.Origin != nil {
		p.Origin = *req.Origin
	}
	if req.CraftType != nil {
		p.CraftType = *req.CraftType
	}
	if req.Images != nil {
		p.Images = req.Images
	}
	if req.Tags != nil {
		p.Tags = req.Tags
	}
	if req.LowStockThreshold != nil {
		p.LowStockThreshold = *req.LowStockThreshold
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.VideoURL != nil {
		p.VideoURL = *req.VideoURL
	}
	if req.VideoPosterURL != nil {
		p.VideoPosterURL = *req.VideoPosterURL
	}
}

// Dimensions represents product dimensions
type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height,omitempty"`
	Unit   string  `json:"unit"`
}

// ==================== PRICING RULE ENTITY ====================

// PricingRule defines how to calculate price for a category/product
type PricingRule struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK     string `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK     string `json:"-" dynamodbav:"GSI2SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description,omitempty" dynamodbav:"description,omitempty"`

	// Scope
	ScopeType    PricingRuleScope `json:"scope_type" dynamodbav:"scope_type"`
	ScopeID      string           `json:"scope_id,omitempty" dynamodbav:"scope_id,omitempty"`
	CategoryID   *string          `json:"category_id,omitempty" dynamodbav:"category_id,omitempty"`
	MaterialName *string          `json:"material_name,omitempty" dynamodbav:"material_name,omitempty"`

	// Pricing Type
	PricingType PricingType `json:"pricing_type" dynamodbav:"pricing_type"`

	// Base Pricing
	BasePrice    int64       `json:"base_price" dynamodbav:"base_price"`         // in paise
	PricePerUnit int64       `json:"price_per_unit" dynamodbav:"price_per_unit"` // in paise
	Unit         PricingUnit `json:"unit" dynamodbav:"unit"`

	// Material Multipliers
	MaterialMultipliers map[string]float64 `json:"material_multipliers,omitempty" dynamodbav:"material_multipliers,omitempty"`

	// Attribute Surcharges
	AttributeSurcharges []AttributeSurcharge `json:"attribute_surcharges,omitempty" dynamodbav:"attribute_surcharges,omitempty"`

	// Tiered Pricing
	Tiers []PricingTier `json:"tiers,omitempty" dynamodbav:"tiers,omitempty"`

	// Formula
	Formula string `json:"formula,omitempty" dynamodbav:"formula,omitempty"`

	// Constraints
	MinArea       *float64 `json:"min_area,omitempty" dynamodbav:"min_area,omitempty"`
	MaxArea       *float64 `json:"max_area,omitempty" dynamodbav:"max_area,omitempty"`
	MinOrderValue int64    `json:"min_order_value" dynamodbav:"min_order_value"`

	// Status
	Priority   int        `json:"priority" dynamodbav:"priority"`
	IsActive   bool       `json:"is_active" dynamodbav:"is_active"`
	ValidFrom  *time.Time `json:"valid_from,omitempty" dynamodbav:"valid_from,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty" dynamodbav:"valid_until,omitempty"`

	BaseEntity
}

// TableName returns the DynamoDB table name for PricingRule
func (p *PricingRule) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for PricingRule
func (p *PricingRule) SetKeys() {
	p.PK = "PRICING_RULE#" + p.ID
	p.SK = SKMetadata
	p.GSI1PK = "SCOPE#" + string(p.ScopeType)
	if p.ScopeID != "" {
		p.GSI1SK = p.ScopeID
	} else {
		p.GSI1SK = "GLOBAL"
	}
	p.GSI2PK = "PRICING_RULE#ALL"
	p.GSI2SK = "PRICING_RULE#" + p.ID
	p.EntityType = "PRICING_RULE"
}

// AttributeSurcharge defines a surcharge for a specific attribute value
type AttributeSurcharge struct {
	AttributeName  string        `json:"attribute_name" dynamodbav:"attribute_name"`
	AttributeValue string        `json:"attribute_value" dynamodbav:"attribute_value"`
	SurchargeType  SurchargeType `json:"surcharge_type" dynamodbav:"surcharge_type"`
	SurchargeValue int64         `json:"surcharge_value" dynamodbav:"surcharge_value"` // in paise or percentage*100
}

// PricingTier defines a tier for tiered pricing
type PricingTier struct {
	MinValue     float64 `json:"min_value" dynamodbav:"min_value"`
	MaxValue     float64 `json:"max_value" dynamodbav:"max_value"`
	PricePerUnit int64   `json:"price_per_unit" dynamodbav:"price_per_unit"`
}

// ==================== PRICE QUOTE ENTITY ====================

// PriceQuote represents a calculated price quote
type PriceQuote struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`
	TTL        int64  `json:"-" dynamodbav:"ttl"`

	CategoryID string                 `json:"category_id" dynamodbav:"category_id"`
	ProductID  *string                `json:"product_id,omitempty" dynamodbav:"product_id,omitempty"`
	Dimensions *Dimensions            `json:"dimensions" dynamodbav:"dimensions"`
	Attributes map[string]interface{} `json:"attributes" dynamodbav:"attributes"`
	Quantity   int                    `json:"quantity" dynamodbav:"quantity"`

	PricingRuleID   string          `json:"pricing_rule_id" dynamodbav:"pricing_rule_id"`
	CalculatedPrice int64           `json:"calculated_price" dynamodbav:"calculated_price"`
	PriceBreakdown  *PriceBreakdown `json:"price_breakdown" dynamodbav:"price_breakdown"`

	ValidUntil  time.Time `json:"valid_until" dynamodbav:"valid_until"`
	UsedInOrder *string   `json:"used_in_order,omitempty" dynamodbav:"used_in_order,omitempty"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

// TableName returns the DynamoDB table name for PriceQuote
func (p *PriceQuote) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for PriceQuote
func (p *PriceQuote) SetKeys() {
	p.PK = "QUOTE#" + p.ID
	p.SK = SKMetadata
	p.EntityType = "PRICE_QUOTE"
}

// PriceBreakdown shows the detailed price calculation
type PriceBreakdown struct {
	Area               float64           `json:"area,omitempty"`
	AreaUnit           string            `json:"area_unit,omitempty"`
	BaseCost           int64             `json:"base_cost"`
	MaterialMultiplier float64           `json:"material_multiplier,omitempty"`
	MaterialCost       int64             `json:"material_cost,omitempty"`
	Surcharges         []SurchargeDetail `json:"surcharges,omitempty"`
	SurchargesTotal    int64             `json:"surcharges_total"`
	SubtotalPerUnit    int64             `json:"subtotal_per_unit"`
	Quantity           int               `json:"quantity"`
	Total              int64             `json:"total"`
}

// SurchargeDetail shows details of a surcharge
type SurchargeDetail struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
	Amount    int64  `json:"amount"`
}

// ==================== INVENTORY ENTITY ====================

// Inventory represents inventory for a product
type Inventory struct {
	ID string `json:"id" db:"id"`

	ProductID   string `json:"product_id" db:"product_id"`
	ProductSKU  string `json:"product_sku,omitempty"`
	ProductName string `json:"product_name,omitempty"`

	Quantity     int `json:"quantity" db:"quantity"`
	ReservedQty  int `json:"reserved_qty" db:"reserved_qty"`
	AvailableQty int `json:"available_qty" db:"available_qty"`

	LowStockThreshold int `json:"low_stock_threshold" db:"low_stock_threshold"`
	ReorderPoint      int `json:"reorder_point" db:"reorder_point"`

	LastRestockAt *time.Time `json:"last_restock_at,omitempty" db:"last_restock_at"`

	BaseEntity
}

// InventoryTransaction represents a transaction in inventory
type InventoryTransaction struct {
	ID string `json:"id" db:"id"`

	ProductID     string                   `json:"product_id" db:"product_id"`
	Type          InventoryTransactionType `json:"type" db:"type"`
	Quantity      int                      `json:"quantity" db:"quantity"`
	PreviousQty   int                      `json:"previous_qty" db:"previous_qty"`
	NewQty        int                      `json:"new_qty" db:"new_qty"`
	Reason        string                   `json:"reason" db:"reason"`
	ReferenceType string                   `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   string                   `json:"reference_id,omitempty" db:"reference_id"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`

	// Resolved on read, not stored: created_by is an opaque id. Empty for
	// order-driven movements, which have no admin behind them.
	CreatedByName string `json:"created_by_name,omitempty" db:"-"`
}

// ==================== OTP ENTITY ====================

// OTP represents a one-time password for customer authentication
type OTP struct {
	PK         string    `json:"-" dynamodbav:"PK"`
	SK         string    `json:"-" dynamodbav:"SK"`
	EntityType string    `json:"-" dynamodbav:"entity_type"`
	Phone      string    `json:"phone" dynamodbav:"phone"`
	CodeHash   string    `json:"-" dynamodbav:"code_hash"`
	Attempts   int       `json:"attempts" dynamodbav:"attempts"`
	CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
	TTL        int64     `json:"-" dynamodbav:"ttl"`
}

// TableName returns the DynamoDB table name for OTP
func (o *OTP) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for OTP
func (o *OTP) SetKeys() {
	o.PK = "OTP#" + o.Phone
	o.SK = SKMetadata
	o.EntityType = "OTP"
}

// SendOTPRequest contains data for sending an OTP
type SendOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164"`
}

// VerifyOTPRequest contains data for verifying an OTP
type VerifyOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164"`
	Code  string `json:"code" validate:"required,len=6"`
}
