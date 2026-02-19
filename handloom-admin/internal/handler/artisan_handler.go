package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// ArtisanHandler handles artisan-related HTTP requests
type ArtisanHandler struct {
	artisanService *service.ArtisanService
	validation     *middleware.Validation
}

// NewArtisanHandler creates a new ArtisanHandler
func NewArtisanHandler(artisanService *service.ArtisanService, validation *middleware.Validation) *ArtisanHandler {
	return &ArtisanHandler{
		artisanService: artisanService,
		validation:     validation,
	}
}

// Routes returns the artisan routes
func (h *ArtisanHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateArtisanRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateArtisanRequest](h.validation)).Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.With(middleware.ValidateJSONTyped[UpdateArtisanStatusRequest](h.validation)).Put("/{id}/status", h.UpdateStatus)
	r.Get("/{id}/products", h.GetProducts)
	r.Get("/{id}/payouts", h.GetPayouts)
	r.With(middleware.ValidateJSONTyped[domain.CreatePayoutRequest](h.validation)).Post("/{id}/payouts", h.CreatePayout)
	r.Get("/search", h.Search)

	return r
}

// Create creates a new artisan
// POST /admin/artisans
func (h *ArtisanHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreateArtisanRequest](r.Context())

	user := getUserFromContext(r.Context())
	createdBy := ""
	if user != nil {
		createdBy = user.ID
	}

	artisan, err := h.artisanService.Create(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, artisan)
}

// GetByID retrieves an artisan by ID
// GET /admin/artisans/{id}
func (h *ArtisanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	artisan, err := h.artisanService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, artisan)
}

// Update updates an existing artisan
// PATCH /admin/artisans/{id}
func (h *ArtisanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.UpdateArtisanRequest](r.Context())

	user := getUserFromContext(r.Context())
	updatedBy := ""
	if user != nil {
		updatedBy = user.ID
	}

	artisan, err := h.artisanService.Update(r.Context(), id, *req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, artisan)
}

// Delete deletes an artisan by ID
// DELETE /admin/artisans/{id}
func (h *ArtisanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.artisanService.Delete(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// List retrieves artisans with filters
// GET /admin/artisans
func (h *ArtisanHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListArtisansRequest{
		PaginationRequest: parsePagination(r),
		CraftType:         r.URL.Query().Get("craft_type"),
		Location:          r.URL.Query().Get("location"),
		Search:            r.URL.Query().Get("search"),
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.ArtisanStatus(status)
		req.Status = &s
	}

	result, err := h.artisanService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// UpdateStatus updates artisan status
// PATCH /admin/artisans/{id}/status
func (h *ArtisanHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[UpdateArtisanStatusRequest](r.Context())

	user := getUserFromContext(r.Context())
	updatedBy := ""
	if user != nil {
		updatedBy = user.ID
	}

	if err := h.artisanService.UpdateStatus(r.Context(), id, req.Status, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GetPayouts retrieves payouts for an artisan
// GET /admin/artisans/{id}/payouts
func (h *ArtisanHandler) GetPayouts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := parsePagination(r)

	result, err := h.artisanService.GetPayouts(r.Context(), id, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// CreatePayout creates a payout for an artisan
// POST /admin/artisans/{id}/payouts
func (h *ArtisanHandler) CreatePayout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.CreatePayoutRequest](r.Context())
	req.ArtisanID = id

	user := getUserFromContext(r.Context())
	createdBy := ""
	if user != nil {
		createdBy = user.ID
	}

	payout, err := h.artisanService.CreatePayout(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, payout)
}

// GetProducts retrieves products for an artisan
// GET /admin/artisans/{id}/products
func (h *ArtisanHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := parsePagination(r)

	result, err := h.artisanService.GetProducts(r.Context(), id, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Search searches artisans
// GET /admin/artisans/search
func (h *ArtisanHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	pagination := parsePagination(r)

	result, err := h.artisanService.Search(r.Context(), query, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
