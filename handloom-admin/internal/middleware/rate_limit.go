package middleware

import (
	"context"
	"net/http"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
)

// RateLimitKeyFunc derives the key a request is limited on. Returning an empty
// string exempts the request, which is how policy owners opt callers out.
type RateLimitKeyFunc func(ctx context.Context) string

// RateLimit claims against rule before the handler runs, so a refused request
// costs nothing downstream. Mount it after any validation the key reads from.
func RateLimit(limiter domain.RateLimiter, rule domain.RateLimitRule, key RateLimitKeyFunc) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := key(r.Context())
			if k == "" {
				next.ServeHTTP(w, r)
				return
			}
			if err := limiter.Claim(r.Context(), k, rule); err != nil {
				response.Error(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
