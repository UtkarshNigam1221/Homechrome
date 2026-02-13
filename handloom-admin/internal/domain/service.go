package domain

import (
	"context"
	"time"
)

//go:generate mockgen -source=service.go -destination=../mocks/service_mock.go -package=mocks

// ==================== AUTH SERVICE ====================

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)

	// RefreshToken refreshes an access token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

	// Logout invalidates a user's tokens
	Logout(ctx context.Context, userID string) error

	// ValidateToken validates an access token and returns claims
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)

	// ChangePassword changes a user's password
	ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error

	// RequestPasswordReset initiates password reset flow
	RequestPasswordReset(ctx context.Context, email string) error

	// ResetPassword resets password with token
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
}

// LoginRequest contains login credentials
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginResponse contains login result
type LoginResponse struct {
	User   *User      `json:"user"`
	Tokens *TokenPair `json:"tokens"`
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TokenClaims contains JWT claims
type TokenClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Role        UserRole `json:"role"`
	Permissions []string `json:"permissions"`
}

// ChangePasswordRequest contains password change data
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordRequest contains password reset data
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ==================== USER SERVICE ====================

// UserService defines the interface for user operations
type UserService interface {
	// Create creates a new user
	Create(ctx context.Context, req CreateUserRequest, createdBy string) (*User, error)

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*User, error)

	// Update updates an existing user
	Update(ctx context.Context, id string, req UpdateUserRequest, updatedBy string) (*User, error)

	// Delete deletes a user by ID
	Delete(ctx context.Context, id string) error

	// List retrieves users with filters
	List(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)

	// UpdateStatus updates user status
	UpdateStatus(ctx context.Context, id string, status UserStatus, updatedBy string) error
}

// CreateUserRequest contains data for creating a user
type CreateUserRequest struct {
	Email       string   `json:"email" validate:"required,email"`
	Password    string   `json:"password" validate:"required,min=8"`
	FirstName   string   `json:"first_name" validate:"required"`
	LastName    string   `json:"last_name" validate:"required"`
	Phone       string   `json:"phone,omitempty"`
	Role        UserRole `json:"role" validate:"required,oneof=ADMIN OPERATOR"`
	Permissions []string `json:"permissions,omitempty"`
}

// UpdateUserRequest contains data for updating a user
type UpdateUserRequest struct {
	FirstName   *string     `json:"first_name,omitempty"`
	LastName    *string     `json:"last_name,omitempty"`
	Phone       *string     `json:"phone,omitempty"`
	Role        *UserRole   `json:"role,omitempty" validate:"omitempty,oneof=ADMIN OPERATOR"`
	Status      *UserStatus `json:"status,omitempty" validate:"omitempty,oneof=ACTIVE INACTIVE PENDING"`
	Password    *string     `json:"password,omitempty" validate:"omitempty,min=8"`
	Permissions []string    `json:"permissions,omitempty"`
}

// ==================== CATEGORY SERVICE ====================

// CategoryService defines the interface for category operations
type CategoryService interface {
	// Create creates a new category
	Create(ctx context.Context, req CreateCategoryRequest, createdBy string) (*Category, error)

	// GetByID retrieves a category by ID
	GetByID(ctx context.Context, id string) (*Category, error)

	// Update updates an existing category
	Update(ctx context.Context, id string, req UpdateCategoryRequest, updatedBy string) (*Category, error)

	// Delete deletes a category by ID
	Delete(ctx context.Context, id string) error

	// List retrieves categories with filters
	List(ctx context.Context, req ListCategoriesRequest) (*ListCategoriesResponse, error)

	// AddAttribute adds a new attribute to a category
	AddAttribute(ctx context.Context, categoryID string, attr CategoryAttribute, updatedBy string) (*Category, error)

	// UpdateAttribute updates an existing attribute
	UpdateAttribute(ctx context.Context, categoryID string, attrName string, attr CategoryAttribute, updatedBy string) (*Category, error)

	// DeleteAttribute removes an attribute from a category
	DeleteAttribute(ctx context.Context, categoryID string, attrName string, updatedBy string) error

	// GetAttributes retrieves all attributes for a category
	GetAttributes(ctx context.Context, categoryID string) (*CategoryAttributesResponse, error)
}

