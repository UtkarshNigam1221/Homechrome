package domain

import "time"

// ==================== SHIPMENT ENTITY ====================

// ShipmentStatus defines the status of a shipment
type ShipmentStatus string

const (
	ShipmentStatusCreated        ShipmentStatus = "CREATED"
	ShipmentStatusPickedUp       ShipmentStatus = "PICKED_UP"
	ShipmentStatusInTransit      ShipmentStatus = "IN_TRANSIT"
	ShipmentStatusOutForDelivery ShipmentStatus = "OUT_FOR_DELIVERY"
	ShipmentStatusDelivered      ShipmentStatus = "DELIVERED"
	ShipmentStatusRTO            ShipmentStatus = "RTO"
	ShipmentStatusManifested     ShipmentStatus = "MANIFESTED"
	ShipmentStatusNDR            ShipmentStatus = "NDR"
	ShipmentStatusNDREscalated   ShipmentStatus = "NDR_ESCALATED"
	ShipmentStatusReturning      ShipmentStatus = "RETURNING"
	ShipmentStatusReturned       ShipmentStatus = "RETURNED"
)

// ShipmentPriority sets manifest+pickup cadence.
type ShipmentPriority string

const (
	PriorityNormal   ShipmentPriority = "NORMAL"
	PriorityPriority ShipmentPriority = "PRIORITY"
)

// Shipment represents a shipment for an order
type Shipment struct {
	ID                  string           `json:"id" dynamodbav:"id"`
	PK                  string           `json:"-" dynamodbav:"PK"`
	SK                  string           `json:"-" dynamodbav:"SK"`
	EntityType          string           `json:"-" dynamodbav:"entity_type"`
	OrderID             string           `json:"order_id" dynamodbav:"order_id"`
	Provider            string           `json:"provider" dynamodbav:"provider"`
	ProviderOrderID     string           `json:"provider_order_id,omitempty" dynamodbav:"provider_order_id,omitempty"`
	ProviderShipmentID  string           `json:"provider_shipment_id,omitempty" dynamodbav:"provider_shipment_id,omitempty"`
	AWBNumber           string           `json:"awb_number,omitempty" dynamodbav:"awb_number,omitempty"`
	CourierName         string           `json:"courier_name,omitempty" dynamodbav:"courier_name,omitempty"`
	Status              ShipmentStatus   `json:"status" dynamodbav:"status"`
	LabelURL            string           `json:"label_url,omitempty" dynamodbav:"label_url,omitempty"`
	EstimatedDelivery   string           `json:"estimated_delivery,omitempty" dynamodbav:"estimated_delivery,omitempty"`
	WeightGrams         int              `json:"weight_grams" dynamodbav:"weight_grams"`
	ShippedAt           *time.Time       `json:"shipped_at,omitempty" dynamodbav:"shipped_at,omitempty"`
	DeliveredAt         *time.Time       `json:"delivered_at,omitempty" dynamodbav:"delivered_at,omitempty"`
	Priority            ShipmentPriority `json:"priority" dynamodbav:"priority"`
	PriorityStatus      string           `json:"-" dynamodbav:"priority_status"` // GSI key e.g. "NORMAL#CREATED"
	PickupLocation      string           `json:"pickup_location" dynamodbav:"pickup_location"`
	ManifestID          string           `json:"manifest_id,omitempty" dynamodbav:"manifest_id,omitempty"`
	NDRCount            int              `json:"ndr_count" dynamodbav:"ndr_count"`
	LastNDRReason       string           `json:"last_ndr_reason,omitempty" dynamodbav:"last_ndr_reason,omitempty"`
	LastNDRAt           *time.Time       `json:"last_ndr_at,omitempty" dynamodbav:"last_ndr_at,omitempty"`
	NDREscalated        bool             `json:"ndr_escalated" dynamodbav:"ndr_escalated"`
	ShippingChargePaise int64            `json:"shipping_charge" dynamodbav:"shipping_charge"`
	ActualWeightGrams   int              `json:"actual_weight_grams" dynamodbav:"actual_weight_grams"`
	ChargedWeightGrams  int              `json:"charged_weight_grams" dynamodbav:"charged_weight_grams"`
	IsCOD               bool             `json:"is_cod" dynamodbav:"is_cod"`
	CODAmountPaise      int64            `json:"cod_amount" dynamodbav:"cod_amount"`
	CODRemitted         bool             `json:"cod_remitted" dynamodbav:"cod_remitted"`
	CODRemittedAt       *time.Time       `json:"cod_remitted_at,omitempty" dynamodbav:"cod_remitted_at,omitempty"`
	CODRemittanceRef    string           `json:"cod_remittance_ref,omitempty" dynamodbav:"cod_remittance_ref,omitempty"`
	BaseEntity
}

// TableName returns the DynamoDB table name for Shipment
func (s *Shipment) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB primary key + composite GSI key for a Shipment.
//
// IMPORTANT: priority_status is a GSI partition key built from Priority + Status.
// SetKeys must run before any Put or status change. ShipmentRepository.UpdateStatus
// takes the priority as a parameter so it can recompute and write priority_status
// atomically in a single UpdateItem (no stale read).
func (s *Shipment) SetKeys() {
	s.PK = "ORDER#" + s.OrderID
	s.SK = "SHIPMENT#" + s.ID
	s.EntityType = "SHIPMENT"
	if s.Priority == "" {
		s.Priority = PriorityNormal
	}
	s.PriorityStatus = string(s.Priority) + "#" + string(s.Status)
}

// ==================== SHIPPING REQUEST/RESPONSE TYPES ====================

// ServiceabilityRequest contains data for checking shipping serviceability
type ServiceabilityRequest struct {
	Pincode string `json:"pincode" validate:"required,len=6"`
}

// ServiceabilityResult contains the result of a serviceability check
type ServiceabilityResult struct {
	Serviceable bool            `json:"serviceable"`
	Couriers    []CourierOption `json:"couriers,omitempty"`
}

// CourierOption represents an available courier option
type CourierOption struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Rate          int64  `json:"rate"` // in paise
	EstimatedDays int    `json:"estimated_days"`
}
