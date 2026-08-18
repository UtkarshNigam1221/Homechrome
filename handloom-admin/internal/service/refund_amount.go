package service

import (
	"fmt"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// refundBreakdown is what a requested set of lines is worth. The client sends
// lines and quantities; this is the only thing that decides money.
type refundBreakdown struct {
	Total int64
	Items []domain.RefundItem

	// IsFinal means this refund clears the last unrefunded unit on the order.
	// It is what earns back the shipping, and what makes this refund absorb the
	// rounding residual.
	IsFinal bool
}

// prorate returns value's share of the whole for a line worth lineSubtotal,
// rounded half-up. Integer paise throughout — no float ever touches money.
func prorate(value, lineSubtotal, subtotal int64) int64 {
	if value == 0 || subtotal == 0 {
		return 0
	}
	return (value*lineSubtotal + subtotal/2) / subtotal
}

// deriveRefundAmount computes the refund for the requested lines.
//
// Each line is worth its own value less its prorated share of the order
// discount, plus its share of the tax. Shipping comes back only with the refund
// that clears the order, because until then the parcel still ships.
//
// That last refund is also made to absorb the residual left by per-line
// rounding: without it the refunds sum to a few paise short of the order and it
// never reaches fully refunded.
func deriveRefundAmount(order *domain.Order, requested []domain.CreateRefundItemRequest, priorRefunded int64) (*refundBreakdown, error) {
	if len(requested) == 0 {
		return nil, errors.BadRequest("A refund needs at least one line")
	}

	byID := make(map[string]*domain.OrderItem, len(order.Items))
	for i := range order.Items {
		byID[order.Items[i].ID] = &order.Items[i]
	}

	seen := make(map[string]bool, len(requested))
	remaining := make(map[string]int, len(requested))

	breakdown := &refundBreakdown{Items: make([]domain.RefundItem, 0, len(requested))}
	var runningTotal int64

	for _, req := range requested {
		item, ok := byID[req.OrderItemID]
		if !ok {
			return nil, errors.BadRequest(fmt.Sprintf("Order has no line %s", req.OrderItemID))
		}
		if seen[req.OrderItemID] {
			// Two entries for one line would each pass the remainder check on
			// their own and together exceed it.
			return nil, errors.BadRequest(fmt.Sprintf("Line %s appears twice", req.OrderItemID))
		}
		seen[req.OrderItemID] = true

		left := item.Quantity - item.RefundedQuantity
		if req.Quantity < 1 || req.Quantity > left {
			return nil, errors.BadRequest(fmt.Sprintf(
				"Line %s has %d unit(s) left to refund, asked for %d", req.OrderItemID, left, req.Quantity))
		}
		remaining[req.OrderItemID] = left - req.Quantity

		lineSubtotal := item.UnitPrice * int64(req.Quantity)
		amount := lineSubtotal -
			prorate(order.DiscountAmount, lineSubtotal, order.Subtotal) +
			prorate(order.TaxAmount, lineSubtotal, order.Subtotal)

		runningTotal += amount
		breakdown.Items = append(breakdown.Items, domain.RefundItem{
			OrderItemID: item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    req.Quantity,
			Amount:      amount,
			Restock:     req.Restock,
		})
	}

	breakdown.IsFinal = clearsOrder(order, remaining)
	breakdown.Total = runningTotal

	if breakdown.IsFinal {
		// Whatever is left of the order is what is left to refund, residual and
		// shipping included. Deriving it this way rather than adding shipping to
		// the running total is what guarantees the refunds sum to the order.
		breakdown.Total = order.TotalAmount - priorRefunded
		if adjust := breakdown.Total - runningTotal; adjust != 0 {
			last := &breakdown.Items[len(breakdown.Items)-1]
			last.Amount += adjust
		}
	}

	if breakdown.Total < 0 || priorRefunded+breakdown.Total > order.TotalAmount {
		return nil, errors.BadRequest("Refund would exceed the order total")
	}

	return breakdown, nil
}

// clearsOrder reports whether nothing would be left unrefunded once the
// requested lines go back.
func clearsOrder(order *domain.Order, remainingAfter map[string]int) bool {
	for i := range order.Items {
		item := &order.Items[i]

		left, requested := remainingAfter[item.ID]
		if !requested {
			left = item.Quantity - item.RefundedQuantity
		}
		if left > 0 {
			return false
		}
	}
	return true
}
