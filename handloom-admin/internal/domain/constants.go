// Package domain contains all business entities and interfaces
package domain

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
