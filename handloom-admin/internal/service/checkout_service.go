package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/telemetry"
)

// CheckoutService implements domain.CheckoutService
type CheckoutService struct {
	cartService    domain.CartService
	orderRepo      domain.OrderRepository
	paymentService domain.PaymentService
	inventoryRepo  domain.InventoryRepository
	customerRepo   domain.CustomerRepository
}

// NewCheckoutService creates a new CheckoutService
func NewCheckoutService(
	cartService domain.CartService,
	orderRepo domain.OrderRepository,
	paymentService domain.PaymentService,
	inventoryRepo domain.InventoryRepository,
	customerRepo domain.CustomerRepository,
) *CheckoutService {
	return &CheckoutService{
		cartService:    cartService,
		orderRepo:      orderRepo,
		paymentService: paymentService,
		inventoryRepo:  inventoryRepo,
		customerRepo:   customerRepo,
	}
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

	// Generated up front so the reservation ledger rows carry the order ID and
	// can be joined against their matching release.
	orderID := uuid.New().String()

	// 5. Reserve inventory, aggregated and all-or-nothing like the admin path. A
	// failure leaves nothing reserved, so there is no partial rollback to unwind.
	if reserveErr := s.inventoryRepo.ReserveOrderStock(ctx, orderID, cartQuantities(cart.Items)); reserveErr != nil {
		slog.ErrorContext(ctx, "Failed to reserve stock", keyOrderID, orderID, "error", reserveErr)
		stockErr := errors.Wrap(reserveErr, "Failed to reserve stock for the cart")
		span.EndWithError(stockErr)
		return nil, stockErr
	}

	// 6. Build order items from cart items
	orderItems := cartItemsToOrderItems(cart.Items)

	// 7. Calculate totals. Shipping is free at checkout — deliveries are
	// scheduled manually, so no courier rate is charged.
	subtotal := cart.Cart.Subtotal
	var discountAmount int64
	var taxAmount int64
	var shippingAmount int64

	totalAmount := subtotal - discountAmount + taxAmount + shippingAmount

	// 8. Generate order number
	orderNumber := generateOrderNumber()

	// 9. Create order
	now := time.Now()
	order := &domain.Order{
		ID:              orderID,
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
		s.releaseReservedItems(ctx, orderID, cart.Items)
		slog.ErrorContext(ctx, "Failed to create order", "error", err)
		createErr := errors.Wrap(err, "Failed to create order")
		span.EndWithError(createErr)
		return nil, createErr
	}

	// Order placed — record business KPI. A count only: the order has reserved
	// stock and nothing more, so no money is booked here.
	metrics.Record(ctx, "orders_placed", metrics.L{metrics.LabelCountry: country, metrics.LabelCity: city})
	metrics.Record(ctx, "cart_size", metrics.L{
		metrics.LabelCountry: country,
		metrics.LabelBucket:  metrics.BucketForCartSize(order.ItemCount),
	})

	// orders_value, per-product counts, coupon-redeemed, first-purchase and order
	// geomap fire in HandlePaymentSuccess — emitting here inflates KPIs on failed
	// payments, and orders_value is what the dashboard reports as revenue.

	// 11. Initiate payment
	paymentResp, err := s.paymentService.InitiatePayment(ctx, domain.InitiatePaymentRequest{
		OrderID:    order.ID,
		CustomerID: customer.ID,
		Amount:     order.TotalAmount,
		Phone:      customer.Phone,
	})
	if err != nil {
		s.releaseReservedItems(ctx, orderID, cart.Items)
		slog.ErrorContext(ctx, "Failed to initiate payment", "error", err)
		payErr := errors.Wrap(err, "Failed to initiate payment")
		span.EndWithError(payErr)
		return nil, payErr
	}

	// 12. Cart is NOT cleared here but after payment success, so it survives a
	// failed payment and the user can retry.

	// Funnel counter: checkout.initiated. cart_to_checkout + geomap dropped per the
	// metrics-PG migration (covered by cart_to_payment + country/city labels).
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

// cartQuantities aggregates cart lines by product, mirroring orderQuantities.
func cartQuantities(items []domain.CartItem) map[string]int {
	quantities := make(map[string]int, len(items))
	for _, item := range items {
		quantities[item.ProductID] += item.Quantity
	}
	return quantities
}

// releaseReservedItems releases inventory for items that were previously reserved
func (s *CheckoutService) releaseReservedItems(ctx context.Context, orderID string, items []domain.CartItem) {
	if err := s.inventoryRepo.ReleaseOrderStock(ctx, orderID, cartQuantities(items)); err != nil {
		slog.ErrorContext(ctx, "Failed to release reserved stock", keyOrderID, orderID, "error", err)
		metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: reasonRelease})
	}
}

// checkoutUTMSource reads first-touch utm_source from context to denormalise onto
// the order; the PhonePe webhook has no browser headers and reads it back.
func checkoutUTMSource(ctx context.Context) string {
	src, _, _ := middleware.GetUTM(ctx)
	return src
}

// Ensure interface compliance
var _ domain.CheckoutService = (*CheckoutService)(nil)
