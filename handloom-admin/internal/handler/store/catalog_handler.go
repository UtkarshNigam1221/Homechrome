// Package store implements public storefront HTTP handlers.
package store

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// CatalogHandler handles public storefront catalog requests.
type CatalogHandler struct {
	productService   domain.ProductService
	categoryService  domain.CategoryService
	inventoryService domain.InventoryService
	logger           *logger.Logger
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(
	ps domain.ProductService,
	cs domain.CategoryService,
	is domain.InventoryService,
	l *logger.Logger,
) *CatalogHandler {
	return &CatalogHandler{
		productService:   ps,
		categoryService:  cs,
		inventoryService: is,
		logger:           l,
	}
}

// Routes returns the public catalog routes.
func (h *CatalogHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/categories", h.ListCategories)
	r.Get("/categories/{idOrSlug}", h.GetCategory)
	r.Get("/products", h.ListProducts)
	r.Get("/products/search", h.SearchProducts)
	r.Get("/products/{idOrSlug}", h.GetProduct)
	r.Get("/products/{id}/availability", h.CheckAvailability)

	return r
}

// ==================== Response Types ====================

// StoreProduct is the public-facing product response that excludes sensitive data.
type StoreProduct struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	SKU         string `json:"sku"`
	Description string `json:"description,omitempty"`

	// Relations
	CategoryID string  `json:"category_id"`
	ArtisanID  *string `json:"artisan_id,omitempty"`

	// Pricing (in paise) — CostPrice intentionally excluded
	BasePrice    int64  `json:"base_price"`
	SellingPrice int64  `json:"selling_price"`
	Currency     string `json:"currency"`

	// Dimensions
	Dimensions *domain.Dimensions `json:"dimensions,omitempty"`
	Weight     int                `json:"weight,omitempty"`

	// Custom Dimension Support
	AllowCustomDimensions bool    `json:"allow_custom_dimensions"`
	PricingRuleID         *string `json:"pricing_rule_id,omitempty"`

	// Attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Common Attributes
	Material  string `json:"material,omitempty"`
	Color     string `json:"color,omitempty"`
	WeaveType string `json:"weave_type,omitempty"`

	// Provenance
	Origin    string `json:"origin,omitempty"`
	CraftType string `json:"craft_type,omitempty"`

	// Media
	Images []domain.ProductImage `json:"images,omitempty"`

	// Tags & SEO
	Tags []string `json:"tags,omitempty"`

	// Stock — public boolean instead of raw counts
	InStock bool `json:"in_stock"`

	// Category summary (populated when available)
	Category *domain.CategorySummary `json:"category,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StoreCategory is the public-facing category response.
type StoreCategory struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Slug          string                   `json:"slug"`
	Description   string                   `json:"description,omitempty"`
	ImageURL      string                   `json:"image_url,omitempty"`
	OwnAttributes []domain.CategoryAttribute `json:"own_attributes,omitempty"`
	ProductCount  int                      `json:"product_count"`
	Status        domain.CategoryStatus    `json:"status"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

// AvailabilityResponse is returned by the CheckAvailability endpoint.
type AvailabilityResponse struct {
	InStock           bool `json:"in_stock"`
	AvailableQuantity int  `json:"available_quantity"`
}

// StoreListProductsResponse wraps the public product list.
type StoreListProductsResponse struct {
	Products   []*StoreProduct          `json:"products"`
	Pagination domain.PaginationResponse `json:"pagination"`
}

// StoreCategoriesResponse wraps the public category list.
type StoreCategoriesResponse struct {
	Categories []*StoreCategory          `json:"categories"`
	Pagination domain.PaginationResponse `json:"pagination"`
}

// ==================== Conversion Helpers ====================

// toStoreProduct converts a domain Product to a public StoreProduct, stripping
// CostPrice and replacing raw inventory numbers with a simple InStock boolean.
func toStoreProduct(p *domain.Product, inStock bool) *StoreProduct {
	return &StoreProduct{
		ID:                    p.ID,
		Name:                  p.Name,
		Slug:                  p.Slug,
		SKU:                   p.SKU,
		Description:           p.Description,
		CategoryID:            p.CategoryID,
		ArtisanID:             p.ArtisanID,
		BasePrice:             p.BasePrice,
		SellingPrice:          p.SellingPrice,
		Currency:              p.Currency,
		Dimensions:            p.Dimensions,
		Weight:                p.Weight,
		AllowCustomDimensions: p.AllowCustomDimensions,
		PricingRuleID:         p.PricingRuleID,
		Attributes:            p.Attributes,
		Material:              p.Material,
		Color:                 p.Color,
		WeaveType:             p.WeaveType,
		Origin:                p.Origin,
		CraftType:             p.CraftType,
		Images:                p.Images,
		Tags:                  p.Tags,
		InStock:               inStock,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// toStoreProductFromRelations converts a ProductWithRelations to a StoreProduct.
func toStoreProductFromRelations(pwr *domain.ProductWithRelations) *StoreProduct {
	inStock := pwr.AvailableQty > 0
	sp := toStoreProduct(pwr.Product, inStock)
	sp.Category = pwr.Category
	return sp
}

// toStoreCategory converts a domain Category to a public StoreCategory.
func toStoreCategory(c *domain.Category) *StoreCategory {
	return &StoreCategory{
		ID:            c.ID,
		Name:          c.Name,
		Slug:          c.Slug,
		Description:   c.Description,
		ImageURL:      c.ImageURL,
		OwnAttributes: c.OwnAttributes,
		ProductCount:  c.ProductCount,
		Status:        c.Status,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// isUUID reports whether s is a valid UUID.
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// ==================== Handlers ====================

// ListCategories handles GET /store/categories
func (h *CatalogHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Only show active categories in the store.
	activeStatus := domain.CategoryStatusActive
	req := domain.ListCategoriesRequest{
		PaginationRequest: parsePagination(r),
		Status:            &activeStatus,
	}

	if search := r.URL.Query().Get("search"); search != "" {
		req.Search = search
	}

	result, err := h.categoryService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	cats := make([]*StoreCategory, 0, len(result.Categories))
	for _, c := range result.Categories {
		cats = append(cats, toStoreCategory(c))
	}

	response.SuccessWithMeta(w, cats, &response.Meta{
		Limit:      result.Pagination.Limit,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	})
}

// GetCategory handles GET /store/categories/{idOrSlug}
func (h *CatalogHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		response.BadRequest(w, "Category identifier is required")
		return
	}

	var cat *domain.Category

	if isUUID(idOrSlug) {
		// Try direct ID lookup first.
		c, err := h.categoryService.GetByID(ctx, idOrSlug)
		if err != nil {
			response.Error(w, err)
			return
		}
		cat = c
	} else {
		// Slug lookup: list with search and find exact slug match.
		cat = h.findCategoryBySlug(w, r, idOrSlug)
		if cat == nil {
			return // response already written
		}
	}

	// Only expose active categories on the storefront.
	if cat.Status != domain.CategoryStatusActive {
		response.NotFound(w, "Category")
		return
	}

	response.Success(w, toStoreCategory(cat))
}

// findCategoryBySlug searches for a category by slug. Returns nil and writes an
// error response if not found.
func (h *CatalogHandler) findCategoryBySlug(w http.ResponseWriter, r *http.Request, slug string) *domain.Category {
	ctx := r.Context()
	activeStatus := domain.CategoryStatusActive
	req := domain.ListCategoriesRequest{
		PaginationRequest: domain.PaginationRequest{Limit: 100},
		Status:            &activeStatus,
		Search:            slug,
	}

	result, err := h.categoryService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return nil
	}

	for _, c := range result.Categories {
		if c.Slug == slug {
			return c
		}
	}

	response.NotFound(w, "Category")
	return nil
}

// ListProducts handles GET /store/products
func (h *CatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Hardcode ACTIVE status filter for the storefront.
	activeStatus := domain.ProductStatusActive
	req := domain.ListProductsRequest{
		PaginationRequest: parsePagination(r),
		Status:            &activeStatus,
	}

	// Parse optional query params.
	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		req.CategoryID = &categoryID
	}
	if search := r.URL.Query().Get("search"); search != "" {
		req.Search = search
	}
	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if val, err := strconv.ParseInt(minPrice, 10, 64); err == nil {
			req.MinPrice = &val
		}
	}
	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if val, err := strconv.ParseInt(maxPrice, 10, 64); err == nil {
			req.MaxPrice = &val
		}
	}
	if material := r.URL.Query().Get("material"); material != "" {
		req.Material = &material
	}
	if color := r.URL.Query().Get("color"); color != "" {
		req.Color = &color
	}
	if attrFiltersJSON := r.URL.Query().Get("attribute_filters"); attrFiltersJSON != "" {
		var attrFilters map[string][]string
		if err := json.Unmarshal([]byte(attrFiltersJSON), &attrFilters); err == nil {
			req.AttributeFilters = attrFilters
		}
	}

	result, err := h.productService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	products := make([]*StoreProduct, 0, len(result.Products))
	for _, p := range result.Products {
		inStock := p.AvailableQty > 0
		products = append(products, toStoreProduct(p, inStock))
	}

	response.SuccessWithMeta(w, products, &response.Meta{
		Limit:      result.Pagination.Limit,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	})
}

