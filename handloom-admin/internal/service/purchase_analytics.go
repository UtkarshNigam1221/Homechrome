package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/metrics"
)

// purchaseAttribution carries the visitor-attribution labels applied to the
// first/repeat-purchase metrics. The admin order-create path has no visitor
// context and passes all "unknown"; the store payment-success path fills these
// from fields the order denormalised at checkout-initiate time.
type purchaseAttribution struct {
	country   string
	city      string
	device    string
	utmSource string
}

// recordPurchaseAnalytics emits the product-purchase KPI signals shared by the
// admin order-create path and the store payment-success webhook: per-line
// product_purchased sums, coupon_redeemed, and first-vs-repeat detection via an
// atomic order-count increment. Fire-and-forget — failures are logged, never
// returned. Channel-specific signals (payment_completed, durations) stay with
// their caller.
func recordPurchaseAnalytics(ctx context.Context, customerRepo domain.CustomerRepository, order *domain.Order, attr purchaseAttribution) {
	// Per-line product purchase counts. revenuePaise = line.TotalPrice.
	for _, line := range order.Items {
		catID := line.CategoryID
		if catID == "" {
			catID = "unknown"
		}
		metrics.RecordSum(ctx, "product_purchased", line.TotalPrice, metrics.L{
			"product_id":  line.ProductID,
			"category_id": catID,
		})
	}

	// Coupon redemption. DiscountAmount is the total order-level discount.
	if order.CouponCode != nil && *order.CouponCode != "" {
		code := strings.ToUpper(strings.TrimSpace(*order.CouponCode))
		if code != "" {
			metrics.RecordSum(ctx, "coupon_redeemed", order.DiscountAmount, metrics.L{
				"coupon_code": code,
			})
		}
	}

	// First-purchase detection — atomic increment returns the new count. Two
	// concurrent orders for the same customer get newCount==1 and ==2; only the
	// first fires customer_first_purchase, repeat_purchase fires when > 1.
	newCount, err := customerRepo.IncrementOrderCount(ctx, order.CustomerID)
	if err != nil {
		slog.WarnContext(ctx, "increment order count failed", "customer_id", order.CustomerID, "error", err)
		return
	}
	switch {
	case newCount == 1:
		metrics.Record(ctx, "customer_first_purchase", metrics.L{
			"country":     attr.country,
			"city":        attr.city,
			"device_type": attr.device,
			"utm_source":  attr.utmSource,
		})
	case newCount > 1:
		metrics.Record(ctx, "repeat_purchase", metrics.L{
			"country":     attr.country,
			"city":        attr.city,
			"device_type": attr.device,
		})
	}
}
