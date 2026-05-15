// Package domain contains all business entities and interfaces
package domain

// DynamoDB table names
const (
	TableCore     = "handloom-core"
	TableOrders   = "handloom-orders"
	TableShipping = "handloom-shipping"
)

// DynamoDB sort key constants
const (
	SKMetadata = "METADATA"
)

// DynamoDB entity_type constants for the shipping subsystem.
// These tag rows that share a GSI partition (entity-status-index, etc.)
// so list-by-status queries can filter to a single logical entity.
const (
	EntityTypeShipping      = "SHIPPING_RATE"
	EntityTypePincodeZone   = "PINCODE_ZONE"
	EntityTypeCODRemittance = "COD_REMITTANCE"
	EntityTypeReturnRequest = "RETURN_REQUEST"
)

// DateLayout is the standard date format used across the application.
const DateLayout = "2006-01-02"

// OTPValidityMinutes is the validity window for customer login OTPs.
// Used by the OTP repository TTL and the SMS template payload.
const (
	OTPValidityMinutes    = 5
	OTPValidityMinutesStr = "5"
)
