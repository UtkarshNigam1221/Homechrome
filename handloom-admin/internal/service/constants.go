package service

// keyProductID is the shared "product_id" key for slog fields and metric labels.
// Label KEYS live in metrics.Label*; this stays because it doubles as a log key.
const keyProductID = "product_id"

// keyOrderID is the slog field for order-scoped inventory movements, which are
// reported per order rather than per product.
const keyOrderID = "order_id"

// Common metric-label fallback / value strings, de-duplicated across the package.
const (
	labelUnknown   = "unknown"
	labelOther     = "other"
	gatewayPhonePe = "phonepe"
)

// Reason labels for inventory_mutation_failed. One per swallowed-failure call
const (
	reasonReserve = "reserve"
	reasonCommit  = "commit"
	reasonRelease = "release"
	reasonRestock = "restock"
)
