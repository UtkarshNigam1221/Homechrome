package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/errors"
)

// defaultPickupPincode is the warehouse/pickup location pincode
const defaultPickupPincode = "560001"

// defaultWeightGrams is the default parcel weight used for serviceability checks
const defaultWeightGrams = 500

// CheckoutService implements domain.CheckoutService
type CheckoutService struct {
	cartService     domain.CartService
	orderRepo       domain.OrderRepository
	paymentService  domain.PaymentService
	shippingService domain.ShippingService
	inventoryRepo   domain.InventoryRepository
	customerRepo    domain.CustomerRepository
	publisher       event.EventPublisher
	pickupPincode   string
}

// NewCheckoutService creates a new CheckoutService
func NewCheckoutService(
	cartService domain.CartService,
	orderRepo domain.OrderRepository,
	paymentService domain.PaymentService,
	shippingService domain.ShippingService,
	inventoryRepo domain.InventoryRepository,
	customerRepo domain.CustomerRepository,
	publisher event.EventPublisher,
) *CheckoutService {
	return &CheckoutService{
		cartService:     cartService,
		orderRepo:       orderRepo,
		paymentService:  paymentService,
		shippingService: shippingService,
		inventoryRepo:   inventoryRepo,
		customerRepo:    customerRepo,
		publisher:       publisher,
		pickupPincode:   defaultPickupPincode,
	}
}

// CheckServiceability checks whether delivery is available for a given pincode
func (s *CheckoutService) CheckServiceability(ctx context.Context, customerID, pincode string) (*domain.ServiceabilityResult, error) {
	// Verify the customer exists
	_, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get customer for serviceability check", "error", err)
		return nil, err
	}

	result, err := s.shippingService.CheckServiceability(ctx, s.pickupPincode, pincode, defaultWeightGrams)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check serviceability", "error", err)
		return nil, errors.Wrap(err, "Failed to check serviceability")
	}

	return result, nil
}

