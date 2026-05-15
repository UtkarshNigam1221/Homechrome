package delhivery

import (
	"strings"

	"github.com/handloom/admin/internal/gateway/courier"
)

// MapStatus converts a Delhivery status string to a canonical ShipmentEvent.
func MapStatus(s string) courier.ShipmentEvent {
	norm := strings.ToLower(strings.TrimSpace(s))
	switch norm {
	case "manifested", "not picked", "pickup awaited":
		return courier.EventManifested
	case "picked up", "pickup complete":
		return courier.EventPickedUp
	case "in transit", "dispatched", "bagged":
		return courier.EventInTransit
	case "out for delivery":
		return courier.EventOutForDelivery
	case "delivered":
		return courier.EventDelivered
	case "ndr", "dto", "undelivered":
		return courier.EventNDR
	case "rto initiated", "rto in transit":
		return courier.EventRTOInitiated
	case "rto delivered":
		return courier.EventRTODelivered
	case "reverse picked up":
		return courier.EventReversePickedUp
	case "reverse delivered":
		return courier.EventReverseDelivered
	default:
		return courier.EventUnknown
	}
}
