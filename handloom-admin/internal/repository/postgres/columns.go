package postgres

// Column name constants for PostgreSQL catalog tables.
// Use these in querybuilder calls and column lists to avoid magic strings.

// Common columns shared across multiple tables.
const (
	ColID        = "id"
	ColName      = "name"
	ColSlug      = "slug"
	ColStatus    = "status"
	ColCreatedAt = "created_at"
	ColUpdatedAt = "updated_at"
	ColCreatedBy = "created_by"
	ColUpdatedBy = "updated_by"
)

// Product table columns.
const (
	ColSKU                   = "sku"
	ColDescription           = "description"
	ColCategoryID            = "category_id"
	ColBasePrice             = "base_price"
	ColSellingPrice          = "selling_price"
	ColCostPrice             = "cost_price"
	ColCurrency              = "currency"
	ColDimLength             = "dim_length"
	ColDimWidth              = "dim_width"
	ColDimHeight             = "dim_height"
	ColDimUnit               = "dim_unit"
	ColWeight                = "weight"
	ColAllowCustomDimensions = "allow_custom_dimensions"
	ColPricingRuleID         = "pricing_rule_id"
	ColTags                  = "tags"
	ColSortOrder             = "sort_order"
)

// Category table columns.
const (
	ColImageURL     = "image_url"
	ColProductCount = "product_count"
)

// Category attribute table columns.
const (
	ColLabel        = "label"
	ColRequired     = "required"
	ColSearchable   = "searchable"
	ColDisplayOrder = "display_order"
)

// Category attribute option table columns.
const (
	ColAttributeID = "attribute_id"
	ColValue       = "value"
)

// Inventory table columns.
const (
	ColProductID         = "product_id"
	ColQuantity          = "quantity"
	ColReservedQty       = "reserved_qty"
	ColAvailableQty      = "available_qty"
	ColLowStockThreshold = "low_stock_threshold"
	ColReorderPoint      = "reorder_point"
	ColLastRestockAt     = "last_restock_at"
)

// Inventory transaction table columns.
const (
	ColType          = "type"
	ColPreviousQty   = "previous_qty"
	ColNewQty        = "new_qty"
	ColReason        = "reason"
	ColReferenceType = "reference_type"
	ColReferenceID   = "reference_id"
)

// Column lists for SELECT queries. Each slice matches the db struct tag order
// used by pgxscan so that queries and struct scanning stay in sync.

// productColumns lists the 24 columns selected for a product row.
var productColumns = []string{
	ColID, ColName, ColSlug, ColSKU, ColDescription, ColCategoryID,
	ColBasePrice, ColSellingPrice, ColCostPrice, ColCurrency,
	ColDimLength, ColDimWidth, ColDimHeight, ColDimUnit,
	ColWeight, ColAllowCustomDimensions, ColPricingRuleID,
	ColTags, ColStatus, ColSortOrder,
	ColCreatedAt, ColUpdatedAt, ColCreatedBy, ColUpdatedBy,
}

// categoryColumns lists the 11 columns selected for a category row.
var categoryColumns = []string{
	ColID, ColName, ColSlug, ColDescription, ColImageURL, ColStatus, ColProductCount,
	ColCreatedAt, ColUpdatedAt, ColCreatedBy, ColUpdatedBy,
}

// inventoryColumns lists the 12 columns selected for an inventory row.
var inventoryColumns = []string{
	ColID, ColProductID,
	ColQuantity, ColReservedQty, ColAvailableQty,
	ColLowStockThreshold, ColReorderPoint, ColLastRestockAt,
	ColCreatedAt, ColUpdatedAt, ColCreatedBy, ColUpdatedBy,
}

// inventoryTxnColumns lists the 11 columns selected for an inventory transaction row.
var inventoryTxnColumns = []string{
	ColID, ColProductID, ColType, ColQuantity, ColPreviousQty, ColNewQty,
	ColReason, ColReferenceType, ColReferenceID, ColCreatedAt, ColCreatedBy,
}

// prefixColumns returns a new slice with each column prefixed by "alias.".
// Useful for aliased queries (e.g. "p.id", "p.name").
func prefixColumns(alias string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = alias + "." + c
	}
	return out
}
