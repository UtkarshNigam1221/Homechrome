package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// order builds a tax-inclusive order whose money adds up: subtotal - discount +
// shipping. TaxAmount is derived from the total via extractTax, exactly as checkout
// computes it — a figure contained within TotalAmount, never added to it. The
// discount is allocated onto the lines by the real allocator, because that is the
// only shape checkout writes.
func refundTestOrder(discount, shipping int64, items ...domain.OrderItem) *domain.Order {
	var subtotal int64
	for i := range items {
		items[i].TotalPrice = items[i].UnitPrice * int64(items[i].Quantity)
		subtotal += items[i].TotalPrice
	}
	shares, err := allocateDiscount(items, discount)
	if err != nil {
		panic("refundTestOrder: " + err.Error())
	}
	for i := range items {
		items[i].DiscountAmount = shares[i]
	}
	total := subtotal - discount + shipping
	return &domain.Order{
		ID:             "order_1",
		Items:          items,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TaxAmount:      extractTax(total),
		ShippingAmount: shipping,
		TotalAmount:    total,
	}
}

func line(id string, unitPrice int64, quantity, refunded int) domain.OrderItem {
	return domain.OrderItem{
		ID: id, ProductID: "prod_" + id, ProductName: "Item " + id,
		UnitPrice: unitPrice, Quantity: quantity, RefundedQuantity: refunded,
	}
}

