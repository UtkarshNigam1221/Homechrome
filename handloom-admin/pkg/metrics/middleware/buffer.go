// Package middleware provides Chi HTTP middleware for the metrics package.
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/handloom/admin/pkg/metrics"
)

// Buffer injects a metrics buffer into the request context and flushes it after
// the handler returns.
func Buffer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := metrics.WithBuffer(r.Context())
		defer func() {
			if err := metrics.Flush(ctx); err != nil {
				slog.ErrorContext(ctx, "metrics: flush failed", "error", err)
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
