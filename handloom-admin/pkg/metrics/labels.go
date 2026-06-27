package metrics

// Canonical metric label-key vocabulary. Every key used in a metrics.L{...}
// map should reference one of these constants rather than a bare string, so
// there is a single source of truth and goconst stays satisfied. The
// cardinality guard (cardinality_test.go) resolves these identifiers back to
// their string value and checks each against its allowlist.
//
// Go uses MixedCaps for constants (not ALL_CAPS) — ALL_CAPS names would fail
// staticcheck ST1003.
const (
	// Geo
	LabelCity    = "city"
	LabelCountry = "country"

	// Device / session
	LabelDeviceType = "device_type"
	LabelIsNewUser  = "is_new_user"

	// Marketing attribution
	LabelUTMSource   = "utm_source"
	LabelUTMMedium   = "utm_medium"
	LabelUTMCampaign = "utm_campaign"

	// Funnel / business
	LabelHasResults = "has_results"
	LabelCategoryID = "category_id"
	LabelCouponCode = "coupon_code"
	LabelOutcome    = "outcome"
	LabelGateway    = "gateway"

	// HTTP / network
	LabelMethod      = "method"
	LabelRoute       = "route"
	LabelStatusClass = "status_class"
	LabelService     = "service"
	LabelTargetHost  = "target_host"

	// DB / AWS
	LabelOperation  = "operation"
	LabelSDKService = "sdk_service"
	LabelStatus     = "status"

	// RUM / page analytics
	LabelPageType  = "page_type"
	LabelErrorType = "error_type"

	// Search intent classification
	LabelIntent = "intent"

	// Catalog filter beacons
	LabelFilterKey = "filter_key"

	// Generic
	LabelReason    = "reason"
	LabelBucket    = "bucket"
	LabelEventType = "event_type"
)
