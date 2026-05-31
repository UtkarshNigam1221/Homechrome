package metrics

// knownServices is the allowlist of Lambda function names that may publish metrics.
var knownServices = map[string]struct{}{
	// Admin Lambdas
	"handloom-auth-dev":     {},
	"handloom-auth-prod":    {},
	"handloom-user-dev":     {},
	"handloom-user-prod":    {},
	"handloom-catalog-dev":  {},
	"handloom-catalog-prod": {},
	"handloom-asset-dev":    {},
	"handloom-asset-prod":   {},
	"handloom-order-dev":    {},
	"handloom-order-prod":   {},
	"handloom-coupon-dev":   {},
	"handloom-coupon-prod":  {},
	"handloom-pricing-dev":  {},
	"handloom-pricing-prod": {},
	"handloom-report-dev":   {},
	"handloom-report-prod":  {},
	"handloom-audit-dev":    {},
	"handloom-audit-prod":   {},
	"handloom-admin-dev":    {},
	"handloom-admin-prod":   {},
	// Store Lambdas
	"handloom-store-auth-dev":      {},
	"handloom-store-auth-prod":     {},
	"handloom-store-catalog-dev":   {},
	"handloom-store-catalog-prod":  {},
	"handloom-store-cart-dev":      {},
	"handloom-store-cart-prod":     {},
	"handloom-store-checkout-dev":  {},
	"handloom-store-checkout-prod": {},
	"handloom-store-orders-dev":    {},
	"handloom-store-orders-prod":   {},
	"handloom-store-profile-dev":   {},
	"handloom-store-profile-prod":  {},
	"handloom-store-tracking-dev":  {},
	"handloom-store-tracking-prod": {},
	"handloom-store-events-dev":    {},
	"handloom-store-events-prod":   {},
	"handloom-store-webhooks-dev":  {},
	"handloom-store-webhooks-prod": {},
	// Worker Lambdas
	"handloom-worker-notification-dev":  {},
	"handloom-worker-notification-prod": {},
	"handloom-worker-report-dev":        {},
	"handloom-worker-report-prod":       {},
	"handloom-worker-analytics-dev":     {},
	"handloom-worker-analytics-prod":    {},
	"handloom-worker-audit-dev":         {},
	"handloom-worker-audit-prod":        {},
	// Metrics consumer
	"handloom-metrics-consumer-dev":  {},
	"handloom-metrics-consumer-prod": {},
	// Migrator
	"handloom-migrator-dev":  {},
	"handloom-migrator-prod": {},
	// Embedder
	"handloom-embedder-dev":  {},
	"handloom-embedder-prod": {},
}

// IsKnownService reports whether name is an allowlisted Lambda function name.
func IsKnownService(name string) bool {
	_, ok := knownServices[name]
	return ok
}
