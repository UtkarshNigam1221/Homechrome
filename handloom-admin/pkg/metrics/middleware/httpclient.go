package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/handloom/admin/pkg/metrics"
)

// HTTPClientTransport wraps http.RoundTripper to emit http_client_call{} +
// http_client_duration{} metrics for outbound calls.
type HTTPClientTransport struct {
	Service string
	Base    http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *HTTPClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	dur := time.Since(start)

	statusClass := "5xx"
	if err == nil && resp != nil {
		statusClass = strconv.Itoa(resp.StatusCode/100) + "xx"
	}
	metrics.Record(req.Context(), "http_client_call", metrics.L{
		"service":      t.Service,
		"target_host":  req.URL.Host,
		"status_class": statusClass,
	})
	metrics.RecordDuration(req.Context(), "http_client_duration", dur, metrics.L{
		"service":     t.Service,
		"target_host": req.URL.Host,
	})
	return resp, err
}
