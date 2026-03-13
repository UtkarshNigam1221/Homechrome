// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// OrderService implements domain.OrderService
type OrderService struct {
	orderRepo      domain.OrderRepository
	customerRepo   domain.CustomerRepository
	productRepo    domain.ProductRepository
	inventoryRepo  domain.InventoryRepository
	priceQuoteRepo domain.PriceQuoteRepository
	pricingService domain.PricingService
	logger         *logger.Logger
}

// NewOrderService creates a new OrderService
func NewOrderService(
	orderRepo domain.OrderRepository,
	customerRepo domain.CustomerRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	pricingService domain.PricingService,
	logger *logger.Logger,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		priceQuoteRepo: priceQuoteRepo,
		pricingService: pricingService,
		logger:         logger,
	}
}

// Create creates a new order
func (s *OrderService) Create(ctx context.Context, req domain.CreateOrderRequest, createdBy string) (*domain.Order, error) {
	// Validate customer
	customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeNotFound, "Customer not found")
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
			return nil, errors.New(errors.ErrCodeNotFound, fmt.Sprintf("Product not found: %s", itemInput.ProductID))
		}

		var unitPrice int64
		var priceQuoteID *string

		// Calculate price
		if itemInput.QuoteID != nil {
			// Use existing quote
			var quote *domain.PriceQuote
			quote, err = s.pricingService.GetQuote(ctx, *itemInput.QuoteID)
			if err != nil {
				return nil, errors.New(errors.ErrCodeQuoteExpired, fmt.Sprintf("Invalid or expired quote for item %d", i+1))
			}
			unitPrice = quote.CalculatedPrice / int64(quote.Quantity)
			priceQuoteID = itemInput.QuoteID

			// Mark quote as used
			_ = s.priceQuoteRepo.MarkAsUsed(ctx, *itemInput.QuoteID, orderNumber)
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
				return nil, errors.Wrap(err, fmt.Sprintf("Failed to calculate price for item %d", i+1))
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
			return nil, errors.New(errors.ErrCodeNotFound, fmt.Sprintf("Inventory not found for product: %s", itemInput.ProductID))
		}

		if inventory.AvailableQty < itemInput.Quantity {
			return nil, errors.New(errors.ErrCodeInsufficientStock, fmt.Sprintf("Insufficient stock for product: %s (available: %d, requested: %d)", product.Name, inventory.AvailableQty, itemInput.Quantity))
		}

		item := domain.OrderItem{
			ID:          fmt.Sprintf("item_%d", i+1),
			ProductID:   itemInput.ProductID,
			ProductSKU:  product.SKU,
			ProductName: product.Name,
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
		Currency:        "INR",
		ShippingAddress: &req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		CouponCode:      req.CouponCode,
	}
	order.CreatedBy = createdBy

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Reserve inventory
	for _, item := range items {
		_, err := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity, order.ID)
		if err != nil {
			s.logger.WithContext(ctx).WithError(err).Errorf("Failed to reserve stock for product %s", item.ProductID)
			// Continue - order is created but stock reservation failed
		}
	}

	s.logger.WithContext(ctx).Infof("Created order: %s", order.ID)
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

	// Get item details
	for _, item := range order.Items {
		itemDetail := domain.OrderItemDetails{
			OrderItem:   item,
			ProductName: item.ProductName,
			ProductSKU:  item.ProductSKU,
		}

		// Get product images
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err == nil {
			itemDetail.ProductImages = product.Images
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
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate status transition
	if !isValidStatusTransition(order.Status, status) {
		return errors.BadRequest(fmt.Sprintf("Cannot transition from %s to %s", order.Status, status))
	}

	// Update status
	order.Status = status
	order.UpdatedBy = updatedBy
	order.UpdatedAt = time.Now()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// Handle status-specific logic
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
				s.logger.WithContext(ctx).WithError(releaseErr).Errorf("Failed to release stock for product %s", item.ProductID)
			}
		}
	}

	s.logger.WithContext(ctx).Infof("Updated order status: %s -> %s", id, status)
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

	s.logger.WithContext(ctx).Infof("Added note to order: %s", id)
	return nil
}

// UpdateTracking updates tracking information
func (s *OrderService) UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, updatedBy string) error {
	if err := s.orderRepo.UpdateTracking(ctx, id, trackingNumber, carrier); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Updated tracking for order: %s", id)
	return nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(ctx context.Context, id string, reason string, updatedBy string) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Can only cancel pending or confirmed orders
	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusConfirmed {
		return errors.BadRequest("Order cannot be canceled in current status")
	}

	// Update status
	order.Status = domain.OrderStatusCancelled
	now := time.Now()
	order.CancelledAt = &now
	order.UpdatedBy = updatedBy

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// Release reserved stock
	for _, item := range order.Items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, order.ID)
		if err != nil {
			s.logger.WithContext(ctx).WithError(err).Errorf("Failed to release stock for product %s", item.ProductID)
		}
	}

	s.logger.WithContext(ctx).Infof("Canceled order: %s", id)
	return nil
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

	s.logger.WithContext(ctx).Infof("Initiated refund for order: %s, amount: %d", id, amount)
	return nil
}

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("HL%s%s", now.Format("20060102"), uuid.New().String()[:6])
}

// isValidStatusTransition checks if a status transition is valid
func isValidStatusTransition(from, to domain.OrderStatus) bool {
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPending:    {domain.OrderStatusConfirmed, domain.OrderStatusCancelled},
		domain.OrderStatusConfirmed:  {domain.OrderStatusProcessing, domain.OrderStatusCancelled},
		domain.OrderStatusProcessing: {domain.OrderStatusShipped, domain.OrderStatusCancelled},
		domain.OrderStatusShipped:    {domain.OrderStatusDelivered, domain.OrderStatusReturned},
		domain.OrderStatusDelivered:  {domain.OrderStatusReturned},
	}

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
