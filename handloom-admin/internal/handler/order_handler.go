// Package handler implements HTTP handlers
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/response"
)

// OrderHandler handles order-related requests
type OrderHandler struct {
	orderService   domain.OrderService
	paymentService domain.PaymentService
	refundService  domain.RefundService
	validation     *middleware.Validation
}

// NewOrderHandler creates a new OrderHandler
func NewOrderHandler(orderService domain.OrderService, paymentService domain.PaymentService, refundService domain.RefundService, validation *middleware.Validation) *OrderHandler {
	return &OrderHandler{
		orderService:   orderService,
		paymentService: paymentService,
		refundService:  refundService,
		validation:     validation,
	}
}

// Routes returns the order routes
func (h *OrderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateOrderRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/payment-status", h.CheckPaymentStatus)
	r.With(middleware.ValidateJSONTyped[UpdateOrderStatusRequest](h.validation)).Patch("/{id}/status", h.UpdateStatus)
	r.With(middleware.ValidateJSONTyped[AddOrderNoteRequest](h.validation)).Post("/{id}/notes", h.AddNote)
	r.With(middleware.ValidateJSONTyped[UpdateTrackingRequest](h.validation)).Patch("/{id}/tracking", h.UpdateTracking)
	r.With(middleware.ValidateJSONTyped[CancelOrderRequest](h.validation)).Post("/{id}/cancel", h.Cancel)
	// Refunds are admin-only end to end, the read included. Gated here, not at the mount
	// site, so the check travels with the routes into the monolith and the Lambda alike.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole(domain.UserRoleAdmin))
		r.Get("/{id}/refunds", h.ListRefunds)
		r.With(middleware.ValidateJSONTyped[domain.CreateRefundRequest](h.validation)).
			Post("/{id}/refunds", h.CreateRefund)
		r.With(middleware.ValidateJSONTyped[domain.PreviewRefundRequest](h.validation)).
			Post("/{id}/refunds/preview", h.PreviewRefund)
		r.Post("/{id}/refunds/{refundID}/recheck", h.RecheckRefund)
	})

	return r
}

// List handles listing orders
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := domain.ListOrdersRequest{
		PaginationRequest: parsePagination(r),
	}

	// Parse filters
	if status := r.URL.Query().Get("status"); status != "" {
		statusEnum := domain.OrderStatus(status)
		req.Status = &statusEnum
	}
	if paymentStatus := r.URL.Query().Get("payment_status"); paymentStatus != "" {
		psEnum := domain.PaymentStatus(paymentStatus)
		req.PaymentStatus = &psEnum
	}
	if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		req.CustomerID = &customerID
	}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		req.StartDate = &startDate
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		req.EndDate = &endDate
	}
	req.Search = r.URL.Query().Get("search")

	result, err := h.orderService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Create handles creating a new order
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.CreateOrderRequest](ctx)

	createdBy := getUserIDFromContext(ctx)
	order, err := h.orderService.Create(ctx, *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, order)
}

// GetByID handles getting an order by ID
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	order, err := h.orderService.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, order)
}

// CheckPaymentStatus checks the payment status from PhonePe for an order
func (h *OrderHandler) CheckPaymentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	result, err := h.paymentService.CheckProviderStatus(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// UpdateStatus handles updating order status
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[UpdateOrderStatusRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	if err := h.orderService.UpdateStatus(ctx, id, req.Status, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Order status updated successfully"})
}

// AddNote handles adding a note to an order
func (h *OrderHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[AddOrderNoteRequest](ctx)

	createdBy := getUserIDFromContext(ctx)
	if err := h.orderService.AddNote(ctx, id, req.Note, req.IsInternal, createdBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Note added successfully"})
}

// UpdateTracking handles updating tracking information
func (h *OrderHandler) UpdateTracking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[UpdateTrackingRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	if err := h.orderService.UpdateTracking(ctx, id, req.TrackingNumber, req.Carrier, req.TrackingURL, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Tracking updated successfully"})
}

// Cancel handles canceling an order
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[CancelOrderRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	if err := h.orderService.CancelOrder(ctx, id, req.Reason, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Order canceled successfully"})
}

// PreviewRefund prices a refund without raising one, so the screen an admin authorizes
// from shows the figure Create will use. Admin-only: it reads an order's money.
func (h *OrderHandler) PreviewRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.PreviewRefundRequest](ctx)

	preview, err := h.refundService.Preview(ctx, chi.URLParam(r, "id"), *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, preview)
}

// CreateRefund refunds part or all of an order. The body carries lines and quantities
// only: the amount is derived server-side, because money is not a client input.
func (h *OrderHandler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.CreateRefundRequest](ctx)

	refund, err := h.refundService.Create(ctx, id, *req, getUserIDFromContext(ctx))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, refund)
}

// ListRefunds returns every refund raised against an order.
func (h *OrderHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	refunds, err := h.refundService.ListByOrder(ctx, chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"refunds": refunds})
}

// RecheckRefund asks the provider for a refund's current state: the escape hatch for a
// webhook that never came, and the only recovery when no provider id was recorded.
func (h *OrderHandler) RecheckRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	refund, err := h.refundService.RecheckStatus(ctx, chi.URLParam(r, "id"), chi.URLParam(r, "refundID"))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, refund)
}
