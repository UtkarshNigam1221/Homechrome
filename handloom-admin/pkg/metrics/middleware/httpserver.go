package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/pkg/metrics"
)

// HTTPServer emits http_request{} + http_request_duration{} per request.
// Wraps the next handler. Place AFTER Buffer middleware.
func HTTPServer(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)

			route := routePattern(r)
			statusClass := strconv.Itoa(ww.status/100) + "xx"
			metrics.Record(r.Context(), "http_request", metrics.L{
				"service":      service,
				"method":       r.Method,
				"route":        route,
				"status_class": statusClass,
			})
			metrics.RecordDuration(r.Context(), "http_request_duration", time.Since(start), metrics.L{
				"service": service,
				"method":  r.Method,
				"route":   route,
			})
		})
	}
}

// routePattern returns the chi route pattern (e.g. /p/{slug}) so the route
// label stays bounded. Falls back to the raw path only when no pattern
// matched (e.g. 404s), which keeps unmatched-route cardinality self-limiting.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
