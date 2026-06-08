package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/handloom/admin/pkg/metrics"
)

// unmatchedRoute is the constant route label used when no chi pattern matched
// (404s, bot scans). Using the raw URL path here would let unauthenticated
// traffic drive unbounded `route` cardinality in metric_counters.
const unmatchedRoute = "__unmatched__"

// HTTPServer emits http_request{} + http_request_duration{} per request.
// Wraps the next handler. Place AFTER Buffer middleware.
func HTTPServer(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			// chi's wrapper forwards Flusher/Hijacker/Pusher, so streaming /
			// SSE / websocket upgrades downstream keep working.
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := routePattern(r)
			statusClass := strconv.Itoa(statusOrOK(ww.Status())/100) + "xx"
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

// statusOrOK treats an unwritten status (0) as 200, matching net/http's
// implicit-200-on-first-write behaviour.
func statusOrOK(code int) int {
	if code == 0 {
		return http.StatusOK
	}
	return code
}

// routePattern returns the chi route pattern (e.g. /p/{slug}) so the route
// label stays bounded. Falls back to a constant sentinel when no pattern
// matched (e.g. 404s / bot scans) so unmatched routes can't inflate cardinality.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return unmatchedRoute
}
