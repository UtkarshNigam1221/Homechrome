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

// Create creates a new order
func (s *OrderService) Create(ctx context.Context, req domain.CreateOrderRequest, createdBy string) (*domain.Order, error) {
	ctx, span := telemetry.StartServiceSpan(ctx, "order", "create")

	// Validate customer
	customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		notFoundErr := errors.New(errors.ErrCodeNotFound, "Customer not found")
		span.EndWithError(notFoundErr)
		return nil, notFoundErr
	}

	// Generate order number
	orderNumber := generateOrderNumber()

	// Process items
	var items []domain.OrderItem
	var subtotal int64

	for i, itemInput := range req.Items {
		// Get product
		product, err := s.productRepo.GetByID(ctx, itemInput.ProductID)
		if err != nil {
			notFoundErr := errors.New(errors.ErrCodeNotFound, fmt.Sprintf("Product not found: %s", itemInput.ProductID))
			span.EndWithError(notFoundErr)
			return nil, notFoundErr
		}

		var unitPrice int64
		var priceQuoteID *string

		// Calculate price
		if itemInput.QuoteID != nil {
			// Use existing quote
			var quote *domain.PriceQuote
			quote, err = s.pricingService.GetQuote(ctx, *itemInput.QuoteID)
			if err != nil {
				quoteErr := errors.New(errors.ErrCodeQuoteExpired, fmt.Sprintf("Invalid or expired quote for item %d", i+1))
				span.EndWithError(quoteErr)
				return nil, quoteErr
			}
			unitPrice = quote.CalculatedPrice / int64(quote.Quantity)
			priceQuoteID = itemInput.QuoteID

			// Mark quote as used
			if markErr := s.priceQuoteRepo.MarkAsUsed(ctx, *itemInput.QuoteID, orderNumber); markErr != nil {
				slog.ErrorContext(ctx, "Failed to mark price quote as used", "quote_id", *itemInput.QuoteID, "order_number", orderNumber, "error", markErr)
			}
		} else if itemInput.CustomDimensions != nil && product.AllowCustomDimensions {
			// Calculate custom price
			calcReq := domain.CalculatePriceRequest{
				ProductID:  &itemInput.ProductID,
				CategoryID: product.CategoryID,
				Dimensions: itemInput.CustomDimensions,
				Attributes: itemInput.Attributes,
				Quantity:   itemInput.Quantity,
			}
			var calcResp *domain.CalculatePriceResponse
			calcResp, err = s.pricingService.CalculatePrice(ctx, calcReq)
			if err != nil {
				priceErr := errors.Wrap(err, fmt.Sprintf("Failed to calculate price for item %d", i+1))
				span.EndWithError(priceErr)
				return nil, priceErr
			}
			unitPrice = calcResp.PriceBreakdown.SubtotalPerUnit
			quoteID := calcResp.QuoteID
			priceQuoteID = &quoteID
		} else {
			// Use standard price
			unitPrice = product.SellingPrice
		}

		// Check inventory
		inventory, err := s.inventoryRepo.GetByProductID(ctx, itemInput.ProductID)
		if err != nil {
			invErr := errors.New(errors.ErrCodeNotFound, fmt.Sprintf("Inventory not found for product: %s", itemInput.ProductID))
			span.EndWithError(invErr)
			return nil, invErr
		}

		if inventory.AvailableQty < itemInput.Quantity {
			stockErr := errors.New(errors.ErrCodeInsufficientStock, fmt.Sprintf("Insufficient stock for product: %s (available: %d, requested: %d)", product.Name, inventory.AvailableQty, itemInput.Quantity))
			span.EndWithError(stockErr)
			return nil, stockErr
		}

		item := domain.OrderItem{
			ID:          fmt.Sprintf("item_%d", i+1),
			ProductID:   itemInput.ProductID,
			ProductSKU:  product.SKU,
			ProductName: product.Name,
			CategoryID:  product.CategoryID,
			Quantity:    itemInput.Quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  unitPrice * int64(itemInput.Quantity),
			Dimensions:  itemInput.CustomDimensions,
			Attributes:  itemInput.Attributes,
			QuoteID:     priceQuoteID,
		}
		items = append(items, item)
		subtotal += item.TotalPrice
	}

	// TODO: Apply coupon if provided
	var discountAmount int64 = 0

	// Calculate shipping (simplified)
	var shippingCost int64 = 0 // Free shipping for now

	// Calculate tax (simplified - 18% GST)
	taxAmount := int64(float64(subtotal-discountAmount) * 0.18)

	totalAmount := subtotal - discountAmount + shippingCost + taxAmount

	order := &domain.Order{
		ID:              "order_" + uuid.New().String()[:8],
		OrderNumber:     orderNumber,
		CustomerID:      req.CustomerID,
		CustomerName:    customer.FirstName + " " + customer.LastName,
		CustomerEmail:   customer.Email,
		CustomerPhone:   customer.Phone,
		Items:           items,
		ItemCount:       len(items),
		Status:          domain.OrderStatusPending,
		PaymentStatus:   domain.PaymentStatusPending,
		Subtotal:        subtotal,
		DiscountAmount:  discountAmount,
		ShippingAmount:  shippingCost,
		TaxAmount:       taxAmount,
		TotalAmount:     totalAmount,
		Currency:        defaultCurrency,
		ShippingAddress: &req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		CouponCode:      req.CouponCode,
	}
	order.CreatedBy = createdBy

	if err := s.orderRepo.Create(ctx, order); err != nil {
		span.EndWithError(err)
		return nil, err
	}

	span.SetAttribute("entity.id", order.ID)
	span.SetAttribute("entity.type", "order")
	span.SetAttribute("admin.action", "order.create")
	span.SetAttribute("order.item_count", len(order.Items))

	// Emit KPI metrics for the placed order (admin channel — this is the admin-facing Create path).
	// Admin orders have no geo context; use "unknown" for country/city.
	metrics.Record(ctx, "orders_placed", metrics.L{metrics.LabelCountry: labelUnknown, metrics.LabelCity: labelUnknown})
	metrics.RecordSum(ctx, "orders_value", order.TotalAmount, metrics.L{
		metrics.LabelCountry: labelUnknown, metrics.LabelCity: labelUnknown, metrics.LabelGateway: string(domain.PaymentMethodUPI),
	})
	metrics.Record(ctx, "cart_size", metrics.L{
		metrics.LabelCountry: labelUnknown,
		metrics.LabelBucket:  metrics.BucketForCartSize(len(order.Items)),
	})

	// Product-analytics signals (fire-and-forget). Admin-placed orders have no
	// visitor context (no CloudFront headers), so all attribution is "unknown".
	recordPurchaseAnalytics(ctx, s.customerRepo, order, purchaseAttribution{
		country:   labelUnknown,
		city:      labelUnknown,
		device:    labelUnknown,
		utmSource: labelUnknown,
	})

	// Reserve inventory — track failures for visibility.
	//
	// inventory_mutation_failed metering rule: every swallowed inventory
	// failure is metered, and only those. A site that instead propagates its
	// error to the caller (e.g. checkout_service.go's initial ReserveStock
	// loop) is deliberately NOT metered here — the caller already sees the
	// failure via the returned error, so metering it too would double-count.
	// The reason values are named in constants.go; each swallowed-failure call
	// site emits exactly one of them.
	// Aggregated and all-or-nothing, like the commit and release paths. One
	// product can appear on two lines here, and reserving per line let the
	// order-scoped guard dedup the second away — the order then held less than it
	// sold. All-or-nothing also means a failure leaves no partial reservation to
	// reconcile.
	if reserveErr := s.inventoryRepo.ReserveOrderStock(ctx, order.ID, orderQuantities(order.Items)); reserveErr != nil {
		slog.ErrorContext(ctx, "Failed to reserve stock",
			"order_id", order.ID, "error", reserveErr)
		metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonReserve})
		slog.WarnContext(ctx, "Order created with no inventory reservation", "order_id", order.ID)
	}

	slog.InfoContext(ctx, "Created order", "order_id", order.ID)
	span.End()
	return order, nil
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
		// CancelledAt mirrors CancelOrder, so tracking_handler's timeline
		// (which gates the "cancelled" entry on CancelledAt != nil) shows a
		// cancellation regardless of which path set it. Reuses the UpdatedAt
		// timestamp above rather than calling time.Now() again.
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

