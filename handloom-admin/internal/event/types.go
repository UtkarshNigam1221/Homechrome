package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of domain event.
type EventType string

// Order events
const (
	OrderCreated       EventType = "order.created"
	OrderStatusChanged EventType = "order.status_changed"
	OrderCancelled     EventType = "order.cancelled"
)

// Payment events
const (
	PaymentReceived EventType = "payment.received"
	PaymentFailed   EventType = "payment.failed"
	PaymentRefunded EventType = "payment.refunded"
)

// Shipment events
const (
	ShipmentCreated   EventType = "shipment.created"
	ShipmentUpdated   EventType = "shipment.updated"
	ShipmentDelivered EventType = "shipment.delivered"

	// Shipping events (Delhivery)
	ShipmentManifested      EventType = "shipment.manifested"
	ShipmentNDRReattempted  EventType = "shipment.ndr_reattempted"
	ShipmentNDREscalated    EventType = "shipment.ndr_escalated"
	ShipmentPickupScheduled EventType = "shipment.pickup_scheduled"
)

// COD events
const (
	CODRemitted  EventType = "cod.remitted"
	CODUnmatched EventType = "cod.unmatched"
)

// Return events
const (
	ReturnRequested EventType = "return.requested"
	ReturnReceived  EventType = "return.received"
	ReturnRefunded  EventType = "return.refunded"
)

// Product events
const (
	ProductCreated EventType = "product.created"
	ProductUpdated EventType = "product.updated"
	ProductDeleted EventType = "product.deleted"
)

// Inventory events
const (
	InventoryLowStock   EventType = "inventory.low_stock"
	InventoryOutOfStock EventType = "inventory.out_of_stock"
	InventoryRestocked  EventType = "inventory.restocked"
)

// Customer events
const (
	CustomerRegistered EventType = "customer.registered"
	CustomerUpdated    EventType = "customer.updated"
)

// Admin events
const (
	AdminEntityModified EventType = "admin.entity_modified"
	AdminUserLogin      EventType = "admin.user_login"
)

// Event represents a domain event that can be published to subscribers.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// New creates an Event with a unique ID, the current timestamp, and a
// JSON-encoded payload. The source is always "handloom-api".
func New(eventType EventType, data interface{}) Event {
	payload, _ := json.Marshal(data)
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    "handloom-api",
		Timestamp: time.Now(),
		Data:      payload,
	}
}