// CreateCategoryRequest contains data for creating a category
type CreateCategoryRequest struct {
	Name          string              `json:"name" validate:"required"`
	Description   string              `json:"description,omitempty"`
	ImageURL      string              `json:"image_url,omitempty"`
	OwnAttributes []CategoryAttribute `json:"own_attributes,omitempty"`
}

// UpdateCategoryRequest contains data for updating a category
type UpdateCategoryRequest struct {
	Name          *string             `json:"name,omitempty"`
	Description   *string             `json:"description,omitempty"`
	ImageURL      *string             `json:"image_url,omitempty"`
	Status        *CategoryStatus     `json:"status,omitempty"`
	OwnAttributes []CategoryAttribute `json:"own_attributes,omitempty"`
}

// CategorySummary contains minimal category info
type CategorySummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CategoryAttributesResponse contains category attributes
type CategoryAttributesResponse struct {
	OwnAttributes []CategoryAttribute `json:"own_attributes"`
	TotalCount    int                 `json:"total_count"`
}

// ==================== DESIGN SERVICE ====================

// DesignService defines the interface for design operations
type DesignService interface {
	// Create creates a new design
	Create(ctx context.Context, req CreateDesignRequest, createdBy string) (*Design, error)

	// GetByID retrieves a design by ID
	GetByID(ctx context.Context, id string) (*DesignWithCategory, error)

	// Update updates an existing design
	Update(ctx context.Context, id string, req UpdateDesignRequest, updatedBy string) (*Design, error)

	// Delete deletes a design by ID
	Delete(ctx context.Context, id string) error

	// List retrieves designs with filters
	List(ctx context.Context, req ListDesignsRequest) (*ListDesignsResponse, error)
}

// CreateDesignRequest contains data for creating a design
type CreateDesignRequest struct {
	Name        string            `json:"name" validate:"required"`
	CategoryID  string            `json:"category_id" validate:"required"`
	Description string            `json:"description,omitempty"`
	Images      []ProductImage    `json:"images,omitempty"`
	Attributes  []DesignAttribute `json:"attributes,omitempty"`
}

// UpdateDesignRequest contains data for updating a design
type UpdateDesignRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Images      []ProductImage    `json:"images,omitempty"`
	Attributes  []DesignAttribute `json:"attributes,omitempty"`
	Status      *DesignStatus     `json:"status,omitempty"`
}

// DesignWithCategory contains a design with its category info
type DesignWithCategory struct {
	*Design
	Category *CategorySummary `json:"category,omitempty"`
}

// ==================== PRODUCT SERVICE ====================

// ProductService defines the interface for product operations
type ProductService interface {
	// Create creates a new product
	Create(ctx context.Context, req CreateProductRequest, createdBy string) (*Product, error)

	// GetByID retrieves a product by ID
	GetByID(ctx context.Context, id string) (*ProductWithRelations, error)

	// Update updates an existing product
	Update(ctx context.Context, id string, req UpdateProductRequest, updatedBy string) (*Product, error)

	// Delete deletes a product by ID
	Delete(ctx context.Context, id string) error

	// List retrieves products with filters
	List(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error)

	// GetAttributeFilterOptions returns distinct values for all searchable attributes in a category
	GetAttributeFilterOptions(ctx context.Context, categoryID string) (map[string][]string, error)
}

