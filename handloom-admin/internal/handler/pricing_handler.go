// Package handler implements HTTP handlers
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/response"
)

// PricingHandler handles pricing-related HTTP requests
type PricingHandler struct {
	pricingService domain.PricingService
	validation     *middleware.Validation
}

// NewPricingHandler creates a new PricingHandler
func NewPricingHandler(pricingService domain.PricingService, validation *middleware.Validation) *PricingHandler {
	return &PricingHandler{
		pricingService: pricingService,
		validation:     validation,
	}
}

// Routes returns the chi router for pricing endpoints
func (h *PricingHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Admin routes
	r.Route("/rules", func(r chi.Router) {
		r.With(middleware.ValidateJSONTyped[domain.CreatePricingRuleRequest](h.validation)).Post("/", h.CreateRule)
		r.Get("/", h.ListRules)
		r.Get("/{id}", h.GetRule)
		r.With(middleware.ValidateJSONTyped[domain.UpdatePricingRuleRequest](h.validation)).Patch("/{id}", h.UpdateRule)
		r.Delete("/{id}", h.DeleteRule)
		r.Get("/category/{categoryId}", h.GetRulesForCategory)
	})

	return r
}

// PublicRoutes returns the chi router for public pricing endpoints
func (h *PricingHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.CalculatePriceRequest](h.validation)).Post("/calculate", h.CalculatePrice)
	r.Get("/dimension-options/{categoryId}", h.GetDimensionOptions)
	r.With(middleware.ValidateJSONTyped[domain.BulkCalculatePriceRequest](h.validation)).Post("/bulk-calculate", h.BulkCalculatePrice)

	return r
}

// CreateRule handles POST /admin/pricing/rules
func (h *PricingHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreatePricingRuleRequest](r.Context())
	userID := getUserIDFromContext(r.Context())

	rule, err := h.pricingService.CreateRule(r.Context(), *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, rule)
}

// GetRule handles GET /admin/pricing/rules/{id}
func (h *PricingHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Rule ID is required")
		return
	}

	rule, err := h.pricingService.GetRule(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, rule)
}

// UpdateRule handles PATCH /admin/pricing/rules/{id}
func (h *PricingHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Rule ID is required")
		return
	}

	req := middleware.MustGetValidatedBody[domain.UpdatePricingRuleRequest](r.Context())
	userID := getUserIDFromContext(r.Context())

	rule, err := h.pricingService.UpdateRule(r.Context(), id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, rule)
}

// DeleteRule handles DELETE /admin/pricing/rules/{id}
func (h *PricingHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "Rule ID is required")
		return
	}

	err := h.pricingService.DeleteRule(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, map[string]string{response.KeyMessage: "Pricing rule deleted successfully"})
}

// ListRules handles GET /admin/pricing/rules
func (h *PricingHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	req := domain.ListPricingRulesRequest{
		PaginationRequest: parsePagination(r),
	}

	// Parse filters
	if scopeType := r.URL.Query().Get("scope_type"); scopeType != "" {
		st := domain.PricingRuleScope(scopeType)
		req.ScopeType = &st
	}
	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		req.CategoryID = &categoryID
	}
	if pricingType := r.URL.Query().Get("pricing_type"); pricingType != "" {
		pt := domain.PricingType(pricingType)
		req.PricingType = &pt
	}
	if isActive := r.URL.Query().Get("is_active"); isActive != "" {
		active := isActive == "true"
		req.IsActive = &active
	}
	req.Search = r.URL.Query().Get("search")

	result, err := h.pricingService.ListRules(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.SuccessWithMeta(w, result.Rules, &response.Meta{
		Limit:      result.Pagination.Limit,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	})
}

// GetRulesForCategory handles GET /admin/pricing/rules/category/{categoryId}
func (h *PricingHandler) GetRulesForCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	result, err := h.pricingService.GetRulesForCategory(r.Context(), categoryID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result)
}

// CalculatePrice handles POST /api/pricing/calculate
func (h *PricingHandler) CalculatePrice(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CalculatePriceRequest](r.Context())

	result, err := h.pricingService.CalculatePrice(r.Context(), *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result)
}

// GetDimensionOptions handles GET /api/pricing/dimension-options/{categoryId}
func (h *PricingHandler) GetDimensionOptions(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	result, err := h.pricingService.GetDimensionOptions(r.Context(), categoryID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result)
}

// BulkCalculatePrice handles POST /api/pricing/bulk-calculate
func (h *PricingHandler) BulkCalculatePrice(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.BulkCalculatePriceRequest](r.Context())

	result, err := h.pricingService.BulkCalculatePrice(r.Context(), *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result)
}
