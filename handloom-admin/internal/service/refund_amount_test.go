package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// order builds an order whose money adds up: subtotal - discount + tax + shipping.
func refundTestOrder(discount, tax, shipping int64, items ...domain.OrderItem) *domain.Order {
	var subtotal int64
	for i := range items {
		items[i].TotalPrice = items[i].UnitPrice * int64(items[i].Quantity)
		subtotal += items[i].TotalPrice
	}
	return &domain.Order{
		ID:             "order_1",
		Items:          items,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TaxAmount:      tax,
		ShippingAmount: shipping,
		TotalAmount:    subtotal - discount + tax + shipping,
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
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 0), line("b", 5000, 1, 0))

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
		order := refundTestOrder(0, 0, 5000, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(10000), got.Total, "no shipping while a unit remains")
	})

	t.Run("the refund that clears the order returns the shipping too", func(t *testing.T) {
		order := refundTestOrder(0, 0, 5000, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, nil, 0)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
		require.Equal(t, order.TotalAmount, got.Total)
	})

	// clearsOrder asks what is left on lines this refund does not name, and a claim
	// counts there too. Reading only RefundedQuantity would call the order unfinished
	// while a refund covering the rest is still in flight — so the last refund would
	// never earn the shipping and the order would never reach fully refunded.
	t.Run("a claim on another line can make this refund the one that clears the order", func(t *testing.T) {
		order := refundTestOrder(0, 0, 5000, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

		// Line b is spoken for by a refund still in flight, so refunding a clears it.
		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, map[string]int{"b": 1}, 20000)

		require.NoError(t, err)
		require.True(t, got.IsFinal, "nothing is left unrefunded once b's claim counts")
		require.Equal(t, int64(5000), got.Shipping, "so this refund earns the shipping back")
		require.Equal(t, order.TotalAmount-20000, got.Total)
	})

	t.Run("tax is prorated the same way as the discount", func(t *testing.T) {
		// subtotal 30000, tax 3000 → a 10000 line carries 1000 of it, added not subtracted.
		order := refundTestOrder(0, 3000, 0, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(11000), got.Total)
		require.Equal(t, int64(1000), got.Tax)
	})

	// Half-up, not truncation. A line carrying 1500 of a 3000 discount across two
	// equal lines rounds away from zero, and truncating would lose the paise.
	t.Run("proration rounds half up", func(t *testing.T) {
		// subtotal 3, discount 1 → each line's share is 1/3, and 0.333 rounds to 0;
		// with subtotal 2 the share is 1/2, which half-up takes to 1 and truncation to 0.
		order := refundTestOrder(1, 0, 0, line("a", 1, 1, 0), line("b", 1, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.Equal(t, int64(1), got.Discount, "half of 1 rounds up to 1, truncation would give 0")
		require.Equal(t, int64(0), got.Total)
	})

	// The four terms are what a screen shows; they have to reconcile to the number
	// underneath the Refund button.
	t.Run("the breakdown adds up to the total", func(t *testing.T) {
		order := refundTestOrder(3000, 7000, 5000, line("a", 10000, 2, 0), line("b", 20000, 1, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2}, {OrderItemID: "b", Quantity: 1},
		}, nil, 0)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
		require.Equal(t, got.Total, got.LineValue-got.Discount+got.Tax+got.Shipping)
		require.Equal(t, int64(5000), got.Shipping, "the clearing refund earns the shipping back")
	})

	t.Run("shipping is zero until the refund that clears the order", func(t *testing.T) {
		order := refundTestOrder(0, 0, 5000, line("a", 10000, 2, 0))

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

	t.Run("discount is prorated by the line's share of the subtotal", func(t *testing.T) {
		// subtotal 30000, discount 3000 → a 10000 line carries 1000 of it.
		order := refundTestOrder(3000, 0, 0, line("a", 10000, 1, 0), line("b", 20000, 1, 0))

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
		order := refundTestOrder(1, 0, 0, line("a", 1, 1, 0), line("b", 1, 1, 0), line("c", 1, 1, 0))

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
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 1))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, nil, 10000)

		require.Error(t, err)
	})

	t.Run("allows exactly what the line has left", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 1))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
	})

	t.Run("rejects a line the order does not have", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 1, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "ghost", Quantity: 1},
		}, nil, 0)

		require.Error(t, err)
	})

	t.Run("rejects the same line twice in one request", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 5, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2}, {OrderItemID: "a", Quantity: 2},
		}, nil, 0)

		require.Error(t, err, "one entry per line, or the remainder check is meaningless")
	})

	t.Run("rejects a request with no lines", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 1, 0))
		_, err := deriveRefundAmount(order, nil, nil, 0)
		require.Error(t, err)
	})

	t.Run("carries the line detail through for the record", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 0))

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

// A refund is PENDING from creation until the provider's webhook lands. Bounding
// the next one on settled refunds alone let the same units go back twice — real
// money out for goods refunded once.
func TestDeriveRefundAmount_CountsClaimedUnits(t *testing.T) {
	t.Run("refuses units an unsettled refund already claimed", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 0))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, map[string]int{"a": 1}, 10000)

		require.Error(t, err, "one unit is already spoken for")
	})

	t.Run("allows what is left once the claim is counted", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 0))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, map[string]int{"a": 1}, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal, "the claimed unit plus this one clears the order")
	})

	// The claim is the authority, not the order's own counter: RefundedQuantity is
	// written at settlement and can lag a refund that is already in flight.
	t.Run("prefers the claim over the order's settled counter", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 3, 1))

		_, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 2},
		}, map[string]int{"a": 2}, 20000)

		require.Error(t, err, "only one unit is unclaimed")
	})

	t.Run("falls back to the order's counter when nothing is claimed", func(t *testing.T) {
		order := refundTestOrder(0, 0, 0, line("a", 10000, 2, 1))

		got, err := deriveRefundAmount(order, []domain.CreateRefundItemRequest{
			{OrderItemID: "a", Quantity: 1},
		}, nil, 10000)

		require.NoError(t, err)
		require.True(t, got.IsFinal)
	})
}