// CreateProductRequest contains data for creating a product
type CreateProductRequest struct {
	Name                  string                 `json:"name" validate:"required"`
	SKU                   string                 `json:"sku" validate:"required"`
	DesignID              string                 `json:"design_id" validate:"required"`
	CategoryID            string                 `json:"category_id" validate:"required"`
	Description           string                 `json:"description,omitempty"`
	BasePrice             int64                  `json:"base_price" validate:"required,gt=0"`
	SellingPrice          int64                  `json:"selling_price" validate:"required,gt=0"`
	CostPrice             int64                  `json:"cost_price,omitempty"`
	Dimensions            *Dimensions            `json:"dimensions,omitempty"`
	Weight                int                    `json:"weight,omitempty"`
	AllowCustomDimensions bool                   `json:"allow_custom_dimensions"`
	PricingRuleID         *string                `json:"pricing_rule_id,omitempty"`
	Attributes            map[string]interface{} `json:"attributes,omitempty"`
	Material              string                 `json:"material,omitempty"`
	Color                 string                 `json:"color,omitempty"`
	WeaveType             string                 `json:"weave_type,omitempty"`
	Origin                string                 `json:"origin,omitempty"`
	CraftType             string                 `json:"craft_type,omitempty"`
	ArtisanID             *string                `json:"artisan_id,omitempty"`
	Images                []ProductImage         `json:"images,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	InitialStock          int                    `json:"initial_stock"`
	LowStockThreshold     int                    `json:"low_stock_threshold"`
}

// UpdateProductRequest contains data for updating a product
type UpdateProductRequest struct {
	Name                  *string                `json:"name,omitempty"`
	Description           *string                `json:"description,omitempty"`
	BasePrice             *int64                 `json:"base_price,omitempty"`
	SellingPrice          *int64                 `json:"selling_price,omitempty"`
	CostPrice             *int64                 `json:"cost_price,omitempty"`
	Dimensions            *Dimensions            `json:"dimensions,omitempty"`
	Weight                *int                   `json:"weight,omitempty"`
	AllowCustomDimensions *bool                  `json:"allow_custom_dimensions,omitempty"`
	PricingRuleID         *string                `json:"pricing_rule_id,omitempty"`
	Attributes            map[string]interface{} `json:"attributes,omitempty"`
	Material              *string                `json:"material,omitempty"`
	Color                 *string                `json:"color,omitempty"`
	WeaveType             *string                `json:"weave_type,omitempty"`
	Origin                *string                `json:"origin,omitempty"`
	CraftType             *string                `json:"craft_type,omitempty"`
	ArtisanID             *string                `json:"artisan_id,omitempty"`
	Images                []ProductImage         `json:"images,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	LowStockThreshold     *int                   `json:"low_stock_threshold,omitempty"`
	Status                *ProductStatus         `json:"status,omitempty"`
}

// ProductWithRelations contains a product with related entities
type ProductWithRelations struct {
	*Product
	Category  *CategorySummary `json:"category,omitempty"`
	Design    *DesignSummary   `json:"design,omitempty"`
	Inventory *Inventory       `json:"inventory,omitempty"`
}

// DesignSummary contains minimal design info
type DesignSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ==================== PRICING SERVICE ====================

// PricingService defines the interface for pricing operations
type PricingService interface {
	// CreateRule creates a new pricing rule
	CreateRule(ctx context.Context, req CreatePricingRuleRequest, createdBy string) (*PricingRule, error)

	// GetRule retrieves a pricing rule by ID
	GetRule(ctx context.Context, id string) (*PricingRule, error)

	// UpdateRule updates an existing pricing rule
	UpdateRule(ctx context.Context, id string, req UpdatePricingRuleRequest, updatedBy string) (*PricingRule, error)

	// DeleteRule deletes a pricing rule
	DeleteRule(ctx context.Context, id string) error

	// ListRules retrieves pricing rules with filters
	ListRules(ctx context.Context, req ListPricingRulesRequest) (*ListPricingRulesResponse, error)

	// GetRulesForCategory retrieves all applicable rules for a category
	GetRulesForCategory(ctx context.Context, categoryID string) (*CategoryPricingRulesResponse, error)

	// CalculatePrice calculates price for given dimensions and attributes
	CalculatePrice(ctx context.Context, req CalculatePriceRequest) (*CalculatePriceResponse, error)

	// GetDimensionOptions retrieves dimension options for a category
	GetDimensionOptions(ctx context.Context, categoryID string) (*DimensionOptionsResponse, error)

	// BulkCalculatePrice calculates prices for multiple configurations
	BulkCalculatePrice(ctx context.Context, req BulkCalculatePriceRequest) (*BulkCalculatePriceResponse, error)

	// GetQuote retrieves a price quote by ID
	GetQuote(ctx context.Context, quoteID string) (*PriceQuote, error)
}

