package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// AssetHandler handles asset-related HTTP requests
type AssetHandler struct {
	assetService *service.AssetService
	validation   *middleware.Validation
}

// NewAssetHandler creates a new AssetHandler
func NewAssetHandler(assetService *service.AssetService, validation *middleware.Validation) *AssetHandler {
	return &AssetHandler{
		assetService: assetService,
		validation:   validation,
	}
}

// Routes returns the asset routes
func (h *AssetHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.UploadAssetRequest](h.validation)).Post("/upload-url", h.GetUploadURL)
	r.Delete("/", h.DeleteAsset)

	return r
}

// GetUploadURL generates a presigned URL for uploading to tmp/
// POST /admin/assets/upload-url
func (h *AssetHandler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.UploadAssetRequest](r.Context())

	result, err := h.assetService.GetUploadURL(r.Context(), *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// DeleteAsset deletes a file from assets/ by its public URL
// DELETE /admin/assets
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	var req domain.DeleteAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if req.URL == "" {
		response.Error(w, errors.BadRequest("url is required"))
		return
	}

	if err := h.assetService.DeleteAsset(r.Context(), req.URL); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Asset deleted successfully",
	})
}
