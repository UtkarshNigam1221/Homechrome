package domain

import (
	"context"
)

//go:generate mockgen -source=repository.go -destination=../mocks/repository_mock.go -package=mocks

// ==================== PAGINATION ====================

// PaginationRequest contains cursor-based pagination parameters
type PaginationRequest struct {
	Limit   int    `json:"limit"`
	Cursor  string `json:"cursor,omitempty"`   // base64-encoded ExclusiveStartKey
	SortBy  string `json:"sort_by,omitempty"`
	SortDir string `json:"sort_dir,omitempty"` // asc or desc
}

// PaginationResponse contains cursor-based pagination metadata
type PaginationResponse struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// ==================== USER REPOSITORY ====================

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id string) error

	// List retrieves users with pagination and filters
	List(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, id string) error
}

// ListUsersRequest contains parameters for listing users
type ListUsersRequest struct {
	PaginationRequest
	Role   *UserRole   `json:"role,omitempty"`
	Status *UserStatus `json:"status,omitempty"`
	Search string      `json:"search,omitempty"`
}

// ListUsersResponse contains the list of users with pagination
type ListUsersResponse struct {
	Users      []*User            `json:"users"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== CATEGORY REPOSITORY ====================

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	// Create creates a new category
	Create(ctx context.Context, category *Category) error

	// GetByID retrieves a category by ID
	GetByID(ctx context.Context, id string) (*Category, error)

	// Update updates an existing category
	Update(ctx context.Context, category *Category) error

	// Delete deletes a category by ID
	Delete(ctx context.Context, id string) error

	// List retrieves categories with filters
	List(ctx context.Context, req ListCategoriesRequest) (*ListCategoriesResponse, error)

	// IncrementProductCount increments the product count
	IncrementProductCount(ctx context.Context, id string, delta int) error
}

// ListCategoriesRequest contains parameters for listing categories
type ListCategoriesRequest struct {
	PaginationRequest
	Status *CategoryStatus `json:"status,omitempty"`
	Search string          `json:"search,omitempty"`
}

// ListCategoriesResponse contains the list of categories
type ListCategoriesResponse struct {
	Categories []*Category        `json:"categories"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== PRODUCT REPOSITORY ====================

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	// CreateWithAttributeIndexes creates a product with its searchable attribute indexes in a transaction.
	// If inventory is non-nil it is included in the same transaction.
	CreateWithAttributeIndexes(ctx context.Context, product *Product, searchableAttrs map[string][]string, inventory *Inventory) error

	// GetByID retrieves a product by ID
	GetByID(ctx context.Context, id string) (*Product, error)

	// GetBySKU retrieves a product by SKU
	GetBySKU(ctx context.Context, sku string) (*Product, error)

	// UpdateWithAttributeIndexes updates a product and syncs its attribute indexes
	UpdateWithAttributeIndexes(ctx context.Context, product *Product, oldSearchableAttrs, newSearchableAttrs map[string][]string) error

	// Delete deletes a product by ID
	Delete(ctx context.Context, id string) error

	// DeleteWithAttributeIndexes deletes a product, its SKU uniqueness item, and its attribute indexes
	DeleteWithAttributeIndexes(ctx context.Context, id string, sku string, searchableAttrs map[string][]string) error

	// List retrieves products with filters
	List(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error)

	// GetByCategory retrieves products by category ID
	GetByCategory(ctx context.Context, categoryID string, pagination PaginationRequest) (*ListProductsResponse, error)

	// FilterByAttributes retrieves products by category and attribute filters
	FilterByAttributes(ctx context.Context, categoryID string, filters map[string][]string, pagination PaginationRequest) (*ListProductsResponse, error)

	// UpdateInventory updates the inventory fields
	UpdateInventory(ctx context.Context, id string, quantity, reservedQty, availableQty int) error

	// BatchGetByIDs retrieves multiple products by IDs
	BatchGetByIDs(ctx context.Context, ids []string) ([]*Product, error)

	// AddAttributeValues adds values to the stored distinct value sets for a category's searchable attributes
	AddAttributeValues(ctx context.Context, categoryID string, attrValues map[string][]string) error

	// GetAttributeValues reads the stored distinct value sets for a category
	GetAttributeValues(ctx context.Context, categoryID string) (map[string][]string, error)
}