func TestDeriveRefundAmount(t *testing.T) {
	t.Run("one line of a multi-line order refunds only its own value", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 0), line("b", 5000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(10000), got.Total)
		require.False(t, got.IsFinal)
		require.Equal(t, int64(10000), got.Items[0].Amount)
	})

	// Shipping stays with the order while any of it still ships.
	t.Run("a partial refund keeps the shipping", func(t *testing.T) {
		order := refundTestOrder(0, 5000, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(10000), got.Total, "no shipping while a unit remains")
	})

	t.Run("the refund that clears the order returns the shipping too", func(t *testing.T) {
		order := refundTestOrder(0, 5000, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, nil, 0)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
		require.Equal(t, order.TotalAmount, got.Total)
	})

	// clearsOrder asks what is left on lines this refund does not name, where a claim
	// counts too — or the last refund never earns the shipping.
	t.Run("a claim on another line can make this refund the one that clears the order", func(t *testing.T) {
		order := refundTestOrder(0, 5000, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

		// Line b is spoken for by a refund still in flight, so refunding a clears it.
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, map[string]int{"b": 1}, 20000)

		require.NoError(t, err)
		require.True(t, got.IsFinal, "nothing is left unrefunded once b's claim counts")
		require.Equal(t, int64(5000), got.Shipping, "so this refund earns the shipping back")
		require.Equal(t, order.TotalAmount-20000, got.Total)
	})

	// Prices are tax-inclusive, so the refund is the line's value less its discount,
	// full stop — GST is already inside that figure, not an addition on top of it.
	t.Run("tax is extracted from the refund, not added on top of it", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(10000), got.Total, "the line's own value, nothing added")
		require.Equal(t, extractTax(10000), got.Tax,
			"the GST contained within the refund, not a proration of order.TaxAmount")
	})

	// Half-up, not truncation. A line carrying 1500 of a 3000 discount across two
	// equal lines rounds away from zero, and truncating would lose the paise.
	t.Run("proration rounds half up", func(t *testing.T) {
		// subtotal 3, discount 1 → each line's share is 1/3, and 0.333 rounds to 0;
		// with subtotal 2 the share is 1/2, which half-up takes to 1 and truncation to 0.
		order := refundTestOrder(1, 0, line("a", 1, 1, 0), line("b", 1, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(1), got.Discount, "half of 1 rounds up to 1, truncation would give 0")
		require.Equal(t, int64(0), got.Total)
	})

	// The terms are what a screen shows; LineValue, Discount and Shipping have to
	// reconcile to the number underneath the Refund button. Tax is informational —
	// contained within LineValue-Discount on a tax-inclusive order, not a fourth addend.
	t.Run("the breakdown adds up to the total", func(t *testing.T) {
		order := refundTestOrder(3000, 5000, line("a", 10000, 2, 0), line("b", 20000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2}, {OrderItemID: "b", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
		require.Equal(t, got.Total, got.LineValue-got.Discount+got.Shipping)
		require.Equal(t, int64(5000), got.Shipping, "the clearing refund earns the shipping back")
		require.Equal(t, order.TotalAmount, got.Total, "a full refund of a discounted order returns exactly what was paid")
	})

	t.Run("shipping is zero until the refund that clears the order", func(t *testing.T) {
		order := refundTestOrder(0, 5000, line("a", 10000, 2, 0))

		partial, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(0), partial.Shipping)

		whole, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(5000), whole.Shipping)
	})

	t.Run("a line refunds its own allocated discount", func(t *testing.T) {
		// subtotal 30000, discount 3000 → the allocator puts 1000 on the 10000 line.
		order := refundTestOrder(3000, 0, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(9000), got.Total)
	})

	// Every refund is whole paise, and the sum of them has to equal the order
	// exactly or the order never reaches fully refunded.
	t.Run("the final refund absorbs the rounding residual", func(t *testing.T) {
		// subtotal 3, discount 1 → each line's prorated share rounds to 0.
		order := refundTestOrder(1, 0, line("a", 1, 1, 0), line("b", 1, 1, 0), line("c", 1, 1, 0))

		first, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)
		require.NoError(t, err)

		order.Items[0].RefundedQuantity = 1
		second, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "b", Quantity: 1}, {OrderItemID: "c", Quantity: 1},
		}, nil, first.Total)
		require.NoError(t, err)

		require.True(t, second.IsFinal)
		require.Equal(t, order.TotalAmount, first.Total+second.Total,
			"the refunds together must equal the order exactly")
	})

	t.Run("rejects more than the line has left", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 1))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, nil, 10000)

		require.Error(t, err)
	})

	t.Run("allows exactly what the line has left", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 1))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
	})

	t.Run("rejects a line the order does not have", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 1, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "ghost", Quantity: 1},
		}, nil, 0)

		require.Error(t, err)
	})

	t.Run("rejects the same line twice in one request", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 5, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2}, {OrderItemID: "a", Quantity: 2},
		}, nil, 0)

		require.Error(t, err, "one entry per line, or the remainder check is meaningless")
	})

	t.Run("rejects a request with no lines", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 1, 0))
		_, err := deriveRefundAmount(order, nil, nil, 0)
		require.Error(t, err)
	})

	t.Run("carries the line detail through for the record", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2, Restock: true},
		}, nil, 0)

		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		require.Equal(t, "prod_a", got.Items[0].ProductID)
		require.Equal(t, "Item a", got.Items[0].ProductName)
		require.Equal(t, 2, got.Items[0].Quantity)
		require.True(t, got.Items[0].Restock)
	})
}

// A refund is PENDING until the webhook lands. Bounding the next on settled refunds
// alone let the same units go back twice.
func TestDeriveRefundAmount_CountsClaimedUnits(t *testing.T) {
	t.Run("refuses units an unsettled refund already claimed", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, map[string]int{"a": 1}, 10000)

		require.Error(t, err, "one unit is already spoken for")
	})

	t.Run("allows what is left once the claim is counted", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, map[string]int{"a": 1}, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal, "the claimed unit plus this one clears the order")
	})

	// The claim is the authority, not the order's own counter: RefundedQuantity is
	// written at settlement and can lag a refund that is already in flight.
	t.Run("prefers the claim over the order's settled counter", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 3, 1))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, map[string]int{"a": 2}, 20000)

		require.Error(t, err, "only one unit is unclaimed")
	})

	t.Run("falls back to the order's counter when nothing is claimed", func(t *testing.T) {
		order := refundTestOrder(0, 0, line("a", 10000, 2, 1))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
	})
}