// CreatePricingRuleRequest contains data for creating a pricing rule
type CreatePricingRuleRequest struct {
	Name                string               `json:"name" validate:"required"`
	Description         string               `json:"description,omitempty"`
	ScopeType           PricingRuleScope     `json:"scope_type" validate:"required"`
	ScopeID             string               `json:"scope_id,omitempty"`
	CategoryID          *string              `json:"category_id,omitempty"`
	MaterialName        *string              `json:"material_name,omitempty"`
	PricingType         PricingType          `json:"pricing_type" validate:"required"`
	BasePrice           int64                `json:"base_price"`
	PricePerUnit        int64                `json:"price_per_unit"`
	Unit                PricingUnit          `json:"unit"`
	MaterialMultipliers map[string]float64   `json:"material_multipliers,omitempty"`
	AttributeSurcharges []AttributeSurcharge `json:"attribute_surcharges,omitempty"`
	Tiers               []PricingTier        `json:"tiers,omitempty"`
	Formula             string               `json:"formula,omitempty"`
	MinArea             *float64             `json:"min_area,omitempty"`
	MaxArea             *float64             `json:"max_area,omitempty"`
	MinOrderValue       int64                `json:"min_order_value"`
	Priority            int                  `json:"priority" validate:"required"`
	IsActive            bool                 `json:"is_active"`
	ValidFrom           *time.Time           `json:"valid_from,omitempty"`
	ValidUntil          *time.Time           `json:"valid_until,omitempty"`
}

// UpdatePricingRuleRequest contains data for updating a pricing rule
type UpdatePricingRuleRequest struct {
	Name                *string              `json:"name,omitempty"`
	Description         *string              `json:"description,omitempty"`
	BasePrice           *int64               `json:"base_price,omitempty"`
	PricePerUnit        *int64               `json:"price_per_unit,omitempty"`
	MaterialMultipliers map[string]float64   `json:"material_multipliers,omitempty"`
	AttributeSurcharges []AttributeSurcharge `json:"attribute_surcharges,omitempty"`
	Tiers               []PricingTier        `json:"tiers,omitempty"`
	Priority            *int                 `json:"priority,omitempty"`
	IsActive            *bool                `json:"is_active,omitempty"`
	ValidFrom           *time.Time           `json:"valid_from,omitempty"`
	ValidUntil          *time.Time           `json:"valid_until,omitempty"`
}

// CategoryPricingRulesResponse contains pricing rules for a category
type CategoryPricingRulesResponse struct {
	CategoryRules []*PricingRuleSummary `json:"category_rules"`
	ParentRules   []*PricingRuleSummary `json:"parent_rules,omitempty"`
	GlobalRules   []*PricingRuleSummary `json:"global_rules,omitempty"`
	EffectiveRule *PricingRuleSummary   `json:"effective_rule,omitempty"`
}

