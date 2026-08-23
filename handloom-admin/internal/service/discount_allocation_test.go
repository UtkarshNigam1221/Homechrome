package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func mustAllocate(t *testing.T, items []domain.OrderItem, total int64) []int64 {
	t.Helper()
	shares, err := allocateDiscount(items, total)
	require.NoError(t, err)
	return shares
}

// discountLines builds order items from alternating unit price and quantity.
func discountLines(unitPriceAndQty ...int64) []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(unitPriceAndQty)/2)
	for i := 0; i+1 < len(unitPriceAndQty); i += 2 {
		items = append(items, domain.OrderItem{
			UnitPrice: unitPriceAndQty[i],
			Quantity:  int(unitPriceAndQty[i+1]),
		})
	}
	return items
}

func sumShares(values []int64) int64 {
	var total int64
	for _, v := range values {
		total += v
	}
	return total
}

func TestAllocateDiscount(t *testing.T) {
	t.Run("splits proportionally", func(t *testing.T) {
		// 3000 and 1000 of a 4000 subtotal: a 400 discount is 300 and 100.
		require.Equal(t, []int64{300, 100}, mustAllocate(t, discountLines(3000, 1, 1000, 1), 400))
	})

	t.Run("a discount at the subtotal zeroes every line and no further", func(t *testing.T) {
		got := mustAllocate(t, discountLines(1000, 1, 500, 2), 2000)
		require.Equal(t, []int64{1000, 1000}, got)
	})

	// Above the subtotal no allocation can both sum to the total and keep every line
	// non-negative. Capping silently produced a sum mismatch, which stranded the
	// order: the refund path read the gap and rejected every line, permanently.
	t.Run("a discount beyond the subtotal is refused", func(t *testing.T) {
		for _, total := range []int64{2001, 999999} {
			_, err := allocateDiscount(discountLines(1000, 1, 1000, 1), total)
			require.Error(t, err, "total %d exceeds the subtotal", total)
		}
	})

	t.Run("bad input is refused, not approximated", func(t *testing.T) {
		_, err := allocateDiscount(discountLines(1000, 1), -50)
		require.Error(t, err, "negative discount")

		_, err = allocateDiscount(discountLines(0, 1), 500)
		require.Error(t, err, "an order with no value cannot carry a discount")

		_, err = allocateDiscount([]domain.OrderItem{{UnitPrice: -500, Quantity: 1}}, 100)
		require.Error(t, err, "negative unit price")

		// Overflowed the multiply, under-counted the assigned total, and then indexed
		// past the remainder slots. Must refuse rather than panic in a money path.
		_, err = allocateDiscount(discountLines(400000000000, 1, 1, 1), 300000000000)
		require.Error(t, err)
	})

	t.Run("a zero discount allocates nothing", func(t *testing.T) {
		require.Equal(t, []int64{0, 0}, mustAllocate(t, discountLines(1000, 1, 1000, 1), 0))
		require.Empty(t, mustAllocate(t, nil, 500))
	})

	// "Sums to the total for any input" is not established by a handful of points.
	t.Run("sums to the total across a swept input space", func(t *testing.T) {
		for a := int64(1); a <= 7; a++ {
			for b := int64(1); b <= 7; b++ {
				for c := int64(0); c <= 7; c++ {
					items := discountLines(a*97, 1, b*31, 2, c*13, 3)
					var subtotal int64
					for _, it := range items {
						subtotal += it.UnitPrice * int64(it.Quantity)
					}
					for total := int64(0); total <= subtotal; total += 7 {
						got, err := allocateDiscount(items, total)
						require.NoError(t, err)
						require.Equal(t, total, sumShares(got), "items=%v total=%d", items, total)
						for i, share := range got {
							require.GreaterOrEqual(t, share, int64(0), "line %d negative", i)
							require.LessOrEqual(t, share, items[i].UnitPrice*int64(items[i].Quantity),
								"line %d discounted past its own value", i)
						}
					}
				}
			}
		}
	})

	// Equal remainders must not depend on sort ordering, or two runs of the same
	// order produce different line discounts.
	t.Run("is deterministic for equal remainders", func(t *testing.T) {
		items := discountLines(1000, 1, 1000, 1, 1000, 1)
		first := mustAllocate(t, items, 2)
		for i := 0; i < 20; i++ {
			require.Equal(t, first, mustAllocate(t, items, 2))
		}
	})
}
