package metrics

import "time"

// L is a label map for metric annotations.
type L map[string]string

// RetentionClass controls how long a metric is stored.
type RetentionClass string

const (
	RetentionBusiness RetentionClass = "business"
	RetentionService  RetentionClass = "service"
)

// Event is a single metric observation.
type Event struct {
	Metric         string         `json:"metric"`
	Labels         L              `json:"labels"`
	Value          int64          `json:"value"`
	SumValue       int64          `json:"sum_value"`
	RetentionClass RetentionClass `json:"retention_class"`
	EmittedAt      time.Time      `json:"emitted_at"`
	IdempotencyKey string         `json:"idempotency_key"`
}
