package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/telemetry"
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
) *CheckoutService {
	return &CheckoutService{
		cartService:     cartService,
		orderRepo:       orderRepo,
		paymentService:  paymentService,
		shippingService: shippingService,
		inventoryRepo:   inventoryRepo,
		customerRepo:    customerRepo,
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
	ctx, span := telemetry.StartServiceSpan(ctx, "checkout", "initiate")
	span.SetAttribute("entity.type", "checkout")

	// Resolve geo context once; used for metrics + persisted on the order.
	country := middleware.GetCountry(ctx)
	city := middleware.GetCity(ctx)

	// 1. Get customer
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get customer during checkout", "error", err)
		span.EndWithError(err)
		return nil, err
	}

	// 2. Find shipping address by ID
	shippingAddr, err := findShippingAddress(customer, req.ShippingAddressID)
	if err != nil {
		span.EndWithError(err)
		return nil, err
	}

	// 3. Get cart
	cart, err := s.cartService.GetCart(ctx, customerID, false)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get cart during checkout", "error", err)
		span.EndWithError(err)
		return nil, err
	}

	// 4. Validate cart is not empty
	if len(cart.Items) == 0 {
		emptyErr := errors.BadRequest("Cart is empty")
		span.EndWithError(emptyErr)
		return nil, emptyErr
	}

	// 5. Reserve inventory for each item
	reservedItems := make([]domain.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		_, reserveErr := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity, "checkout")
		if reserveErr != nil {
			// Rollback: release all previously reserved items
			s.releaseReservedItems(ctx, reservedItems)
			slog.ErrorContext(ctx, "Failed to reserve stock", keyProductID, item.ProductID, "error", reserveErr)
			stockErr := errors.Wrap(reserveErr, fmt.Sprintf("Failed to reserve stock for product %s", item.ProductID))
			span.EndWithError(stockErr)
			return nil, stockErr
		}
		reservedItems = append(reservedItems, item)
	}

	// 6. Build order items from cart items
	orderItems := cartItemsToOrderItems(cart.Items)

	// 7. Calculate totals
	subtotal := cart.Cart.Subtotal
	var discountAmount int64
	var taxAmount int64
	shippingAmount := s.resolveShippingAmount(ctx, shippingAddr, req.CourierID)

	totalAmount := subtotal - discountAmount + taxAmount + shippingAmount

	// shipping_cost_shown: emitted once the price is computed so we can
	// dashboard shipping revenue/cost by country. Fires even when shippingAmount==0.
	metrics.Record(ctx, "shipping_cost_shown", metrics.L{
		metrics.LabelCountry: country,
	})
	if shippingAmount > 0 {
		metrics.RecordSum(ctx, "shipping_cost_shown", shippingAmount, metrics.L{
			metrics.LabelCountry: country,
		})
	}

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
		Currency:        defaultCurrency,
		Status:          domain.OrderStatusPending,
		PaymentStatus:   domain.PaymentStatusPending,
		ShippingAddress: &shippingAddr,
		Country:         country,
		City:            city,
		DeviceType:      middleware.GetDeviceType(ctx),
		UTMSource:       checkoutUTMSource(ctx),
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
		createErr := errors.Wrap(err, "Failed to create order")
		span.EndWithError(createErr)
		return nil, createErr
	}

	// Order placed — record business KPI.
	metrics.Record(ctx, "orders_placed", metrics.L{metrics.LabelCountry: country, metrics.LabelCity: city})
	metrics.RecordSum(ctx, "orders_value", order.TotalAmount, metrics.L{
		metrics.LabelCountry: country, metrics.LabelCity: city, metrics.LabelGateway: gatewayPhonePe,
	})
	metrics.Record(ctx, "cart_size", metrics.L{
		metrics.LabelCountry: country,
		metrics.LabelBucket:  metrics.BucketForCartSize(order.ItemCount),
	})

	// NOTE: Per-product purchase counts, coupon-redeemed, first-purchase, and
	// order geomap are emitted in PaymentService.HandlePaymentSuccess AFTER
	// payment is confirmed — not here. Emitting them before payment completes
	// would inflate KPIs for failed-payment orders.

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
		payErr := errors.Wrap(err, "Failed to initiate payment")
		span.EndWithError(payErr)
		return nil, payErr
	}

	// 12. Cart is NOT cleared here — it's cleared after payment success
	// in PaymentService.HandlePaymentSuccess. This ensures the cart
	// is preserved if payment fails, so the user can retry.

	// Funnel counter: checkout.initiated. cart_to_checkout duration + geomap
	// were dropped per metrics-PG migration plan (covered by cart_to_payment
	// + country/city labels on funnel metrics).
	metrics.Record(ctx, "checkout_initiated", metrics.L{
		metrics.LabelCountry:    country,
		metrics.LabelCity:       city,
		metrics.LabelDeviceType: middleware.GetDeviceType(ctx),
	})

	span.SetAttribute("entity.id", order.ID)
	span.SetAttribute("order.item_count", len(orderItems))
	slog.InfoContext(ctx, "Checkout completed", "order_number", order.OrderNumber, "customer_id", customerID)
	span.End()

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

// findShippingAddress returns the customer address matching id, or NotFound.
func findShippingAddress(customer *domain.Customer, id string) (domain.Address, error) {
	for _, addr := range customer.Addresses {
		if addr.ID == id {
			return addr, nil
		}
	}
	return domain.Address{}, errors.NotFound("Shipping address")
}

// cartItemsToOrderItems maps cart items to order items, assigning a fresh ID to each.
func cartItemsToOrderItems(items []domain.CartItem) []domain.OrderItem {
	orderItems := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
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
	return orderItems
}

// resolveShippingAmount returns the shipping cost for the destination, selecting
// the requested courier or defaulting to the first available. Returns 0 when the
// serviceability check fails or the pincode isn't serviceable — checkout proceeds
// with free shipping rather than blocking the order.
func (s *CheckoutService) resolveShippingAmount(ctx context.Context, addr domain.Address, courierID *int) int64 {
	serviceResult, err := s.shippingService.CheckServiceability(ctx, s.pickupPincode, addr.PostalCode, defaultWeightGrams)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check serviceability for shipping cost", "error", err)
		return 0
	}
	if !serviceResult.Serviceable || len(serviceResult.Couriers) == 0 {
		return 0
	}
	if courierID != nil {
		for _, c := range serviceResult.Couriers {
			if c.ID == *courierID {
				return c.Rate
			}
		}
	}
	return serviceResult.Couriers[0].Rate
}

// releaseReservedItems releases inventory for items that were previously reserved
func (s *CheckoutService) releaseReservedItems(ctx context.Context, items []domain.CartItem) {
	for _, item := range items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, "checkout")
		if err != nil {
			slog.ErrorContext(ctx, "Failed to release reserved stock", keyProductID, item.ProductID, "error", err)
			// Continue releasing other items even if one fails
		}
	}
}

// checkoutUTMSource returns the visitor's first-touch utm_source from
// context for denormalising onto the order. PhonePe webhook ctx is
// server-to-server (no browser headers), so payment_completed +
// customer_first_purchase have to read it back from order.UTMSource.
func checkoutUTMSource(ctx context.Context) string {
	src, _, _ := middleware.GetUTM(ctx)
	return src
}

// Ensure interface compliance
var _ domain.CheckoutService = (*CheckoutService)(nil)
