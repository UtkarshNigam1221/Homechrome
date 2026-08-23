package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// allocateDiscount splits total across items in proportion to line value, so the
// parts sum to exactly total. Integer paise throughout — no float touches money.
//
// Floor each share, then hand the leftover paise out one at a time to the largest
// fractional remainders. Rounding each share independently would not sum back.
//
// Errors rather than approximating: a total the lines cannot carry has no correct
// allocation, and returning a mismatched one strands the order — every later refund
// reads the gap and rejects itself.
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
		if items[i].Quantity > 0 && items[i].UnitPrice > math.MaxInt64/int64(items[i].Quantity) {
			return nil, errors.BadRequest(fmt.Sprintf("Line %s value overflows", items[i].ID))
		}
		values[i] = items[i].UnitPrice * int64(items[i].Quantity)
		if values[i] > math.MaxInt64-subtotal {
			return nil, errors.BadRequest("Order subtotal overflows")
		}
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

	type slot struct {
		index     int
		remainder int64
	}
	slots := make([]slot, len(items))
	var assigned int64
	for i, value := range values {
		shares[i] = total * value / subtotal
		assigned += shares[i]
		slots[i] = slot{index: i, remainder: (total * value) % subtotal}
	}

	// Largest remainder first, then index, so equal remainders resolve the same way
	// every run rather than however sort happens to order them.
	sort.Slice(slots, func(a, b int) bool {
		if slots[a].remainder != slots[b].remainder {
			return slots[a].remainder > slots[b].remainder
		}
		return slots[a].index < slots[b].index
	})

	// Floor loses under a paisa per line, so the shortfall never exceeds one per line.
	for i := int64(0); i < total-assigned; i++ {
		shares[slots[i].index]++
	}
	return shares, nil
}
