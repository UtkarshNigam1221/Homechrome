package metrics

import "time"

// BucketForDuration returns the smallest bucket label such that duration <= bucket boundary.
// boundaries must be sorted ascending in seconds. labels has len(boundaries)+1 entries
// (last is the +Inf bucket).
func BucketForDuration(d time.Duration, boundaries []float64, labels []string) string {
	s := d.Seconds()
	for i, b := range boundaries {
		if s <= b {
			return labels[i]
		}
	}
	return labels[len(labels)-1]
}

// Histogram bucket labels. Defined as constants so the shared boundary sets
// below don't repeat string literals (goconst), and renames stay consistent.
const (
	leInf   = "le_inf"
	le30s   = "le_30s"
	le2m    = "le_2m"
	le10m   = "le_10m"
	le1h    = "le_1h"
	le50ms  = "le_50ms"
	le200ms = "le_200ms"
	le1s    = "le_1s"
	le5s    = "le_5s"
	le10ms  = "le_10ms"
)

// Predefined boundary sets used across the codebase.
var (
	DurationCartToPaymentBoundaries = []float64{30, 120, 600, 3600}
	DurationCartToPaymentLabels     = []string{le30s, le2m, le10m, le1h, leInf}

	DurationHTTPServerBoundaries = []float64{0.05, 0.2, 1.0, 5.0}
	DurationHTTPServerLabels     = []string{le50ms, le200ms, le1s, le5s, leInf}

	DurationDBQueryBoundaries = []float64{0.01, 0.05, 0.2, 1.0}
	DurationDBQueryLabels     = []string{le10ms, le50ms, le200ms, le1s, leInf}
)

// BucketForCartSize returns the matching bucket label for an integer item count.
func BucketForCartSize(n int) string {
	switch {
	case n <= 1:
		return "1"
	case n <= 3:
		return "2_3"
	case n <= 5:
		return "4_5"
	case n <= 10:
		return "6_10"
	default:
		return "11_plus"
	}
}