// SearchProducts handles GET /store/products/search
// This is a convenience alias for ListProducts with the search parameter.
func (h *CatalogHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	h.ListProducts(w, r)
}

// GetProduct handles GET /store/products/{idOrSlug}
func (h *CatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		response.BadRequest(w, "Product identifier is required")
		return
	}

	if isUUID(idOrSlug) {
		// Direct ID lookup.
		pwr, err := h.productService.GetByID(ctx, idOrSlug)
		if err != nil {
			response.Error(w, err)
			return
		}

		if pwr.Status != domain.ProductStatusActive {
			response.NotFound(w, "Product")
			return
		}

		response.Success(w, toStoreProductFromRelations(pwr))
		return
	}

	// Slug lookup: search for the product by slug.
	product := h.findProductBySlug(w, r, idOrSlug)
	if product == nil {
		return // response already written
	}

	// Re-fetch via GetByID to get full relations.
	pwr, err := h.productService.GetByID(ctx, product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, toStoreProductFromRelations(pwr))
}

// findProductBySlug searches for an active product by slug. Returns nil and
// writes an error response if not found.
func (h *CatalogHandler) findProductBySlug(w http.ResponseWriter, r *http.Request, slug string) *domain.Product {
	ctx := r.Context()
	activeStatus := domain.ProductStatusActive
	req := domain.ListProductsRequest{
		PaginationRequest: domain.PaginationRequest{Limit: 100},
		Status:            &activeStatus,
		Search:            slug,
	}

	result, err := h.productService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return nil
	}

	for _, p := range result.Products {
		if p.Slug == slug {
			return p
		}
	}

	response.NotFound(w, "Product")
	return nil
}

