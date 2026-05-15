package courier

import "time"

// ShipmentEvent is the canonical carrier-agnostic event/state value.
type ShipmentEvent string

const (
	EventManifested       ShipmentEvent = "MANIFESTED"
	EventPickedUp         ShipmentEvent = "PICKED_UP"
	EventInTransit        ShipmentEvent = "IN_TRANSIT"
	EventOutForDelivery   ShipmentEvent = "OUT_FOR_DELIVERY"
	EventDelivered        ShipmentEvent = "DELIVERED"
	EventNDR              ShipmentEvent = "NDR"
	EventRTOInitiated     ShipmentEvent = "RTO_INITIATED"
	EventRTODelivered     ShipmentEvent = "RTO_DELIVERED"
	EventReversePickedUp  ShipmentEvent = "REVERSE_PICKED_UP"
	EventReverseDelivered ShipmentEvent = "REVERSE_DELIVERED"
	EventUnknown          ShipmentEvent = "UNKNOWN"
)

// PaymentMode for shipments.
type PaymentMode string

const (
	PaymentPrepaid PaymentMode = "PREPAID"
	PaymentCOD     PaymentMode = "COD"
)

// PincodeInfo is the result of a pincode serviceability check.
type PincodeInfo struct {
	Pincode          string
	Serviceable      bool
	Zone             string // A|B|C|D|E
	City             string
	State            string
	CODAvailable     bool
	PrepaidAvailable bool
	EstimatedDays    int
}

// RateRow is one row in a carrier rate matrix.
type RateRow struct {
	Zone            string
	WeightSlabGrams int // 500, 1000, 2000, 5000, 10000
	PrepaidPaise    int64
	CODPaise        int64
	RTOPaise        int64
}

// Address used in shipment + reverse pickup requests.
type Address struct {
	FirstName    string
	LastName     string
	Phone        string
	Email        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Pincode      string
	Country      string
}

// ShipmentItem is one line item in a shipment.
type ShipmentItem struct {
	Name      string
	SKU       string
	Quantity  int
	UnitPaise int64
}

// CreateShipmentRequest is a request to create a forward shipment.
type CreateShipmentRequest struct {
	OrderID            string
	PickupLocation     string
	Customer           Address
	Items              []ShipmentItem
	PaymentMode        PaymentMode
	CODAmountPaise     int64
	WeightGrams        int
	LengthCm           int
	BreadthCm          int
	HeightCm           int
	DeclaredValuePaise int64
}

// CreateShipmentResult is returned by CreateShipment.
type CreateShipmentResult struct {
	AWB               string
	CarrierShipmentID string
	UploadWBN         string
	EstimatedDays     int
	RawResponse       []byte
}

// ManifestResult is returned by CreateManifest.
type ManifestResult struct {
	ManifestID string
	PDFURL     string
	AWBCount   int
}

// TrackingScan is a single scan event in tracking history.
type TrackingScan struct {
	Status      ShipmentEvent
	Location    string
	Time        time.Time
	Description string
}

// TrackingInfo is the result of TrackByAWB.
type TrackingInfo struct {
	AWB             string
	Status          ShipmentEvent
	CurrentLocation string
	LastUpdate      time.Time
	History         []TrackingScan
	NDRReason       string
	NDRAttempts     int
}

// ReversePickupRequest creates a reverse pickup from customer to warehouse.
type ReversePickupRequest struct {
	OriginalOrderID string
	OriginalAWB     string
	Customer        Address
	PickupLocation  string
	Items           []ShipmentItem
	Reason          string
}

// ReversePickupResult is returned by CreateReversePickup.
type ReversePickupResult struct {
	ReverseAWB        string
	CarrierShipmentID string
	EstimatedDays     int
}

// RemittanceRow is one entry in a daily COD remittance payout.
type RemittanceRow struct {
	AWB         string
	OrderRef    string
	AmountPaise int64
	UTR         string
	RemittedAt  time.Time
}

// WebhookEvent is a parsed carrier webhook payload.
type WebhookEvent struct {
	AWB       string
	Status    ShipmentEvent
	Location  string
	Timestamp time.Time
	NDRReason string
	IsReverse bool
}
