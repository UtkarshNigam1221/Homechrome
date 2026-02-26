package middleware

import (
	"net/http"
	"strings"
)

// CatalogCacheControl returns a middleware that applies cache headers to catalog routes.
// The availability endpoint gets "no-store" (needs real-time data),
// all other catalog GET routes get the specified cache value.
func CatalogCacheControl(cacheValue string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				if strings.HasSuffix(r.URL.Path, "/availability") {
					w.Header().Set("Cache-Control", "no-store")
				} else {
					w.Header().Set("Cache-Control", cacheValue)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
