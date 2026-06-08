package domain

import "time"

// ValidStoreEventTypes is the set of accepted frontend tracking event types.
var ValidStoreEventTypes = map[string]struct{}{
	"page_view":                      {},
	"product_viewed":                 {},
	"add_to_cart":                    {},
	"checkout_started":               {},
	"scroll_depth":                   {},
	"category_viewed":                {},
	"out_of_stock_shown":             {},
	"back_in_stock_notify_requested": {},
	"catalog_filter_applied":         {},
	"rum_lcp":                        {},
	"rum_inp":                        {},
	"rum_cls":                        {},
	"rum_ttfb":                       {},
	"rum_js_error":                   {},
	"rum_page_view":                  {},
}

// IsValidStoreEventType checks if the event type is in the allowlist.
func IsValidStoreEventType(eventType string) bool {
	_, ok := ValidStoreEventTypes[eventType]
	return ok
}

// StoreEvent represents a single frontend tracking event
type StoreEvent struct {
	EventType  string                 `json:"event_type" validate:"required"`
	Timestamp  time.Time              `json:"timestamp" validate:"required"`
	SessionID  string                 `json:"session_id" validate:"required"`
	VisitorID  string                 `json:"visitor_id" validate:"required"`
	DeviceType string                 `json:"device_type"`
	PagePath   string                 `json:"page_path"`
	Properties map[string]interface{} `json:"properties"`
}

// StoreEventBatch is the request body for POST /api/v1/store/events
type StoreEventBatch struct {
	Events []StoreEvent `json:"events" validate:"required,min=1,max=25"`
}