// Per-line discounts are what buy-N-get-M needs: the discount belongs to the line
// that earned it, not to the order in proportion.
func TestDeriveRefundAmount_PerLineDiscount(t *testing.T) {
	// Mirrors what a discount writer must produce: lines carrying the figure, the
	// order agreeing, and the marker set. Tax-inclusive, like refundTestOrder: TaxAmount
	// is derived from the total via extractTax, not an independent addend.
	perLineOrder := func(shipping int64, items ...domain.OrderItem) *domain.Order {
		var subtotal, discount int64
		for _, it := range items {
			subtotal += it.UnitPrice * int64(it.Quantity)
			discount += it.DiscountAmount
		}
		total := subtotal - discount + shipping
		return &domain.Order{
			ID: "order_1", Items: items,
			Subtotal: subtotal, DiscountAmount: discount,
			TaxAmount: extractTax(total), ShippingAmount: shipping,
			TotalAmount: total,
		}
	}

	// The whole point: line b carries the entire discount, so refunding a must not
	// get a share of it. Proration would have handed a two thirds of it.
	t.Run("a line's discount stays on that line", func(t *testing.T) {
		order := perLineOrder(0,
			domain.OrderItem{ID: "a", ProductID: "pa", UnitPrice: 2000, Quantity: 1},
			domain.OrderItem{ID: "b", ProductID: "pb", UnitPrice: 1000, Quantity: 1, DiscountAmount: 1000},
		)
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(0), got.Discount, "line a was not discounted")
		require.Equal(t, int64(2000), got.Total)
	})

	t.Run("a fully discounted line refunds nothing", func(t *testing.T) {
		order := perLineOrder(0,
			domain.OrderItem{ID: "a", ProductID: "pa", UnitPrice: 2000, Quantity: 1},
			domain.OrderItem{ID: "b", ProductID: "pb", UnitPrice: 1000, Quantity: 1, DiscountAmount: 1000},
		)
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "b", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(0), got.Total, "the free unit cost nothing, so nothing goes back")
	})

	t.Run("a partial refund takes its share of the line's discount", func(t *testing.T) {
		order := perLineOrder(0,
			domain.OrderItem{ID: "a", ProductID: "pa", UnitPrice: 1000, Quantity: 3, DiscountAmount: 1000},
		)
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(333), got.Discount, "one of three units, half-up")
		require.Equal(t, int64(667), got.Total)
	})

	// The branch most likely to leak money, and it had no coverage: the per-line
	// discount alongside shipping and the clearing adjustment, on a tax-inclusive order.
	t.Run("a clearing refund still lands exactly on the order total", func(t *testing.T) {
		for _, order := range []string{"discounted line last", "discounted line first"} {
			t.Run(order, func(t *testing.T) {
				items := []domain.OrderItem{
					{ID: "a", ProductID: "pa", UnitPrice: 2000, Quantity: 1},
					{ID: "b", ProductID: "pb", UnitPrice: 1000, Quantity: 1, DiscountAmount: 400},
				}
				if order == "discounted line first" {
					items[0], items[1] = items[1], items[0]
				}
				o := perLineOrder(500, items...)

				var refunded int64
				for _, it := range o.Items {
					got, err := deriveRefundAmount(o,
						[]domain.CreateRefundItemRequest{{OrderItemID: it.ID, Quantity: 1}},
						map[string]int{items[0].ID: 0, items[1].ID: 0}, refunded)
					require.NoError(t, err)
					refunded += got.Total
					// Mark it settled so the next call sees the remainder.
					for i := range o.Items {
						if o.Items[i].ID == it.ID {
							o.Items[i].RefundedQuantity = o.Items[i].Quantity
						}
					}
				}
				require.Equal(t, o.TotalAmount, refunded,
					"refunds must sum to the order, shipping and residual included")
			})
		}
	})

	// Repeated partial refunds of one multi-unit line, which was untested.
	t.Run("repeated partial refunds of a discounted line stay bounded", func(t *testing.T) {
		order := perLineOrder(0,
			domain.OrderItem{ID: "a", ProductID: "pa", UnitPrice: 1000, Quantity: 3, DiscountAmount: 1000},
			domain.OrderItem{ID: "b", ProductID: "pb", UnitPrice: 500, Quantity: 1},
		)
		var refunded int64
		for i := 0; i < 3; i++ {
			got, err := deriveRefundAmount(order,
				[]domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, refunded)
			require.NoError(t, err)
			refunded += got.Total
			order.Items[0].RefundedQuantity++
		}
		// 667*3 = 2001 against a line worth 3000 less 1000. Rounding per call is
		// off by a paisa; a clearing refund absorbs it, which is why b is left.
		require.Equal(t, int64(2001), refunded)
		require.LessOrEqual(t, refunded, order.TotalAmount)
	})

	// Corrupt stored data must not refund more than the line is worth.
	t.Run("a stored discount outside the line's range is clamped", func(t *testing.T) {
		order := perLineOrder(0,
			domain.OrderItem{ID: "a", ProductID: "pa", UnitPrice: 1000, Quantity: 1, DiscountAmount: -500},
			domain.OrderItem{ID: "b", ProductID: "pb", UnitPrice: 1000, Quantity: 1, DiscountAmount: 500},
		)
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(1000), got.Total, "never above the line's own value")

		order.Items[0].DiscountAmount = 99999
		got, err = deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, int64(0), got.Total, "never below zero")
	})

	// No order-level fallback: an order with no line discounts refunds its full
	// line value, which is what every pre-coupon order carries.
	t.Run("an order with no line discounts refunds the whole line", func(t *testing.T) {
		order := &domain.Order{
			ID: "order_plain",
			Items: []domain.OrderItem{
				{ID: "a", ProductID: "pa", UnitPrice: 2000, Quantity: 1},
				{ID: "b", ProductID: "pb", UnitPrice: 1000, Quantity: 1},
			},
			Subtotal: 3000, TotalAmount: 3000,
		}
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{{OrderItemID: "a", Quantity: 1}}, nil, 0)
		require.NoError(t, err)
		require.Zero(t, got.Discount)
		require.Equal(t, int64(2000), got.Total)
	})
}

