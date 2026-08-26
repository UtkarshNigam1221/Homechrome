// Package domain contains all business entities and interfaces
package domain

import "time"

// DynamoDB table names
const (
	TableCore   = "handloom-core"
	TableOrders = "handloom-orders"
)

// DynamoDB sort key constants
const (
	SKMetadata = "METADATA"
)

// DateLayout is the standard date format used across the application.
const DateLayout = "2006-01-02"

// OTPValidityMinutes is the validity window for customer login OTPs.
// Used by the OTP repository TTL and the SMS template payload.
const (
	OTPValidityMinutes    = 5
	OTPValidityMinutesStr = "5"
)

// OTP send limits, keyed on the recipient rather than the caller: an attacker
// rotating IPs still cannot use us to bill or spam one number.
const (
	OTPSendCooldown   = time.Minute
	OTPSendsPerWindow = 3
	OTPSendWindow     = time.Hour
)
