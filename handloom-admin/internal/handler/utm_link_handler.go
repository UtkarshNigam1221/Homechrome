package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// UTMLinkHandler handles UTM campaign link HTTP requests
type UTMLinkHandler struct {
	linkService *service.UTMLinkService
	validation  *middleware.Validation
}

// NewUTMLinkHandler creates a new UTMLinkHandler
func NewUTMLinkHandler(linkService *service.UTMLinkService, validation *middleware.Validation) *UTMLinkHandler {
	return &UTMLinkHandler{
		linkService: linkService,
		validation:  validation,
	}
}

// Routes returns the UTM link routes
func (h *UTMLinkHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateUTMLinkRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateUTMLinkRequest](h.validation)).Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// Create creates a new UTM link
// POST /admin/utm-links
func (h *UTMLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreateUTMLinkRequest](r.Context())

	createdBy := middleware.GetCreatedBy(r.Context())

	link, err := h.linkService.Create(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, link)
}

// GetByID retrieves a UTM link by ID
// GET /admin/utm-links/{id}
func (h *UTMLinkHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	link, err := h.linkService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, link)
}

// Update updates an existing UTM link
// PATCH /admin/utm-links/{id}
func (h *UTMLinkHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.UpdateUTMLinkRequest](r.Context())

	updatedBy := middleware.GetCreatedBy(r.Context())

	link, err := h.linkService.Update(r.Context(), id, *req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, link)
}

// Delete deletes a UTM link by ID
// DELETE /admin/utm-links/{id}
func (h *UTMLinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.linkService.Delete(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// List retrieves UTM links
// GET /admin/utm-links
func (h *UTMLinkHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListUTMLinksRequest{
		PaginationRequest: parsePagination(r),
		Search:            r.URL.Query().Get("search"),
	}

	result, err := h.linkService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
