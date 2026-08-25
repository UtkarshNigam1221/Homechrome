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
	couponService  domain.CouponService
}

// NewCheckoutService creates a new CheckoutService
func NewCheckoutService(
	cartService domain.CartService,
	orderRepo domain.OrderRepository,
	paymentService domain.PaymentService,
	inventoryRepo domain.InventoryRepository,
	customerRepo domain.CustomerRepository,
	couponService domain.CouponService,
) *CheckoutService {
	return &CheckoutService{
		cartService:    cartService,
		orderRepo:      orderRepo,
		paymentService: paymentService,
		inventoryRepo:  inventoryRepo,
		customerRepo:   customerRepo,
		couponService:  couponService,
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

	// 7. Calculate totals. Prices are tax-inclusive, so tax is extracted from the total
	// rather than added to it, and shipping is free — deliveries are scheduled manually.
	subtotal := cart.Cart.Subtotal
	discountAmount, couponID, couponCode, couponNotice := s.resolveCoupon(ctx, customerID, subtotal, req.CouponCode)

	var shippingAmount int64
	totalAmount := subtotal - discountAmount + shippingAmount
	taxAmount := extractTax(totalAmount)

	// Allocate the discount onto the lines, which become the source of truth. Refuse
	// rather than write an order whose lines disagree with its total — every later
	// refund reads the gap and rejects itself, so this is the one coupon failure that
	// aborts the checkout instead of pricing around it.
	shares, allocErr := allocateDiscount(orderItems, discountAmount)
	if allocErr != nil {
		s.releaseReservedItems(ctx, orderID, cart.Items)
		slog.ErrorContext(ctx, "Failed to allocate discount onto order lines", "error", allocErr)
		span.EndWithError(allocErr)
		return nil, allocErr
	}
	for i := range orderItems {
		orderItems[i].DiscountAmount = shares[i]
	}

	// 8. Generate order number
	orderNumber := generateOrderNumber()

	// 9. Create order
	now := time.Now()
	order := &domain.Order{
		ID:                orderID,
		OrderNumber:       orderNumber,
		CustomerID:        customer.ID,
		CustomerName:      customer.FirstName + " " + customer.LastName,
		CustomerEmail:     customer.Email,
		CustomerPhone:     customer.Phone,
		Items:             orderItems,
		ItemCount:         len(orderItems),
		Subtotal:          subtotal,
		DiscountAmount:    discountAmount,
		TaxAmount:         taxAmount,
		ShippingAmount:    shippingAmount,
		TotalAmount:       totalAmount,
		Currency:          defaultCurrency,
		CouponID:          couponID,
		CouponCode:        couponCode,
		DiscountAllocated: true,
		Status:            domain.OrderStatusPending,
		PaymentStatus:     domain.PaymentStatusPending,
		ShippingAddress:   &shippingAddr,
		Country:           country,
		City:              city,
		DeviceType:        middleware.GetDeviceType(ctx),
		UTMSource:         checkoutUTMSource(ctx),
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
		CouponNotice:  couponNotice,
	}, nil
}

// resolveCoupon validates an optional coupon code against the server's cart total. A
// coupon problem is a result, never an error: it prices the order without the discount
// and reports why through notice. Only the caller's later allocation step can abort
// the checkout.
func (s *CheckoutService) resolveCoupon(
	ctx context.Context,
	customerID string,
	cartTotal int64,
	code *string,
) (discountAmount int64, couponID, couponCode *string, notice string) {
	if code == nil || *code == "" {
		return 0, nil, nil, ""
	}

	// Phase 1 has no automatic offers, so both context fields take their Phase 1
	// values. Phase 4 subtracts the buy-N-get-M discount from CartTotal and sets
	// HasAutomaticOffer.
	res, err := s.couponService.Validate(ctx, *code, domain.CouponContext{
		CartTotal:         cartTotal,
		CustomerID:        customerID,
		HasAutomaticOffer: false,
	})
	switch {
	case err != nil:
		// A coupon lookup failure must not cost the sale.
		slog.WarnContext(ctx, "Coupon validation failed", "error", err)
		return 0, nil, nil, "We couldn't apply that code."
	case !res.Valid:
		return 0, nil, nil, res.ErrorMessage
	default:
		// Notice is non-empty when the discount was applied but reduced, so the
		// shortfall reaches the customer instead of just the smaller number.
		return res.DiscountAmount, &res.CouponID, &res.Code, res.Notice
	}
}

// PreviewCoupon prices a code against the customer's current cart without placing an
// order. The cart total is read server-side for the same reason checkout re-validates:
// the client never supplies a figure that decides money.
func (s *CheckoutService) PreviewCoupon(ctx context.Context, customerID, code string) (*domain.CouponValidationResult, error) {
	cart, err := s.cartService.GetCart(ctx, customerID, false)
	if err != nil {
		return nil, err
	}

	return s.couponService.Validate(ctx, code, domain.CouponContext{
		CartTotal:         cart.Cart.Subtotal,
		CustomerID:        customerID,
		HasAutomaticOffer: false, // Phase 4 sets this
	})
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
