package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// CouponHandler handles coupon-related HTTP requests
type CouponHandler struct {
	couponService *service.CouponService
	validation    *middleware.Validation
}

// NewCouponHandler creates a new CouponHandler
func NewCouponHandler(couponService *service.CouponService, validation *middleware.Validation) *CouponHandler {
	return &CouponHandler{
		couponService: couponService,
		validation:    validation,
	}
}

// Routes returns the coupon routes
func (h *CouponHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateCouponRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	// PATCH, matching every other partial update in the admin API. The frontend has
	// always sent PATCH here, so PUT served nothing and every edit 405d.
	r.With(middleware.ValidateJSONTyped[domain.UpdateCouponRequest](h.validation)).Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.With(middleware.ValidateJSONTyped[ValidateCouponRequest](h.validation)).Post("/validate", h.Validate)
	r.With(middleware.ValidateJSONTyped[ApplyCouponRequest](h.validation)).Post("/apply", h.Apply)
	r.Get("/code/{code}", h.GetByCode)

	return r
}

// Create creates a new coupon
// POST /admin/coupons
func (h *CouponHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreateCouponRequest](r.Context())

	createdBy := middleware.GetCreatedBy(r.Context())

	coupon, err := h.couponService.Create(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, coupon)
}

// GetByID retrieves a coupon by ID
// GET /admin/coupons/{id}
func (h *CouponHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	coupon, err := h.couponService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, coupon)
}

// Update updates an existing coupon
// PATCH /admin/coupons/{id}
func (h *CouponHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.UpdateCouponRequest](r.Context())

	updatedBy := middleware.GetCreatedBy(r.Context())

	coupon, err := h.couponService.Update(r.Context(), id, *req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, coupon)
}

// Delete deletes a coupon by ID
// DELETE /admin/coupons/{id}
func (h *CouponHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.couponService.Delete(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// List retrieves coupons with filters
// GET /admin/coupons
func (h *CouponHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListCouponsRequest{
		PaginationRequest: parsePagination(r),
		Search:            r.URL.Query().Get("search"),
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.CouponStatus(status)
		req.Status = &s
	}

	if couponType := r.URL.Query().Get("type"); couponType != "" {
		t := domain.CouponType(couponType)
		req.Type = &t
	}

	if active := parseBoolParam(r.URL.Query().Get("is_active")); active != nil {
		req.IsActive = active
	}

	result, err := h.couponService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Validate validates a coupon for an order
// POST /admin/coupons/validate
func (h *CouponHandler) Validate(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[ValidateCouponRequest](r.Context())

	result, err := h.couponService.Validate(r.Context(), req.Code, req.OrderTotal, req.CustomerID, req.ProductIDs)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Apply applies a coupon to an order
// POST /admin/coupons/apply
func (h *CouponHandler) Apply(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[ApplyCouponRequest](r.Context())

	if err := h.couponService.Apply(r.Context(), req.CouponID, req.OrderID, req.CustomerID, req.Discount); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "applied"})
}

// GetByCode retrieves a coupon by code
// GET /admin/coupons/code/{code}
func (h *CouponHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	coupon, err := h.couponService.GetByCode(r.Context(), code)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, coupon)
}
