// Package handler implements HTTP handlers
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// DesignHandler handles design-related requests
type DesignHandler struct {
	designService domain.DesignService
	logger        *logger.Logger
	validation    *middleware.Validation
}

// NewDesignHandler creates a new DesignHandler
func NewDesignHandler(designService domain.DesignService, logger *logger.Logger, validation *middleware.Validation) *DesignHandler {
	return &DesignHandler{
		designService: designService,
		logger:        logger,
		validation:    validation,
	}
}

// Routes returns the design routes
func (h *DesignHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateDesignRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateDesignRequest](h.validation)).Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// List handles listing designs
func (h *DesignHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := domain.ListDesignsRequest{
		PaginationRequest: parsePagination(r),
	}

	// Parse filters
	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		req.CategoryID = &categoryID
	}
	if status := r.URL.Query().Get("status"); status != "" {
		statusEnum := domain.DesignStatus(status)
		req.Status = &statusEnum
	}
	req.Search = r.URL.Query().Get("search")

	result, err := h.designService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Create handles creating a new design
func (h *DesignHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.CreateDesignRequest](ctx)

	createdBy := getUserIDFromContext(ctx)
	design, err := h.designService.Create(ctx, *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, design)
}

// GetByID handles getting a design by ID
func (h *DesignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	design, err := h.designService.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, design)
}

// Update handles updating a design
func (h *DesignHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.UpdateDesignRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	design, err := h.designService.Update(ctx, id, *req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, design)
}

// Delete handles deleting a design
func (h *DesignHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.designService.Delete(ctx, id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Design deleted successfully"})
}
