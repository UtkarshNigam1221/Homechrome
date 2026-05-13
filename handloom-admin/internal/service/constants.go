package service

// Shared structured-log / event-payload keys. Producers (slog calls, event
// payloads) and consumers (analytics aggregator extracts these fields from
// stored events) must use identical names — drift here breaks aggregation.
const (
	keyProductID = "product_id"

	// analyticsPeriodDaily is the default analytics aggregation period.
	analyticsPeriodDaily = "daily"
)
