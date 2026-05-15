package courier

import "github.com/handloom/admin/internal/domain"

// ToShipmentStatus maps a canonical ShipmentEvent to a domain.ShipmentStatus.
func ToShipmentStatus(e ShipmentEvent) domain.ShipmentStatus {
	switch e {
	case EventManifested:
		return domain.ShipmentStatusManifested
	case EventPickedUp:
		return domain.ShipmentStatusPickedUp
	case EventInTransit:
		return domain.ShipmentStatusInTransit
	case EventOutForDelivery:
		return domain.ShipmentStatusOutForDelivery
	case EventDelivered:
		return domain.ShipmentStatusDelivered
	case EventNDR:
		return domain.ShipmentStatusNDR
	case EventRTOInitiated, EventRTODelivered:
		return domain.ShipmentStatusRTO
	case EventReversePickedUp:
		return domain.ShipmentStatusReturning
	case EventReverseDelivered:
		return domain.ShipmentStatusReturned
	default:
		return domain.ShipmentStatusCreated
	}
}

// ToReturnStatus maps a canonical ShipmentEvent to a domain.ReturnStatus.
func ToReturnStatus(e ShipmentEvent) domain.ReturnStatus {
	switch e {
	case EventReversePickedUp:
		return domain.ReturnStatusPickedUp
	case EventReverseDelivered:
		return domain.ReturnStatusReceived
	default:
		return domain.ReturnStatusInTransit
	}
}