// PricingRuleSummary contains minimal pricing rule info
type PricingRuleSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Priority       int    `json:"priority"`
	IsActive       bool   `json:"is_active"`
	SourceCategory string `json:"source_category,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// CalculatePriceRequest contains data for price calculation
type CalculatePriceRequest struct {
	ProductID  *string                `json:"product_id,omitempty"`
	CategoryID string                 `json:"category_id" validate:"required_without=ProductID"`
	Dimensions *Dimensions            `json:"dimensions" validate:"required"`
	Attributes map[string]interface{} `json:"attributes" validate:"required"`
	Quantity   int                    `json:"quantity" validate:"gte=1"`
}

// CalculatePriceResponse contains the calculated price
type CalculatePriceResponse struct {
	PriceBreakdown       *PriceBreakdown      `json:"price_breakdown"`
	FormattedPrice       *FormattedPrice      `json:"formatted_price"`
	PricingRuleID        string               `json:"pricing_rule_id"`
	QuoteID              string               `json:"quote_id"`
	QuoteValidUntil      time.Time            `json:"quote_valid_until"`
	DimensionConstraints *DimensionConfig     `json:"dimension_constraints,omitempty"`
}

// FormattedPrice contains formatted price strings
type FormattedPrice struct {
	Subtotal string `json:"subtotal"`
	Total    string `json:"total"`
	Currency string `json:"currency"`
}

// DimensionOptionsResponse contains dimension options for a category
type DimensionOptionsResponse struct {
	CategoryID            string              `json:"category_id"`
	CategoryName          string              `json:"category_name"`
	AllowCustomDimensions bool                `json:"allow_custom_dimensions"`
	DimensionConfig       *DimensionConfig    `json:"dimension_config,omitempty"`
	PricingModel          string              `json:"pricing_model"`
	PricingAttributes     []PricingAttribute  `json:"pricing_attributes,omitempty"`
	MinOrderValue         int64               `json:"min_order_value"`
}

// PricingAttribute is an attribute that affects pricing
type PricingAttribute struct {
	Name           string                  `json:"name"`
	Label          string                  `json:"label"`
	Type           AttributeType           `json:"type"`
	AffectsPricing bool                    `json:"affects_pricing"`
	Options        []PricingAttributeOption `json:"options,omitempty"`
}

// PricingAttributeOption is an option with pricing info
type PricingAttributeOption struct {
	Value           string  `json:"value"`
	Label           string  `json:"label"`
	PriceMultiplier float64 `json:"price_multiplier,omitempty"`
	Surcharge       int64   `json:"surcharge,omitempty"`
}

// BulkCalculatePriceRequest contains multiple configurations to calculate
type BulkCalculatePriceRequest struct {
	CategoryID     string                `json:"category_id" validate:"required"`
	Configurations []PriceConfiguration  `json:"configurations" validate:"required,max=10"`
}

// PriceConfiguration is a single configuration for bulk calculation
type PriceConfiguration struct {
	Dimensions *Dimensions            `json:"dimensions"`
	Attributes map[string]interface{} `json:"attributes"`
	Quantity   int                    `json:"quantity"`
}

// BulkCalculatePriceResponse contains multiple calculated prices
type BulkCalculatePriceResponse struct {
	Calculations    []BulkCalculationResult `json:"calculations"`
	QuoteID         string                  `json:"quote_id"`
	QuoteValidUntil time.Time               `json:"quote_valid_until"`
}

// BulkCalculationResult is a single calculation result
type BulkCalculationResult struct {
	ConfigurationIndex int                    `json:"configuration_index"`
	Dimensions         *Dimensions            `json:"dimensions"`
	Attributes         map[string]interface{} `json:"attributes"`
	Quantity           int                    `json:"quantity"`
	Price              int64                  `json:"price"`
	FormattedPrice     string                 `json:"formatted_price"`
	Error              string                 `json:"error,omitempty"`
}

// ==================== INVENTORY SERVICE ====================

// InventoryService defines the interface for inventory operations
type InventoryService interface {
	// GetByProductID retrieves inventory for a product
	GetByProductID(ctx context.Context, productID string) (*Inventory, error)

	// AddStock adds stock to a product
	AddStock(ctx context.Context, productID string, req AddStockRequest, userID string) (*InventoryTransactionResult, error)

	// RemoveStock removes stock from a product
	RemoveStock(ctx context.Context, productID string, req RemoveStockRequest, userID string) (*InventoryTransactionResult, error)

	// AdjustStock adjusts stock to a specific quantity
	AdjustStock(ctx context.Context, productID string, req AdjustStockRequest, userID string) (*InventoryTransactionResult, error)

	// GetTransactions retrieves inventory transactions
	GetTransactions(ctx context.Context, productID string, pagination PaginationRequest) (*ListInventoryTransactionsResponse, error)

	// GetLowStockProducts retrieves products with low stock
	GetLowStockProducts(ctx context.Context, pagination PaginationRequest) (*ListInventoryResponse, error)
}

// AddStockRequest contains data for adding stock
type AddStockRequest struct {
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Reason      string `json:"reason" validate:"required"`
	ReferenceID string `json:"reference_id,omitempty"`
}

// RemoveStockRequest contains data for removing stock
type RemoveStockRequest struct {
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Reason      string `json:"reason" validate:"required"`
	ReferenceID string `json:"reference_id,omitempty"`
}

// AdjustStockRequest contains data for adjusting stock
type AdjustStockRequest struct {
	NewQuantity int    `json:"new_quantity" validate:"gte=0"`
	Reason      string `json:"reason" validate:"required"`
}

// InventoryTransactionResult contains the result of an inventory operation
type InventoryTransactionResult struct {
	ProductID        string `json:"product_id"`
	PreviousQuantity int    `json:"previous_quantity"`
	ChangeQuantity   int    `json:"change_quantity"`
	NewQuantity      int    `json:"new_quantity"`
	AvailableQty     int    `json:"available_qty"`
	TransactionID    string `json:"transaction_id"`
}
