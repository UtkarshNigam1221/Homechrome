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
	pricingService domain.PricingService
}

// NewOrderService creates a new OrderService
func NewOrderService(
	orderRepo domain.OrderRepository,
	customerRepo domain.CustomerRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	pricingService domain.PricingService,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		priceQuoteRepo: priceQuoteRepo,
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

	// Reserve inventory — track failures for visibility
	var reservationFailures []string
	for _, item := range items {
		_, err := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity, order.ID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to reserve stock",
				keyProductID, item.ProductID,
				"order_id", order.ID,
				"quantity", item.Quantity,
				"error", err,
			)
			reservationFailures = append(reservationFailures, item.ProductID)
		}
	}

	if len(reservationFailures) > 0 {
		slog.WarnContext(ctx, "Order created with inventory reservation failures",
			"order_id", order.ID,
			"failed_products", reservationFailures,
		)
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
	order.UpdatedAt = time.Now()

	// Handle status-specific logic before persisting
	switch status {
	case domain.OrderStatusShipped:
		// Update shipped date
		now := time.Now()
		order.ShippedAt = &now
	case domain.OrderStatusDelivered:
		now := time.Now()
		order.DeliveredAt = &now
		// Release reserved stock (it's now sold)
		// In real app, this would decrement actual stock
	case domain.OrderStatusCancelled:
		// Release reserved stock
		for _, item := range order.Items {
			if _, releaseErr := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, order.ID); releaseErr != nil {
				slog.ErrorContext(ctx, "Failed to release stock", keyProductID, item.ProductID, "error", releaseErr)
			}
		}
		// order_cancelled — status-transition path, no admin reason text.
		metrics.Record(ctx, "order_cancelled", metrics.L{
			metrics.LabelReason:  "status_update",
			metrics.LabelGateway: gatewayPhonePe,
		})
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		span.EndWithError(err)
		return err
	}

	slog.InfoContext(ctx, "Updated order status", "order_id", id, "status", status)
	span.End()
	return nil
}

// AddNote adds a note to an order
func (s *OrderService) AddNote(ctx context.Context, id string, note string, isInternal bool, createdBy string) error {
	orderNote := domain.OrderNote{
		ID:        uuid.New().String()[:8],
		Note:      note,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
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

	// Can only cancel pending or confirmed orders
	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusConfirmed {
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

	// Release reserved stock
	for _, item := range order.Items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, order.ID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to release stock", keyProductID, item.ProductID, "error", err)
		}
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

// RefundOrder initiates a refund for an order
func (s *OrderService) RefundOrder(ctx context.Context, id string, amount int64, reason string, updatedBy string) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate refund amount
	if amount > order.TotalAmount {
		return errors.Validation("Refund amount cannot exceed order total")
	}

	// Update payment status
	order.PaymentStatus = domain.PaymentStatusRefunded
	order.UpdatedBy = updatedBy

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// TODO: Integrate with payment gateway for actual refund

	slog.InfoContext(ctx, "Initiated refund", "order_id", id, "amount", amount)
	return nil
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