// Initiate orchestrates the full checkout flow: reserve inventory, create order,
// initiate payment, and clear the cart.
func (s *CheckoutService) Initiate(ctx context.Context, customerID string, req domain.CheckoutRequest) (*domain.CheckoutResult, error) {
	// 1. Get customer
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get customer during checkout", "error", err)
		return nil, err
	}

	// 2. Find shipping address by ID
	var shippingAddr domain.Address
	addressFound := false
	for _, addr := range customer.Addresses {
		if addr.ID == req.ShippingAddressID {
			shippingAddr = addr
			addressFound = true
			break
		}
	}
	if !addressFound {
		return nil, errors.NotFound("Shipping address")
	}

	// 3. Get cart
	cart, err := s.cartService.GetCart(ctx, customerID, false)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get cart during checkout", "error", err)
		return nil, err
	}

	// 4. Validate cart is not empty
	if len(cart.Items) == 0 {
		return nil, errors.BadRequest("Cart is empty")
	}

	// 5. Reserve inventory for each item
	reservedItems := make([]domain.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		_, reserveErr := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity, "checkout")
		if reserveErr != nil {
			// Rollback: release all previously reserved items
			s.releaseReservedItems(ctx, reservedItems)
			slog.ErrorContext(ctx, "Failed to reserve stock", "product_id", item.ProductID, "error", reserveErr)
			return nil, errors.Wrap(reserveErr, fmt.Sprintf("Failed to reserve stock for product %s", item.ProductID))
		}
		reservedItems = append(reservedItems, item)
	}

	// 6. Build order items from cart items
	orderItems := make([]domain.OrderItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		orderItems = append(orderItems, domain.OrderItem{
			ID:           uuid.New().String(),
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			ProductSKU:   item.ProductSKU,
			ProductImage: item.ProductImage,
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			IsCustomSize: item.IsCustomSize,
			Dimensions:   item.Dimensions,
			QuoteID:      item.QuoteID,
			UnitPrice:    item.UnitPrice,
			Quantity:     item.Quantity,
			TotalPrice:   item.TotalPrice,
		})
	}

	// 7. Calculate totals
	subtotal := cart.Cart.Subtotal
	var discountAmount int64
	var taxAmount int64
	var shippingAmount int64

	// Check serviceability for shipping cost
	serviceResult, err := s.shippingService.CheckServiceability(ctx, s.pickupPincode, shippingAddr.PostalCode, defaultWeightGrams)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check serviceability for shipping cost", "error", err)
		// Proceed with zero shipping if serviceability check fails
	} else if serviceResult.Serviceable && len(serviceResult.Couriers) > 0 {
		// Use selected courier or default to first available
		if req.CourierID != nil {
			for _, c := range serviceResult.Couriers {
				if c.ID == *req.CourierID {
					shippingAmount = c.Rate
					break
				}
			}
		}
		if shippingAmount == 0 {
			shippingAmount = serviceResult.Couriers[0].Rate
		}
	}

	totalAmount := subtotal - discountAmount + taxAmount + shippingAmount

	// 8. Generate order number
	orderNumber := generateOrderNumber()

	// 9. Create order
	now := time.Now()
	order := &domain.Order{
		ID:              uuid.New().String(),
		OrderNumber:     orderNumber,
		CustomerID:      customer.ID,
		CustomerName:    customer.FirstName + " " + customer.LastName,
		CustomerEmail:   customer.Email,
		CustomerPhone:   customer.Phone,
		Items:           orderItems,
		ItemCount:       len(orderItems),
		Subtotal:        subtotal,
		DiscountAmount:  discountAmount,
		TaxAmount:       taxAmount,
		ShippingAmount:  shippingAmount,
		TotalAmount:     totalAmount,
		Currency:        "INR",
		Status:          domain.OrderStatusPending,
		PaymentStatus:   domain.PaymentStatusPending,
		ShippingAddress: &shippingAddr,
		BaseEntity: domain.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	order.SetKeys()

	// 10. Save order (repository also writes the order number index)
	if err = s.orderRepo.Create(ctx, order); err != nil {
		s.releaseReservedItems(ctx, reservedItems)
		slog.ErrorContext(ctx, "Failed to create order", "error", err)
		return nil, errors.Wrap(err, "Failed to create order")
	}

	// 11. Initiate payment
	paymentResp, err := s.paymentService.InitiatePayment(ctx, domain.InitiatePaymentRequest{
		OrderID:    order.ID,
		CustomerID: customer.ID,
		Amount:     order.TotalAmount,
		Phone:      customer.Phone,
	})
	if err != nil {
		s.releaseReservedItems(ctx, reservedItems)
		slog.ErrorContext(ctx, "Failed to initiate payment", "error", err)
		return nil, errors.Wrap(err, "Failed to initiate payment")
	}

	// 12. Clear cart
	if err := s.cartService.ClearCart(ctx, customerID); err != nil {
		// Log but don't fail the checkout - order and payment are already created
		slog.ErrorContext(ctx, "Failed to clear cart after checkout", "error", err)
	}

	// 13. Publish order.created event
	if pubErr := s.publisher.Publish(ctx, event.New(event.OrderCreated, order)); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish order.created event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Checkout completed", "order_number", order.OrderNumber, "customer_id", customerID)

	return &domain.CheckoutResult{
		Order:         order,
		RedirectURL:   paymentResp.RedirectURL,
		MerchantTxnID: paymentResp.MerchantTxnID,
	}, nil
}

// GetPaymentStatus retrieves the current payment status for an order
func (s *CheckoutService) GetPaymentStatus(ctx context.Context, customerID, orderID string) (*domain.PaymentStatusResult, error) {
	// Get the order
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Validate the order belongs to the customer
	if order.CustomerID != customerID {
		return nil, errors.NotFound("Order")
	}

	// Get payment by order ID
	payment, err := s.paymentService.GetByOrderID(ctx, orderID)
	if err != nil {
		// If no payment found, return the order's payment status
		if errors.IsNotFound(err) {
			return &domain.PaymentStatusResult{
				PaymentStatus: order.PaymentStatus,
				Order:         order,
			}, nil
		}
		return nil, err
	}

	return &domain.PaymentStatusResult{
		PaymentStatus: payment.Status,
		Order:         order,
	}, nil
}

// releaseReservedItems releases inventory for items that were previously reserved
func (s *CheckoutService) releaseReservedItems(ctx context.Context, items []domain.CartItem) {
	for _, item := range items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, "checkout")
		if err != nil {
			slog.ErrorContext(ctx, "Failed to release reserved stock", "product_id", item.ProductID, "error", err)
			// Continue releasing other items even if one fails
		}
	}
}

// Ensure interface compliance
var _ domain.CheckoutService = (*CheckoutService)(nil)
