// Package store implements public storefront HTTP handlers.
package store

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// CatalogHandler handles public storefront catalog requests.
type CatalogHandler struct {
	productService   domain.ProductService
	categoryService  domain.CategoryService
	inventoryService domain.InventoryService
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(
	ps domain.ProductService,
	cs domain.CategoryService,
	is domain.InventoryService,
) *CatalogHandler {
	return &CatalogHandler{
		productService:   ps,
		categoryService:  cs,
		inventoryService: is,
	}
}

// Routes returns the public catalog routes.
func (h *CatalogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.CatalogCacheControl("public, max-age=3600"))

	r.Get("/categories", h.ListCategories)
	r.Get("/categories/{idOrSlug}", h.GetCategory)
	r.Get("/products", h.ListProducts)
	r.Get("/products/{idOrSlug}", h.GetProduct)
	r.Get("/products/filter-options/{categoryId}", h.GetFilterOptions)
	r.Get("/products/{id}/availability", h.CheckAvailability)

	return r
}

// Package-level active status pointers used by every handler to restrict
// storefront results to published entities.
var (
	activeCategoryStatus = domain.CategoryStatusActive
	activeProductStatus  = domain.ProductStatusActive
)

// ==================== Response Types ====================

// StoreProduct is the public-facing product response that excludes sensitive data.
type StoreProduct struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	SKU         string `json:"sku"`
	Description string `json:"description,omitempty"`

	// Relations
	CategoryID string `json:"category_id"`

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
	Images         []domain.ProductImage `json:"images,omitempty"`
	VideoURL       string                `json:"video_url,omitempty"`
	VideoPosterURL string                `json:"video_poster_url,omitempty"`

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
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Slug          string                     `json:"slug"`
	Description   string                     `json:"description,omitempty"`
	ImageURL      string                     `json:"image_url,omitempty"`
	OwnAttributes []domain.CategoryAttribute `json:"own_attributes,omitempty"`
	ProductCount  int                        `json:"product_count"`
	Status        domain.CategoryStatus      `json:"status"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

// AvailabilityResponse is returned by the CheckAvailability endpoint.
type AvailabilityResponse struct {
	InStock           bool `json:"in_stock"`
	AvailableQuantity int  `json:"available_quantity"`
}

// ==================== Conversion Helpers ====================

// toStoreProduct converts a domain Product to a public StoreProduct, stripping
// CostPrice and replacing raw inventory numbers with a simple InStock boolean.
func toStoreProduct(p *domain.Product) *StoreProduct {
	return &StoreProduct{
		ID:                    p.ID,
		Name:                  p.Name,
		Slug:                  p.Slug,
		SKU:                   p.SKU,
		Description:           p.Description,
		CategoryID:            p.CategoryID,
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
		VideoURL:              p.VideoURL,
		VideoPosterURL:        p.VideoPosterURL,
		Tags:                  p.Tags,
		InStock:               p.AvailableQty > 0,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// toStoreProductFromRelations converts a ProductWithRelations to a StoreProduct.
func toStoreProductFromRelations(pwr *domain.ProductWithRelations) *StoreProduct {
	sp := toStoreProduct(pwr.Product)
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

	req := domain.ListCategoriesRequest{
		PaginationRequest: parsePagination(r),
		Status:            &activeCategoryStatus,
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

	response.SuccessWithMeta(w, cats, paginationMeta(result.Pagination))
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
		found := h.findCategoryBySlug(w, r, idOrSlug)
		if found == nil {
			return // response already written
		}
		// Re-fetch by ID to include attributes.
		c, err := h.categoryService.GetByID(ctx, found.ID)
		if err != nil {
			response.Error(w, err)
			return
		}
		cat = c
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
	req := domain.ListCategoriesRequest{
		PaginationRequest: domain.PaginationRequest{Limit: 1},
		Status:            &activeCategoryStatus,
		Slug:              slug,
	}

	result, err := h.categoryService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return nil
	}

	if len(result.Categories) > 0 {
		return result.Categories[0]
	}

	response.NotFound(w, "Category")
	return nil
}

// ListProducts handles GET /store/products
func (h *CatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := domain.ListProductsRequest{
		PaginationRequest: parsePagination(r),
		Status:            &activeProductStatus,
	}

	// Parse optional query params.
	q := r.URL.Query()
	req.CategoryID = queryStr(q, "category_id")
	req.Search = q.Get("search")
	req.MinPrice = queryInt64(q, "min_price")
	req.MaxPrice = queryInt64(q, "max_price")
	if inStock := q.Get("in_stock"); inStock == "true" {
		t := true
		req.InStock = &t
	}
	req.Material = queryStr(q, "material")
	req.Color = queryStr(q, "color")

	// Parse attribute filters: prefer af_<name>=val1,val2 format (avoids JSON
	// encoding issues in query strings). Falls back to legacy JSON format.
	attrFilters := make(map[string][]string)
	for key, values := range q {
		if strings.HasPrefix(key, "af_") {
			attrName := strings.TrimPrefix(key, "af_")
			for _, v := range values {
				for _, sv := range strings.Split(v, ",") {
					sv = strings.TrimSpace(sv)
					if sv != "" {
						attrFilters[attrName] = append(attrFilters[attrName], sv)
					}
				}
			}
		}
	}
	if len(attrFilters) > 0 {
		req.AttributeFilters = attrFilters
	} else if attrJSON := q.Get("attribute_filters"); attrJSON != "" {
		var af map[string][]string
		if err := json.Unmarshal([]byte(attrJSON), &af); err == nil {
			req.AttributeFilters = af
		}
	}

	result, err := h.productService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	products := make([]*StoreProduct, 0, len(result.Products))
	for _, p := range result.Products {
		products = append(products, toStoreProduct(p))
	}

	response.SuccessWithMeta(w, products, paginationMeta(result.Pagination))
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
	req := domain.ListProductsRequest{
		PaginationRequest: domain.PaginationRequest{Limit: 1},
		Status:            &activeProductStatus,
		Slug:              slug,
	}

	result, err := h.productService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return nil
	}

	if len(result.Products) > 0 {
		return result.Products[0]
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

// GetFilterOptions handles GET /store/products/filter-options/{categoryId}
func (h *CatalogHandler) GetFilterOptions(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	options, err := h.productService.GetAttributeFilterOptions(r.Context(), categoryID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, options)
}

// ==================== Internal Helpers ====================

// parsePagination parses cursor-based pagination parameters from the request.
func parsePagination(r *http.Request) domain.PaginationRequest {
	q := r.URL.Query()
	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	sortDir := "desc"
	if sd := q.Get("sort_order"); sd == "asc" || sd == "desc" {
		sortDir = sd
	}

	return domain.PaginationRequest{
		Limit:   limit,
		Cursor:  q.Get("cursor"),
		SortBy:  q.Get("sort_by"),
		SortDir: sortDir,
	}
}

// paginationMeta converts a domain PaginationResponse to a response.Meta.
func paginationMeta(p domain.PaginationResponse) *response.Meta {
	return &response.Meta{
		Limit:      p.Limit,
		NextCursor: p.NextCursor,
		HasMore:    p.HasMore,
	}
}

// queryStr returns a *string if the key is present and non-empty, nil otherwise.
func queryStr(q url.Values, key string) *string {
	if v := q.Get(key); v != "" {
		return &v
	}
	return nil
}

// queryInt64 returns a *int64 if the key is a valid integer, nil otherwise.
func queryInt64(q url.Values, key string) *int64 {
	if v := q.Get(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &n
		}
	}
	return nil
}
