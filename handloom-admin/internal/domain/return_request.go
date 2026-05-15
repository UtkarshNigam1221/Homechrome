package domain

import "time"

// ReturnStatus is the lifecycle state of a customer return.
type ReturnStatus string

const (
	ReturnStatusRequested ReturnStatus = "REQUESTED"
	ReturnStatusPickedUp  ReturnStatus = "PICKED_UP"
	ReturnStatusInTransit ReturnStatus = "IN_TRANSIT"
	ReturnStatusReceived  ReturnStatus = "RECEIVED"
	ReturnStatusRefunded  ReturnStatus = "REFUNDED"
	ReturnStatusCancelled ReturnStatus = "CANCELLED"
)

// ReturnItem is one line item being returned (supports partial returns).
type ReturnItem struct {
	ProductID string `json:"product_id" dynamodbav:"product_id"`
	SKU       string `json:"sku" dynamodbav:"sku"`
	Quantity  int    `json:"quantity" dynamodbav:"quantity"`
	UnitPaise int64  `json:"unit_paise" dynamodbav:"unit_paise"`
}

// ReturnRequest represents an admin-initiated reverse shipment.
type ReturnRequest struct {
	PK                string       `json:"-" dynamodbav:"PK"`
	SK                string       `json:"-" dynamodbav:"SK"`
	EntityType        string       `json:"-" dynamodbav:"entity_type"`
	ID                string       `json:"id" dynamodbav:"id"`
	OrderID           string       `json:"order_id" dynamodbav:"order_id"`
	ShipmentID        string       `json:"shipment_id" dynamodbav:"shipment_id"`
	ReverseAWB        string       `json:"reverse_awb" dynamodbav:"reverse_awb"`
	ReverseShipmentID string       `json:"reverse_shipment_id" dynamodbav:"reverse_shipment_id"`
	Reason            string       `json:"reason" dynamodbav:"reason"`
	Items             []ReturnItem `json:"items" dynamodbav:"items"`
	Status            ReturnStatus `json:"status" dynamodbav:"status"`
	RefundAmountPaise int64        `json:"refund_amount_paise" dynamodbav:"refund_amount_paise"`
	RefundedAt        *time.Time   `json:"refunded_at,omitempty" dynamodbav:"refunded_at,omitempty"`
	CreatedBy         string       `json:"created_by" dynamodbav:"created_by"`
	BaseEntity
}

// SetKeys assigns PK/SK colocated with the originating order.
func (r *ReturnRequest) SetKeys() {
	r.PK = "ORDER#" + r.OrderID
	r.SK = "RETURN#" + r.ID
	r.EntityType = EntityTypeReturnRequest
}

// TableName returns the DynamoDB table for ReturnRequest (orders table).
func (r *ReturnRequest) TableName() string {
	return TableOrders
}
