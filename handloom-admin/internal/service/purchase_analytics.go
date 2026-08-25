package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/metrics"
)

// purchaseAttribution carries the visitor-attribution labels applied to the
// first/repeat-purchase metrics, filled from fields the order denormalised at
// checkout-initiate time.
type purchaseAttribution struct {
	country   string
	city      string
	device    string
	utmSource string
}

// recordPurchaseAnalytics emits the product-purchase KPI signals for the store
// payment-success webhook: per-line product_purchased sums, coupon_redeemed,
// and first-vs-repeat detection via an atomic order-count increment.
// Fire-and-forget — failures are logged, never returned. Channel-specific
// signals (payment_completed, durations) stay with the caller.
func recordPurchaseAnalytics(ctx context.Context, customerRepo domain.CustomerRepository, order *domain.Order, attr purchaseAttribution) {
	// Per-line product purchase counts. revenuePaise = line.TotalPrice.
	for _, line := range order.Items {
		catID := line.CategoryID
		if catID == "" {
			catID = labelUnknown
		}
		metrics.RecordSum(ctx, "product_purchased", line.TotalPrice, metrics.L{
			keyProductID:            line.ProductID,
			metrics.LabelCategoryID: catID,
		})
	}

	// Coupon redemption. DiscountAmount is the total order-level discount.
	if order.CouponCode != nil && *order.CouponCode != "" {
		code := strings.ToUpper(strings.TrimSpace(*order.CouponCode))
		if code != "" {
			metrics.RecordSum(ctx, "coupon_redeemed", order.DiscountAmount, metrics.L{
				metrics.LabelCouponCode: code,
			})
		}
	}

	// First-purchase detection — atomic increment returns the new count. Two
	// concurrent orders for the same customer get newCount==1 and ==2; only the
	// first fires customer_first_purchase, repeat_purchase fires when > 1.
	newCount, err := customerRepo.RecordPurchase(ctx, order.CustomerID, order.TotalAmount)
	if err != nil {
		slog.WarnContext(ctx, "record purchase failed", "customer_id", order.CustomerID, "error", err)
		return
	}
	switch {
	case newCount == 1:
		metrics.Record(ctx, "customer_first_purchase", metrics.L{
			metrics.LabelCountry:    attr.country,
			metrics.LabelCity:       attr.city,
			metrics.LabelDeviceType: attr.device,
			metrics.LabelUTMSource:  attr.utmSource,
		})
	case newCount > 1:
		metrics.Record(ctx, "repeat_purchase", metrics.L{
			metrics.LabelCountry:    attr.country,
			metrics.LabelCity:       attr.city,
			metrics.LabelDeviceType: attr.device,
		})
	}
}
