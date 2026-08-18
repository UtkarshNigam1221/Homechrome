package service

// keyProductID is the shared "product_id" key — used both as a slog field and
// as a metric label. Metric label KEYS live in the central vocabulary
// (metrics.Label*); this stays here because it doubles as a log key.
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
// site; release_unreserved marks the benign already-released case.
const (
	reasonReserve           = "reserve"
	reasonCommit            = "commit"
	reasonRelease           = "release"
	reasonReleaseUnreserved = "release_unreserved"
	reasonRestock           = "restock"
)
