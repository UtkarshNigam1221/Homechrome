package domain

import "time"

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
