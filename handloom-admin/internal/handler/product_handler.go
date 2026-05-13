// Package handler implements HTTP handlers
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/response"
)

// ProductHandler handles product-related requests
type ProductHandler struct {
	productService   domain.ProductService
	inventoryService domain.InventoryService
	validation       *middleware.Validation
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(
	productService domain.ProductService,
	inventoryService domain.InventoryService,
	validation *middleware.Validation,
) *ProductHandler {
	return &ProductHandler{
		productService:   productService,
		inventoryService: inventoryService,
		validation:       validation,
	}
}

// Routes returns the product routes
func (h *ProductHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateProductRequest](h.validation)).Post("/", h.Create)
	r.Get("/filter-options/{categoryId}", h.GetAttributeFilterOptions)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateProductRequest](h.validation)).Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.With(middleware.ValidateJSONTyped[domain.ReorderProductsRequest](h.validation)).Put("/categories/{categoryId}/reorder", h.Reorder)

	// Inventory routes
	r.Get("/{id}/inventory", h.GetInventory)
	r.With(middleware.ValidateJSONTyped[domain.AddStockRequest](h.validation)).Post("/{id}/inventory/add", h.AddStock)
	r.With(middleware.ValidateJSONTyped[domain.RemoveStockRequest](h.validation)).Post("/{id}/inventory/remove", h.RemoveStock)
	r.With(middleware.ValidateJSONTyped[domain.AdjustStockRequest](h.validation)).Post("/{id}/inventory/adjust", h.AdjustStock)
	r.Get("/{id}/inventory/transactions", h.GetInventoryTransactions)

	return r
}

// List handles listing products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := r.URL.Query()
	req := domain.ListProductsRequest{
		PaginationRequest: parsePagination(r),
		Search:            q.Get("search"),
		CategoryID:        parseStringPtr(q.Get("category_id")),
		MinPrice:          parseInt64Ptr(q.Get("min_price")),
		MaxPrice:          parseInt64Ptr(q.Get("max_price")),
		InStock:           parseBoolParam(q.Get("in_stock")),
		LowStock:          parseBoolParam(q.Get("low_stock")),
		Material:          parseStringPtr(q.Get("material")),
		Color:             parseStringPtr(q.Get("color")),
	}
	if status := q.Get("status"); status != "" {
		statusEnum := domain.ProductStatus(status)
		req.Status = &statusEnum
	}
	if attrFiltersJSON := q.Get("attribute_filters"); attrFiltersJSON != "" {
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

	response.JSON(w, http.StatusOK, result)
}

// GetAttributeFilterOptions returns distinct values for each searchable attribute in a category
func (h *ProductHandler) GetAttributeFilterOptions(w http.ResponseWriter, r *http.Request) {
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

// Create handles creating a new product
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.CreateProductRequest](ctx)

	createdBy := getUserIDFromContext(ctx)
	product, err := h.productService.Create(ctx, *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, product)
}

// GetByID handles getting a product by ID
func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	product, err := h.productService.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, product)
}

// Update handles updating a product
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.UpdateProductRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	product, err := h.productService.Update(ctx, id, *req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, product)
}

// Delete handles deleting a product
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.productService.Delete(ctx, id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Product deleted successfully"})
}

// Reorder handles PUT /admin/products/categories/{categoryId}/reorder
func (h *ProductHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	req := middleware.MustGetValidatedBody[domain.ReorderProductsRequest](ctx)

	count, err := h.productService.ReorderProducts(ctx, categoryID, req.ProductIDs)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"reordered": count})
}

// GetInventory handles getting inventory for a product
func (h *ProductHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	inventory, err := h.inventoryService.GetByProductID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, inventory)
}

// AddStock handles adding stock to a product
func (h *ProductHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.AddStockRequest](ctx)

	userID := getUserIDFromContext(ctx)
	result, err := h.inventoryService.AddStock(ctx, id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// RemoveStock handles removing stock from a product
func (h *ProductHandler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.RemoveStockRequest](ctx)

	userID := getUserIDFromContext(ctx)
	result, err := h.inventoryService.RemoveStock(ctx, id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// AdjustStock handles adjusting stock for a product
func (h *ProductHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.AdjustStockRequest](ctx)

	userID := getUserIDFromContext(ctx)
	result, err := h.inventoryService.AdjustStock(ctx, id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetInventoryTransactions handles getting inventory transactions
func (h *ProductHandler) GetInventoryTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	pagination := parsePagination(r)
	result, err := h.inventoryService.GetTransactions(ctx, id, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// InventoryHandler handles inventory-related requests
type InventoryHandler struct {
	inventoryService domain.InventoryService
}

// NewInventoryHandler creates a new InventoryHandler
func NewInventoryHandler(inventoryService domain.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

// Routes returns the inventory routes
func (h *InventoryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/low-stock", h.GetLowStock)

	return r
}

// GetLowStock handles getting low stock products
func (h *InventoryHandler) GetLowStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pagination := parsePagination(r)
	result, err := h.inventoryService.GetLowStockProducts(ctx, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
