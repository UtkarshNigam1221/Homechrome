package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// AuditHandler handles audit-related HTTP requests
type AuditHandler struct {
	auditService *service.AuditService
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// List retrieves audit logs with filters
// GET /admin/audit
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListAuditLogsRequest{
		PaginationRequest: parsePagination(r),
	}

	if action := r.URL.Query().Get("action"); action != "" {
		req.Action = &action
	}

	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		req.EntityType = &entityType
	}

	if entityID := r.URL.Query().Get("entity_id"); entityID != "" {
		req.EntityID = &entityID
	}

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		req.UserID = &userID
	}

	result, err := h.auditService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetByID retrieves an audit log by ID
// GET /admin/audit/{id}
func (h *AuditHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	log, err := h.auditService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, log)
}

// GetByEntity retrieves audit logs for a specific entity
// GET /admin/audit/entity/{type}/{id}
func (h *AuditHandler) GetByEntity(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "type")
	entityID := chi.URLParam(r, "id")
	pagination := parsePagination(r)

	result, err := h.auditService.GetByEntity(r.Context(), entityType, entityID, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetByUser retrieves audit logs for a specific user
// GET /admin/audit/user/{id}
func (h *AuditHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	pagination := parsePagination(r)

	result, err := h.auditService.GetByUser(r.Context(), userID, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