// orderQuantities aggregates an order's lines by product. The admin Create path
// takes items straight from the request, so one product can appear on two lines.
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
		// Goods have left the warehouse: convert the reservations into a
		// dispatch. available_qty is unaffected — these units were already
		// unavailable while reserved.
		if commitErr := s.inventoryRepo.CommitOrderStock(ctx, order.ID, orderQuantities(order.Items)); commitErr != nil {
			slog.ErrorContext(ctx, "Failed to commit stock", keyOrderID, order.ID, "error", commitErr)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonCommit})
		}
	case domain.OrderStatusCancelled:
		if releaseErr := s.inventoryRepo.ReleaseOrderStock(ctx, order.ID, orderQuantities(order.Items)); releaseErr != nil {
			slog.ErrorContext(ctx, "Failed to release stock", keyOrderID, order.ID, "error", releaseErr)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: releaseFailureReason(releaseErr)})
		}
		// order_cancelled — status-transition path, no admin reason text.
		metrics.Record(ctx, "order_cancelled", metrics.L{
			metrics.LabelReason:  "status_update",
			metrics.LabelGateway: gatewayPhonePe,
		})
	case domain.OrderStatusReturned:
		// Goods are back. The repository restocks what the order actually
		// committed, not what it ordered: RETURNED is reachable from SHIPPED,
		// but the commit at dispatch is best-effort, so a line that failed to
		// commit was never decremented and must not be added back.
		if restockErr := s.inventoryRepo.RestockOrderStock(ctx, order.ID); restockErr != nil {
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

	// Cancellable up to dispatch. Mirrors validTransitions, which allows
	// PENDING/CONFIRMED/PROCESSING -> CANCELLED; the two paths previously
	// disagreed about PROCESSING.
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
		metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: releaseFailureReason(releaseErr)})
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

// releaseFailureReason maps a ReleaseStock failure to the metric reason label
// for inventory_mutation_failed. A release that fails with insufficient-stock
// means the reservation was already zeroed out by an earlier release — most
// commonly HandlePaymentFailure's rollback (payment_service.go) or the
// checkout rollback (checkout_service.go), both of which leave the order in
// PENDING, still cancellable by customer or admin. That later cancel finding
// nothing left to release is benign and expected, not a leak, so it's
// counted separately as "release_unreserved" rather than "release" — see the
// runbook's "Ongoing drift check" section for why on-call should not treat it
// as a page-worthy signal. Any other error keeps the "release" reason, which
// does indicate a real problem.
func releaseFailureReason(err error) string {
	if appErr, ok := errors.AsAppError(err); ok && appErr.Code == errors.ErrCodeInsufficientStock {
		return reasonReleaseUnreserved
	}
	return reasonRelease
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

// validTransitions defines allowed order status transitions.
// CONFIRMED may go straight to SHIPPED: fulfillment is manual, and forcing a
// stop at PROCESSING made shipping a single order three separate updates.
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