// CheckAvailability handles GET /store/products/{id}/availability
func (h *CatalogHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Product ID is required")
		return
	}

	// Verify the product exists and is active.
	pwr, err := h.productService.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	if pwr.Status != domain.ProductStatusActive {
		response.NotFound(w, "Product")
		return
	}

	// Fetch live inventory data.
	inv, err := h.inventoryService.GetByProductID(ctx, id)
	if err != nil {
		// If inventory record not found, fall back to denormalized product data.
		if errors.IsNotFound(err) {
			response.Success(w, AvailabilityResponse{
				InStock:           pwr.AvailableQty > 0,
				AvailableQuantity: pwr.AvailableQty,
			})
			return
		}
		response.Error(w, err)
		return
	}

	response.Success(w, AvailabilityResponse{
		InStock:           inv.AvailableQty > 0,
		AvailableQuantity: inv.AvailableQty,
	})
}

// ==================== Internal Helpers ====================

// parsePagination parses cursor-based pagination parameters from the request.
func parsePagination(r *http.Request) domain.PaginationRequest {
	limit := 20
	sortBy := ""
	sortDir := "desc"

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	cursor := r.URL.Query().Get("cursor")

	if sb := r.URL.Query().Get("sort_by"); sb != "" {
		sortBy = sb
	}

	if sd := r.URL.Query().Get("sort_order"); sd == "asc" || sd == "desc" {
		sortDir = sd
	}

	return domain.PaginationRequest{
		Limit:   limit,
		Cursor:  cursor,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
}