// The round trip the design rests on: what allocateDiscount writes is what the
// refund path reads back, with no gap between them.
func TestAllocateThenRefund_RoundTrip(t *testing.T) {
	items := []domain.OrderItem{
		{ID: "a", ProductID: "pa", UnitPrice: 999, Quantity: 1},
		{ID: "b", ProductID: "pb", UnitPrice: 333, Quantity: 2},
		{ID: "c", ProductID: "pc", UnitPrice: 111, Quantity: 3},
	}
	var subtotal int64
	for _, it := range items {
		subtotal += it.UnitPrice * int64(it.Quantity)
	}

	for total := int64(0); total <= subtotal; total += 13 {
		shares, err := allocateDiscount(items, total)
		require.NoError(t, err)

		order := &domain.Order{ID: "o", Items: append([]domain.OrderItem(nil), items...),
			Subtotal: subtotal, DiscountAmount: total,
			TotalAmount: subtotal - total}
		for i := range order.Items {
			order.Items[i].DiscountAmount = shares[i]
		}

		var refunded int64
		for i := range order.Items {
			got, err := deriveRefundAmount(order,
				[]domain.CreateRefundItemRequest{{OrderItemID: order.Items[i].ID, Quantity: order.Items[i].Quantity}},
				nil, refunded)
			require.NoError(t, err, "total=%d line=%s", total, order.Items[i].ID)
			refunded += got.Total
			order.Items[i].RefundedQuantity = order.Items[i].Quantity
		}
		require.Equal(t, order.TotalAmount, refunded, "discount=%d", total)
	}
}
