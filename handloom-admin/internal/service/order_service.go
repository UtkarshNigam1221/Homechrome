// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/telemetry"
)

// OrderService implements domain.OrderService
type OrderService struct {
	orderRepo      domain.OrderRepository
	customerRepo   domain.CustomerRepository
	productRepo    domain.ProductRepository
	inventoryRepo  domain.InventoryRepository
	priceQuoteRepo domain.PriceQuoteRepository
	paymentRepo    domain.PaymentRepository
	pricingService domain.PricingService
}

// NewOrderService creates a new OrderService
func NewOrderService(
	orderRepo domain.OrderRepository,
	customerRepo domain.CustomerRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	paymentRepo domain.PaymentRepository,
	pricingService domain.PricingService,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		priceQuoteRepo: priceQuoteRepo,
		paymentRepo:    paymentRepo,
		pricingService: pricingService,
	}
}

// GetByID retrieves an order by ID
func (s *OrderService) GetByID(ctx context.Context, id string) (*domain.OrderWithDetails, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &domain.OrderWithDetails{
		Order: order,
	}

	// What has actually gone back, from the payment rather than reconstructed by
	// the caller. A missing payment simply means nothing has.
	if payment, payErr := s.paymentRepo.GetByOrderID(ctx, order.ID); payErr == nil && payment != nil {
		result.RefundedAmount = payment.RefundAmount
	}

	// Get customer details
	customer, err := s.customerRepo.GetByID(ctx, order.CustomerID)
	if err == nil {
		result.Customer = customer
	}

	// Batch-fetch all products for images in a single round-trip
	productIDs := make([]string, 0, len(order.Items))
	for _, item := range order.Items {
		productIDs = append(productIDs, item.ProductID)
	}
	productsByID := make(map[string]*domain.Product, len(productIDs))
	if len(productIDs) > 0 {
		products, batchErr := s.productRepo.BatchGetByIDs(ctx, productIDs)
		if batchErr == nil {
			for _, p := range products {
				productsByID[p.ID] = p
			}
		}
	}

	// Build item details
	for _, item := range order.Items {
		itemDetail := domain.OrderItemDetails{
			OrderItem:   item,
			ProductName: item.ProductName,
			ProductSKU:  item.ProductSKU,
		}
		if p, ok := productsByID[item.ProductID]; ok {
			itemDetail.ProductImages = p.Images
		}
		result.ItemDetails = append(result.ItemDetails, itemDetail)
	}

	return result, nil
}

// List retrieves orders with filters
func (s *OrderService) List(ctx context.Context, req domain.ListOrdersRequest) (*domain.ListOrdersResponse, error) {
	return s.orderRepo.List(ctx, req)
}

// UpdateStatus updates order status
func (s *OrderService) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus, updatedBy string) error {
	ctx, span := telemetry.StartServiceSpan(ctx, "order", "update_status")
	span.SetAttribute("entity.id", id)
	span.SetAttribute("entity.type", "order")
	span.SetAttribute("admin.action", "order.update_status")
	span.SetAttribute("order.new_status", string(status))

	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		span.EndWithError(err)
		return err
	}

	// Validate status transition
	if !isValidStatusTransition(order.Status, status) {
		transitionErr := errors.BadRequest(fmt.Sprintf("Cannot transition from %s to %s", order.Status, status))
		span.EndWithError(transitionErr)
		return transitionErr
	}

	if err := s.guardPaymentSettled(ctx, order, status); err != nil {
		span.EndWithError(err)
		return err
	}

	// Update status
	order.Status = status
	order.UpdatedBy = updatedBy
	now := time.Now()
	order.UpdatedAt = now

	// Set status-specific timestamp fields before persisting — these are part
	// of the write below, so they must land on `order` first.
	switch status {
	case domain.OrderStatusShipped:
		order.ShippedAt = &now
	case domain.OrderStatusDelivered:
		order.DeliveredAt = &now
	case domain.OrderStatusCancelled:
		// CancelledAt mirrors CancelOrder so tracking_handler's timeline shows the
		// cancellation whichever path set it. Reuses UpdatedAt, not a new time.Now().
		order.CancelledAt = &now
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		span.EndWithError(err)
		return err
	}

	s.applyInventoryEffect(ctx, order, status, updatedBy)

	slog.InfoContext(ctx, "Updated order status", "order_id", id, "status", status)
	span.End()
	return nil
}

