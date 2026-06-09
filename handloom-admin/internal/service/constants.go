package service

// Metric label keys (kept as constants so goconst is satisfied; the cardinality
// guard in pkg/metrics resolves these identifiers back to their string value).
const (
	keyProductID    = "product_id"
	labelCountry    = "country"
	labelCity       = "city"
	labelDeviceType = "device_type"
	labelGateway    = "gateway"
	labelBucket     = "bucket"
	labelOutcome    = "outcome"
	labelReason     = "reason"
	labelCategoryID = "category_id"
	labelUTMSource  = "utm_source"
	labelPageType   = "page_type"
)

// Common metric-label fallback / value strings, de-duplicated across the package.
const (
	labelUnknown   = "unknown"
	labelOther     = "other"
	gatewayPhonePe = "phonepe"
)
