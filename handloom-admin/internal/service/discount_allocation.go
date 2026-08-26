package service

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// allocateDiscount splits total across items by line value, in integer paise, so the
// shares sum to exactly total: floor each, then hand the remainder to the largest
// fractional parts. Errors rather than approximating — a mismatched allocation strands
// the order, because every later refund reads the gap and rejects itself.
func allocateDiscount(items []domain.OrderItem, total int64) ([]int64, error) {
	shares := make([]int64, len(items))
	if total < 0 {
		return nil, errors.BadRequest("Discount cannot be negative")
	}
	if total == 0 || len(items) == 0 {
		return shares, nil
	}

	var subtotal int64
	values := make([]int64, len(items))
	for i := range items {
		if items[i].UnitPrice < 0 || items[i].Quantity < 0 {
			return nil, errors.BadRequest(fmt.Sprintf("Line %s has a negative price or quantity", items[i].ID))
		}
		values[i] = items[i].UnitPrice * int64(items[i].Quantity)
		subtotal += values[i]
	}
	if subtotal == 0 {
		return nil, errors.BadRequest("Cannot discount an order with no value")
	}
	// Above the subtotal there is no allocation that both sums to total and leaves
	// every line non-negative. The caller caps the discount; this refuses to guess.
	if total > subtotal {
		return nil, errors.BadRequest("Discount exceeds the order subtotal")
	}
	// total <= subtotal, so total*value <= subtotal*value; guard the product anyway
	// because the loop below multiplies before dividing.
	if total > math.MaxInt64/subtotal {
		return nil, errors.BadRequest("Discount allocation overflows")
	}

	remainders := make([]int64, len(items))
	byRemainder := make([]int, len(items))
	var assigned int64
	for i, value := range values {
		shares[i] = total * value / subtotal
		assigned += shares[i]
		remainders[i] = (total * value) % subtotal
		byRemainder[i] = i
	}

	// Largest remainder first. Stable, so equal remainders resolve by original order
	// every run rather than however the sort happens to land.
	slices.SortStableFunc(byRemainder, func(a, b int) int {
		return cmp.Compare(remainders[b], remainders[a])
	})

	// Floor loses under a paisa per line, so the shortfall never exceeds one per line.
	for i := int64(0); i < total-assigned; i++ {
		shares[byRemainder[i]]++
	}
	return shares, nil
}
