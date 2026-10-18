package service

import (
	"fmt"

	"github.com/handloom/admin/internal/domain"
)

// orderStatusPush is the notification for a status, or nil for one no shopper
// waits on. Apart from the order service so copy edits miss fulfillment logic.
func orderStatusPush(order *domain.Order) *domain.PushPayload {
	var title, body string

	switch order.Status {
	case domain.OrderStatusShipped:
		title = fmt.Sprintf("%s is on its way", order.OrderNumber)
		body = "Your handloom order has left our workshop. Tap to follow it."
	case domain.OrderStatusDelivered:
		title = fmt.Sprintf("%s has arrived", order.OrderNumber)
		body = "Your order was delivered. We hope you love it."
	case domain.OrderStatusCancelled:
		title = fmt.Sprintf("%s was cancelled", order.OrderNumber)
		body = "This order will not ship. Any payment is on its way back to you."
	default:
		return nil
	}

	return &domain.PushPayload{
		Title: title,
		Body:  body,
		URL:   "/account/orders/" + order.ID,
		// One tag per order: a delivered notice replaces the shipped one rather
		// than stacking two cards about the same parcel.
		Tag: "order-" + order.ID,
	}
}
