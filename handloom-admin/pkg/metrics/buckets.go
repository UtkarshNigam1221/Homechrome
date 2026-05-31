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

// Predefined boundary sets used across the codebase.
var (
	DurationCartToPaymentBoundaries = []float64{30, 120, 600, 3600}
	DurationCartToPaymentLabels     = []string{"le_30s", "le_2m", "le_10m", "le_1h", "le_inf"}

	DurationHTTPServerBoundaries = []float64{0.05, 0.2, 1.0, 5.0}
	DurationHTTPServerLabels     = []string{"le_50ms", "le_200ms", "le_1s", "le_5s", "le_inf"}

	DurationDBQueryBoundaries = []float64{0.01, 0.05, 0.2, 1.0}
	DurationDBQueryLabels     = []string{"le_10ms", "le_50ms", "le_200ms", "le_1s", "le_inf"}
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
