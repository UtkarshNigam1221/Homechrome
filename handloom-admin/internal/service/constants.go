package service

// keyProductID is the shared "product_id" key — used both as a slog field and
// as a metric label. Metric label KEYS live in the central vocabulary
// (metrics.Label*); this stays here because it doubles as a log key.
const keyProductID = "product_id"

// Common metric-label fallback / value strings, de-duplicated across the package.
const (
	labelUnknown   = "unknown"
	labelOther     = "other"
	gatewayPhonePe = "phonepe"
)
