package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/metrics"
)

// purchaseAttribution carries the visitor-attribution labels applied to the
type purchaseAttribution struct {
	country   string
	city      string
	device    string
	utmSource string
}

func recordPurchaseAnalytics(ctx context.Context, customerRepo domain.CustomerRepository, order *domain.Order, attr purchaseAttribution) {
	// Per-line product purchase revenue, net of the line's share of the discount.
	// orders_value books order.TotalAmount, which is already net, so summing gross line
	// prices here made the two disagree by the discount on every couponed order — the
	// dashboard would show more product revenue than the business took. They used to
	// agree only because every discount was zero.
	//
	// Not gated on order.DiscountAllocated: an order without an allocation has zero on
	// every line, so the subtraction is the identity there. A gate would add a branch
	// that cannot change the result today, and would over-report if some future writer
	// set line discounts without the flag.
	for _, line := range order.Items {
		catID := line.CategoryID
		if catID == "" {
			catID = labelUnknown
		}
		metrics.RecordSum(ctx, "product_purchased", line.TotalPrice-line.DiscountAmount, metrics.L{
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
