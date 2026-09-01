package store

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
)

// StoreCoupon is the public face of a coupon. An explicit struct rather than the
// entity: usage counters advertise scarcity, and customer_id is another customer's id.
type StoreCoupon struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Type          string     `json:"type"`
	Value         int64      `json:"value"`
	MinOrderValue int64      `json:"min_order_value"`
	MaxDiscount   int64      `json:"max_discount,omitempty"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"`
}

func toStoreCoupon(c *domain.Coupon) StoreCoupon {
	return StoreCoupon{
		Code:          c.Code,
		Name:          c.Name,
		Description:   c.Description,
		Type:          string(c.Type),
		Value:         c.Value,
		MinOrderValue: c.MinOrderValue,
		MaxDiscount:   c.MaxDiscount,
		ValidUntil:    c.ValidUntil,
	}
}

// ListCoupons handles GET /api/v1/store/catalog/coupons — the public offers banner.
// A read failure is an empty banner, never a broken homepage.
func (h *CatalogHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	coupons, err := h.couponService.ListPublic(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "Public coupon list unavailable", "error", err)
		// Override the route's inherited hour-long cache so a transient failure
		// isn't remembered as "no offers" by browsers and intermediaries.
		w.Header().Set("Cache-Control", "no-store")
		coupons = nil
	}

	out := make([]StoreCoupon, 0, len(coupons))
	for _, c := range coupons {
		out = append(out, toStoreCoupon(c))
	}

	response.Success(w, out)
}
