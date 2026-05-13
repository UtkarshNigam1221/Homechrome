package store

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// OrderHandler handles customer-facing order requests.
// All routes require customer authentication (applied at router level).
type OrderHandler struct {
	orderService domain.OrderService
	orderRepo    domain.OrderRepository
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(
	os domain.OrderService,
	or domain.OrderRepository,
) *OrderHandler {
	return &OrderHandler{
		orderService: os,
		orderRepo:    or,
	}
}

// Routes returns the customer order routes.
func (h *OrderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListMyOrders)
	r.Get("/{id}", h.GetOrder)
	r.Post("/{id}/cancel", h.CancelOrder)

	return r
}

// ListMyOrders handles listing orders for the authenticated customer.
func (h *OrderHandler) ListMyOrders(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	pagination := parsePagination(r)

	result, err := h.orderRepo.GetByCustomer(ctx, customerID, pagination)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list customer orders", "error", err)
		response.Error(w, err)
		return
	}

	response.SuccessWithMeta(w, result.Orders, &response.Meta{
		Limit:      pagination.Limit,
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	})
}

// GetOrder handles getting a specific order for the authenticated customer.
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	id := chi.URLParam(r, "id")

	order, err := h.orderRepo.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Validate that the order belongs to the authenticated customer
	if order.CustomerID != customerID {
		response.NotFound(w, "Order")
		return
	}

	response.Success(w, order)
}

// CancelOrder handles canceling an order for the authenticated customer.
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Fetch the order to validate ownership and status
	order, err := h.orderRepo.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Validate ownership
	if order.CustomerID != customerID {
		response.NotFound(w, "Order")
		return
	}

	// Validate that the order can be canceled (only PENDING or CONFIRMED)
	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusConfirmed {
		response.Error(w, errors.BadRequest("Order cannot be canceled in its current status"))
		return
	}

	// Cancel via service (uses customerID as the actor)
	if err := h.orderService.CancelOrder(ctx, id, "Canceled by customer", customerID); err != nil {
		slog.ErrorContext(ctx, "failed to cancel order", "error", err)
		response.Error(w, err)
		return
	}

	response.Success(w, map[string]string{response.KeyMessage: "Order canceled successfully"})
}
