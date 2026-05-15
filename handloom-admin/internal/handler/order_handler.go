// Package handler implements HTTP handlers
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// OrderHandler handles order-related requests
type OrderHandler struct {
	orderService   domain.OrderService
	paymentService domain.PaymentService
	returnService  domain.ReturnService
	validation     *middleware.Validation
}

// NewOrderHandler creates a new OrderHandler
func NewOrderHandler(
	orderService domain.OrderService,
	paymentService domain.PaymentService,
	returnService domain.ReturnService,
	validation *middleware.Validation,
) *OrderHandler {
	return &OrderHandler{
		orderService:   orderService,
		paymentService: paymentService,
		returnService:  returnService,
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
	r.With(middleware.ValidateJSONTyped[RefundOrderRequest](h.validation)).Post("/{id}/refund", h.Refund)

	// Return endpoints scoped to an order
	r.Post("/{id}/returns", h.createReturn)
	r.Get("/{id}/returns", h.listOrderReturns)

	return r
}

// ReturnRoutes returns admin routes for managing existing returns. Mounted
// separately because the path is /returns/{id}/... not /orders/{id}/...
func (h *OrderHandler) ReturnRoutes() chi.Router {
	r := chi.NewRouter()
	r.Patch("/{id}/cancel", h.cancelReturn)
	r.Post("/{id}/refund", h.processRefund)
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
	if err := h.orderService.UpdateTracking(ctx, id, req.TrackingNumber, req.Carrier, updatedBy); err != nil {
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

// Refund handles initiating a refund
func (h *OrderHandler) Refund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[RefundOrderRequest](ctx)

	updatedBy := getUserIDFromContext(ctx)
	if err := h.orderService.RefundOrder(ctx, id, req.Amount, req.Reason, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyMessage: "Refund initiated successfully"})
}

// createReturn creates an admin-initiated customer return for the order.
func (h *OrderHandler) createReturn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := chi.URLParam(r, "id")
	var body struct {
		Items  []domain.ReturnItem `json:"items"`
		Reason string              `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, errors.BadRequest("Invalid JSON"))
		return
	}
	adminID := getUserIDFromContext(ctx)
	rr, err := h.returnService.Create(ctx, orderID, domain.CreateReturnRequest{
		Items:  body.Items,
		Reason: body.Reason,
	}, adminID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, rr)
}

// listOrderReturns lists returns associated with an order.
// TODO(phase-3): wire to ReturnRepository.ListByOrder when exposed via a service method.
func (h *OrderHandler) listOrderReturns(w http.ResponseWriter, _ *http.Request) {
	response.Error(w, errors.NotImplemented("Return listing wired in Phase 3"))
}

// cancelReturn cancels an in-flight return request.
func (h *OrderHandler) cancelReturn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	adminID := getUserIDFromContext(ctx)
	if err := h.returnService.Cancel(ctx, id, adminID); err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, map[string]any{"status": "cancelled"})
}

// processRefund finalizes a refund for a completed return.
func (h *OrderHandler) processRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var body struct {
		AmountPaise int64 `json:"amount_paise"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, errors.BadRequest("Invalid JSON"))
		return
	}
	if body.AmountPaise <= 0 {
		response.Error(w, errors.BadRequest("Refund amount must be positive"))
		return
	}
	adminID := getUserIDFromContext(ctx)
	if err := h.returnService.ProcessRefund(ctx, id, body.AmountPaise, adminID); err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, map[string]any{"status": "refunded"})
}