// ListProductsRequest contains parameters for listing products
type ListProductsRequest struct {
	PaginationRequest
	CategoryID       *string             `json:"category_id,omitempty"`
	Status           *ProductStatus      `json:"status,omitempty"`
	Search           string              `json:"search,omitempty"`
	MinPrice         *int64              `json:"min_price,omitempty"`
	MaxPrice         *int64              `json:"max_price,omitempty"`
	InStock          *bool               `json:"in_stock,omitempty"`
	LowStock         *bool               `json:"low_stock,omitempty"`
	Material         *string             `json:"material,omitempty"`
	Color            *string             `json:"color,omitempty"`
	AttributeFilters map[string][]string `json:"attribute_filters,omitempty"` // Dynamic attribute filters: {"material": ["silk"], "color": ["red", "blue"]}
}

// ListProductsResponse contains the list of products
type ListProductsResponse struct {
	Products   []*Product         `json:"products"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== PRICING RULE REPOSITORY ====================

// PricingRuleRepository defines the interface for pricing rule data access
type PricingRuleRepository interface {
	// Create creates a new pricing rule
	Create(ctx context.Context, rule *PricingRule) error

	// GetByID retrieves a pricing rule by ID
	GetByID(ctx context.Context, id string) (*PricingRule, error)

	// Update updates an existing pricing rule
	Update(ctx context.Context, rule *PricingRule) error

	// Delete deletes a pricing rule by ID
	Delete(ctx context.Context, id string) error

	// List retrieves pricing rules with filters
	List(ctx context.Context, req ListPricingRulesRequest) (*ListPricingRulesResponse, error)

	// GetByScope retrieves pricing rules by scope
	GetByScope(ctx context.Context, scopeType PricingRuleScope, scopeID string) ([]*PricingRule, error)

	// GetApplicableRules retrieves all applicable rules for a category/product
	GetApplicableRules(ctx context.Context, categoryID string, productID *string, material *string) ([]*PricingRule, error)

	// GetGlobalRule retrieves the global fallback rule
	GetGlobalRule(ctx context.Context) (*PricingRule, error)
}

// ListPricingRulesRequest contains parameters for listing pricing rules
type ListPricingRulesRequest struct {
	PaginationRequest
	ScopeType   *PricingRuleScope `json:"scope_type,omitempty"`
	CategoryID  *string           `json:"category_id,omitempty"`
	PricingType *PricingType      `json:"pricing_type,omitempty"`
	IsActive    *bool             `json:"is_active,omitempty"`
	Search      string            `json:"search,omitempty"`
}

// ListPricingRulesResponse contains the list of pricing rules
type ListPricingRulesResponse struct {
	Rules      []*PricingRule     `json:"rules"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== PRICE QUOTE REPOSITORY ====================

// PriceQuoteRepository defines the interface for price quote data access
type PriceQuoteRepository interface {
	// Create creates a new price quote
	Create(ctx context.Context, quote *PriceQuote) error

	// GetByID retrieves a price quote by ID
	GetByID(ctx context.Context, id string) (*PriceQuote, error)

	// MarkAsUsed marks a quote as used in an order
	MarkAsUsed(ctx context.Context, id string, orderID string) error
}

// ==================== INVENTORY REPOSITORY ====================

// InventoryRepository defines the interface for inventory data access
type InventoryRepository interface {
	// Create creates a new inventory record
	Create(ctx context.Context, inventory *Inventory) error

	// GetByProductID retrieves inventory by product ID
	GetByProductID(ctx context.Context, productID string) (*Inventory, error)

	// Update updates an existing inventory record
	Update(ctx context.Context, inventory *Inventory) error

	// AddStock adds stock to inventory
	AddStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*InventoryTransaction, error)

	// RemoveStock removes stock from inventory
	RemoveStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*InventoryTransaction, error)

	// ReserveStock reserves stock for an order
	ReserveStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)

	// ReleaseStock releases reserved stock
	ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)

	// AdjustStock adjusts stock to a specific quantity
	AdjustStock(ctx context.Context, productID string, newQuantity int, reason string, userID string) (*InventoryTransaction, error)

	// GetTransactions retrieves inventory transactions
	GetTransactions(ctx context.Context, productID string, pagination PaginationRequest) (*ListInventoryTransactionsResponse, error)

	// GetLowStockProducts retrieves products with low stock
	GetLowStockProducts(ctx context.Context, pagination PaginationRequest) (*ListInventoryResponse, error)

	// DeleteByProductID deletes the inventory record and all its transactions for a product
	DeleteByProductID(ctx context.Context, productID string) error
}

// ListInventoryTransactionsResponse contains inventory transactions
type ListInventoryTransactionsResponse struct {
	Transactions []*InventoryTransaction `json:"transactions"`
	Pagination   PaginationResponse      `json:"pagination"`
}

// ListInventoryResponse contains inventory records
type ListInventoryResponse struct {
	Inventories []*Inventory       `json:"inventories"`
	Pagination  PaginationResponse `json:"pagination"`
}
