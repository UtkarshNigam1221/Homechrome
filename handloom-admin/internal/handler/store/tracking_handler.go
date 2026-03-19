package store

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// TrackingHandler handles public order tracking requests (no auth required).
type TrackingHandler struct {
	orderRepo    domain.OrderRepository
	shipmentRepo domain.ShipmentRepository
}

// NewTrackingHandler creates a new TrackingHandler.
func NewTrackingHandler(
	or domain.OrderRepository,
	sr domain.ShipmentRepository,
) *TrackingHandler {
	return &TrackingHandler{
		orderRepo:    or,
		shipmentRepo: sr,
	}
}

// Routes returns the public tracking routes.
func (h *TrackingHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/{orderNumber}", h.TrackOrder)

	return r
}

// ==================== Response Types ====================

// TrackingResponse is the combined tracking response for an order.
type TrackingResponse struct {
	OrderNumber   string             `json:"order_number"`
	Status        domain.OrderStatus `json:"status"`
	StatusHistory []StatusEntry      `json:"status_history"`
	Shipment      *ShipmentInfo      `json:"shipment,omitempty"`
}

// StatusEntry represents a single status change in the order timeline.
type StatusEntry struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Note      string    `json:"note,omitempty"`
}

// ShipmentInfo contains public-facing shipment tracking details.
type ShipmentInfo struct {
	AWBNumber     string `json:"awb_number,omitempty"`
	CourierName   string `json:"courier_name,omitempty"`
	TrackingURL   string `json:"tracking_url,omitempty"`
	Status        string `json:"status"`
	EstimatedDays string `json:"estimated_delivery,omitempty"`
}

// ==================== Handlers ====================

// TrackOrder handles tracking an order by order number.
// It looks up the order, builds a status timeline, and includes shipment info if available.
func (h *TrackingHandler) TrackOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderNumber := chi.URLParam(r, "orderNumber")

	// Look up the order by order number
	order, err := h.orderRepo.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		if errors.IsNotFound(err) {
			response.NotFound(w, "Order")
			return
		}
		slog.ErrorContext(ctx, "failed to look up order by number", "error", err)
		response.Error(w, err)
		return
	}

	// Build status history from the order's known timestamps
	statusHistory := buildStatusHistory(order)

	// Attempt to fetch shipment info (may not exist for all orders)
	var shipmentInfo *ShipmentInfo
	shipment, err := h.shipmentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		// Only log; shipment not found is not an error for tracking
		if !errors.IsNotFound(err) {
			slog.WarnContext(ctx, "failed to fetch shipment info", "error", err)
		}
	} else if shipment != nil {
		shipmentInfo = &ShipmentInfo{
			AWBNumber:     shipment.AWBNumber,
			CourierName:   shipment.CourierName,
			Status:        string(shipment.Status),
			EstimatedDays: shipment.EstimatedDelivery,
		}
	}

	resp := TrackingResponse{
		OrderNumber:   order.OrderNumber,
		Status:        order.Status,
		StatusHistory: statusHistory,
		Shipment:      shipmentInfo,
	}

	response.Success(w, resp)
}

// buildStatusHistory constructs a timeline of status changes from the order's
// known timestamps. Since there is no dedicated status-history repository method,
// we derive the timeline from the order's timestamp fields.
func buildStatusHistory(order *domain.Order) []StatusEntry {
	entries := []StatusEntry{
		{
			Status:    string(domain.OrderStatusPending),
			Timestamp: order.CreatedAt,
			Note:      "Order placed",
		},
	}

	// If the order was shipped, add the shipped entry
	if order.ShippedAt != nil {
		entries = append(entries, StatusEntry{
			Status:    string(domain.OrderStatusShipped),
			Timestamp: *order.ShippedAt,
			Note:      "Order shipped",
		})
	}

	// If the order was delivered, add the delivered entry
	if order.DeliveredAt != nil {
		entries = append(entries, StatusEntry{
			Status:    string(domain.OrderStatusDelivered),
			Timestamp: *order.DeliveredAt,
			Note:      "Order delivered",
		})
	}

	// If the order was canceled, add the canceled entry
	if order.CancelledAt != nil {
		entries = append(entries, StatusEntry{
			Status:    string(domain.OrderStatusCancelled),
			Timestamp: *order.CancelledAt,
			Note:      "Order canceled",
		})
	}

	return entries
}
