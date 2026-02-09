package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// CategoryHandler handles category-related HTTP requests
type CategoryHandler struct {
	categoryService domain.CategoryService
	logger          *logger.Logger
	validation      *middleware.Validation
}

// NewCategoryHandler creates a new CategoryHandler
func NewCategoryHandler(categoryService domain.CategoryService, logger *logger.Logger, validation *middleware.Validation) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		logger:          logger,
		validation:      validation,
	}
}

// Routes returns the chi router for category endpoints
func (h *CategoryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.CreateCategoryRequest](h.validation)).Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateCategoryRequest](h.validation)).Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	// Attribute management
	r.With(middleware.ValidateJSONTyped[domain.CategoryAttribute](h.validation)).Post("/{id}/attributes", h.AddAttribute)
	r.With(middleware.ValidateJSONTyped[domain.CategoryAttribute](h.validation)).Patch("/{id}/attributes/{attrName}", h.UpdateAttribute)
	r.Delete("/{id}/attributes/{attrName}", h.DeleteAttribute)
	r.Get("/{id}/attributes", h.GetAttributes)

	return r
}

// Create handles POST /admin/categories
func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreateCategoryRequest](r.Context())
	userID := getUserIDFromContext(r.Context())

	category, err := h.categoryService.Create(r.Context(), *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, category)
}

// GetByID handles GET /admin/categories/{id}
func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	category, err := h.categoryService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, category)
}

// Update handles PATCH /admin/categories/{id}
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	req := middleware.MustGetValidatedBody[domain.UpdateCategoryRequest](r.Context())
	userID := getUserIDFromContext(r.Context())

	category, err := h.categoryService.Update(r.Context(), id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, category)
}

// Delete handles DELETE /admin/categories/{id}
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	err := h.categoryService.Delete(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Category deleted successfully"})
}

// List handles GET /admin/categories
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListCategoriesRequest{
		PaginationRequest: parsePagination(r),
	}

	// Parse filters
	if status := r.URL.Query().Get("status"); status != "" {
		st := domain.CategoryStatus(status)
		req.Status = &st
	}

	result, err := h.categoryService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.SuccessWithMeta(w, result.Categories, &response.Meta{
		Limit:      result.Pagination.Limit,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	})
}

// AddAttribute handles POST /admin/categories/{id}/attributes
func (h *CategoryHandler) AddAttribute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	attr := middleware.MustGetValidatedBody[domain.CategoryAttribute](r.Context())
	userID := getUserIDFromContext(r.Context())

	category, err := h.categoryService.AddAttribute(r.Context(), id, *attr, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, map[string]interface{}{
		"attribute": attr,
		"category": map[string]interface{}{
			"id":                   category.ID,
			"own_attributes_count": len(category.OwnAttributes),
		},
	})
}

// UpdateAttribute handles PATCH /admin/categories/{id}/attributes/{attrName}
func (h *CategoryHandler) UpdateAttribute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	attrName := chi.URLParam(r, "attrName")
	if id == "" || attrName == "" {
		response.BadRequest(w, "Category ID and attribute name are required")
		return
	}

	attr := middleware.MustGetValidatedBody[domain.CategoryAttribute](r.Context())
	// Keep the same name
	attr.Name = attrName

	userID := getUserIDFromContext(r.Context())

	category, err := h.categoryService.UpdateAttribute(r.Context(), id, attrName, *attr, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"attribute":               attr,
		"affected_products_count": category.ProductCount,
	})
}

// DeleteAttribute handles DELETE /admin/categories/{id}/attributes/{attrName}
func (h *CategoryHandler) DeleteAttribute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	attrName := chi.URLParam(r, "attrName")
	if id == "" || attrName == "" {
		response.BadRequest(w, "Category ID and attribute name are required")
		return
	}

	userID := getUserIDFromContext(r.Context())

	err := h.categoryService.DeleteAttribute(r.Context(), id, attrName, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Attribute removed successfully"})
}

// GetAttributes handles GET /admin/categories/{id}/attributes
func (h *CategoryHandler) GetAttributes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	result, err := h.categoryService.GetAttributes(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result)
}