// forwardStatuses are the moves that commit us to fulfilling an order, and so the
// ones worth blocking while the money is not in. DELIVERED is absent: the goods
// have already gone, and recording their arrival changes nothing. CANCELLED and
// RETURNED are the recovery paths and must stay open.
var forwardStatuses = map[domain.OrderStatus]bool{
	domain.OrderStatusConfirmed:  true,
	domain.OrderStatusProcessing: true,
	domain.OrderStatusShipped:    true,
}

// guardPaymentSettled refuses to move an order forward on money that never
// arrived. An order with no payment record was placed through the admin, where
// payment is handled off-platform — gating those would strand every phone order.
func (s *OrderService) guardPaymentSettled(ctx context.Context, order *domain.Order, status domain.OrderStatus) error {
	if !forwardStatuses[status] {
		return nil
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "Failed to read the order's payment")
	}
	if payment == nil {
		return nil
	}

	switch payment.Status {
	case domain.PaymentStatusPaid, domain.PaymentStatusSuccess, domain.PaymentStatusPartiallyRefunded:
		return nil
	}

	// The payment's own status, not the order's copy: the order's can be stale until
	// someone re-checks the provider, and that staleness is what shipped unpaid goods.
	return errors.BadRequest(fmt.Sprintf(
		"Cannot move the order to %s while its payment is %s — re-check the payment, then cancel or collect",
		status, payment.Status))
}

// retryOnce runs fn again after a short pause if it fails, unless the failure is
// the movement itself being refused rather than the database being unreachable.
func retryOnce(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil || terminal(err) {
		return err
	}

	// Safe because every order-scoped movement is idempotent per
	// (product, order, type): a retry of one that landed is a no-op.
	select {
	case <-ctx.Done():
		return err
	case <-time.After(250 * time.Millisecond):
	}
	return fn()
}

// terminal reports whether retrying could not possibly help.
func terminal(err error) bool {
	appErr, ok := errors.AsAppError(err)
	if !ok {
		return false
	}
	switch appErr.Code {
	case errors.ErrCodeInsufficientStock, errors.ErrCodeNotFound,
		errors.ErrCodeInventoryNotFound, errors.ErrCodeConflict:
		return true
	}
	return false
}

// orderQuantities aggregates an order's lines by product, so a status change moves
// stock for each product once rather than once per line.
func orderQuantities(items []domain.OrderItem) map[string]int {
	quantities := make(map[string]int, len(items))
	for _, item := range items {
		quantities[item.ProductID] += item.Quantity
	}
	return quantities
}

// applyInventoryEffect moves stock for a status change, only after the order
// write succeeds: AddStock has no idempotency guard, so a retry would double-add.
func (s *OrderService) applyInventoryEffect(ctx context.Context, order *domain.Order, status domain.OrderStatus, updatedBy string) {
	switch status {
	case domain.OrderStatusShipped:
		// Goods have left the warehouse: convert reservations into a dispatch.
		// available_qty is unaffected — these units were already unavailable.
		if commitErr := retryOnce(ctx, func() error {
			return s.inventoryRepo.CommitOrderStock(ctx, order.ID, orderQuantities(order.Items))
		}); commitErr != nil {
			slog.ErrorContext(ctx, "Failed to commit stock", keyOrderID, order.ID, "error", commitErr)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonCommit})
		}
	case domain.OrderStatusCancelled:
		if releaseErr := s.inventoryRepo.ReleaseOrderStock(ctx, order.ID, orderQuantities(order.Items)); releaseErr != nil {
			slog.ErrorContext(ctx, "Failed to release stock", keyOrderID, order.ID, "error", releaseErr)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonRelease})
		}
		// order_cancelled — status-transition path, no admin reason text.
		metrics.Record(ctx, "order_cancelled", metrics.L{
			metrics.LabelReason:  "status_update",
			metrics.LabelGateway: gatewayPhonePe,
		})
	case domain.OrderStatusReturned:
		// Goods are back. The repository restocks what the order committed, not what it
		// ordered: commit at dispatch is best-effort, so uncommitted lines stay out.
		if restockErr := s.inventoryRepo.RestockOrderStock(ctx, order.ID, updatedBy); restockErr != nil {
			slog.ErrorContext(ctx, "Failed to restock returned order", keyOrderID, order.ID, "error", restockErr)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonRestock})
		}
	}
	// domain.OrderStatusDelivered: no inventory effect — stock was committed
	// at dispatch. No case needed above.
}

// AddNote adds a note to an order
func (s *OrderService) AddNote(ctx context.Context, id string, note string, isInternal bool, createdBy string) error {
	orderNote := domain.OrderNote{
		ID:         uuid.New().String()[:8],
		Note:       note,
		IsInternal: isInternal,
		CreatedAt:  time.Now(),
		CreatedBy:  createdBy,
	}

	if err := s.orderRepo.AddNote(ctx, id, orderNote); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Added note to order", "order_id", id)
	return nil
}

// UpdateTracking updates tracking information
func (s *OrderService) UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, trackingURL string, updatedBy string) error {
	if err := s.orderRepo.UpdateTracking(ctx, id, trackingNumber, carrier, trackingURL); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Updated tracking for order", "order_id", id)
	return nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(ctx context.Context, id string, reason string, updatedBy string) error {
	ctx, span := telemetry.StartServiceSpan(ctx, "order", "cancel")
	span.SetAttribute("entity.id", id)
	span.SetAttribute("entity.type", "order")
	span.SetAttribute("admin.action", "order.cancel")

	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		span.EndWithError(err)
		return err
	}

	// Cancellable up to dispatch, mirroring validTransitions
	// (PENDING/CONFIRMED/PROCESSING -> CANCELLED); the two disagreed on PROCESSING.
	if order.Status != domain.OrderStatusPending &&
		order.Status != domain.OrderStatusConfirmed &&
		order.Status != domain.OrderStatusProcessing {
		cancelErr := errors.BadRequest("Order cannot be canceled in current status")
		span.EndWithError(cancelErr)
		return cancelErr
	}

	// Update status
	order.Status = domain.OrderStatusCancelled
	now := time.Now()
	order.CancelledAt = &now
	order.UpdatedBy = updatedBy

	if err := s.orderRepo.Update(ctx, order); err != nil {
		span.EndWithError(err)
		return err
	}

	if releaseErr := s.inventoryRepo.ReleaseOrderStock(ctx, order.ID, orderQuantities(order.Items)); releaseErr != nil {
		slog.ErrorContext(ctx, "Failed to release stock", keyOrderID, order.ID, "error", releaseErr)
		metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonRelease})
	}

	// order_cancelled — reason label is bounded (admin-supplied free text is
	// truncated to a small prefix to keep cardinality under control).
	reasonLabel := normaliseCancelReason(reason)
	metrics.Record(ctx, "order_cancelled", metrics.L{
		metrics.LabelReason:  reasonLabel,
		metrics.LabelGateway: gatewayPhonePe,
	})

	slog.InfoContext(ctx, "Canceled order", "order_id", id)
	span.End()
	return nil
}

// normaliseCancelReason maps free-text reasons to a bounded label set.
func normaliseCancelReason(reason string) string {
	r := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case r == "":
		return "unspecified"
	case strings.Contains(r, "payment"):
		return "payment_failed"
	case strings.Contains(r, "stock") || strings.Contains(r, "inventory"):
		return "out_of_stock"
	case strings.Contains(r, "fraud"):
		return "fraud"
	case strings.Contains(r, "customer"):
		return "customer_request"
	default:
		return labelOther
	}
}

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("HL%s%s", now.Format("20060102"), uuid.New().String()[:6])
}

// validTransitions defines allowed order status transitions. CONFIRMED may go
// straight to SHIPPED: fulfillment is manual and the PROCESSING stop cost 3 updates.
var validTransitions = map[domain.OrderStatus][]domain.OrderStatus{
	domain.OrderStatusPending:    {domain.OrderStatusConfirmed, domain.OrderStatusCancelled},
	domain.OrderStatusConfirmed:  {domain.OrderStatusProcessing, domain.OrderStatusShipped, domain.OrderStatusCancelled},
	domain.OrderStatusProcessing: {domain.OrderStatusShipped, domain.OrderStatusCancelled},
	domain.OrderStatusShipped:    {domain.OrderStatusDelivered, domain.OrderStatusReturned},
	domain.OrderStatusDelivered:  {domain.OrderStatusReturned},
}

// isValidStatusTransition checks if a status transition is valid
func isValidStatusTransition(from, to domain.OrderStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}

// Ensure interface compliance
var _ domain.OrderService = (*OrderService)(nil)
