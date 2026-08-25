package router

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/validator"
)

// The API Gateway integration is a proxy, so the Lambda is handed the full
// request path — /admin/utm-links, not /utm-links. Mounting without the /admin
// prefix made every authenticated request 404 inside chi while API Gateway and
// the auth middleware both still looked healthy. Match real paths rather than
// chi's pattern strings: a pattern can read correctly while the path it is
// supposed to serve fails to resolve.
func TestNewUTMLinkRouter_MatchesAdminPaths(t *testing.T) {
	validation := middleware.NewValidation(validator.New(), middleware.ValidationConfig{})
	r := chi.NewMux()
	NewUTMLinkRouter(r, handler.NewUTMLinkHandler(nil, validation))

	routed := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/utm-links"},
		{http.MethodPost, "/admin/utm-links"},
		{http.MethodGet, "/admin/utm-links/utm_abc123"},
		{http.MethodPatch, "/admin/utm-links/utm_abc123"},
		{http.MethodDelete, "/admin/utm-links/utm_abc123"},
	}
	for _, tt := range routed {
		assert.True(t, r.Match(chi.NewRouteContext(), tt.method, tt.path),
			"%s %s must route", tt.method, tt.path)
	}

	// The un-prefixed path is what the old mount exposed. Assert it is gone so a
	// future edit cannot quietly reintroduce the mismatch.
	assert.False(t, r.Match(chi.NewRouteContext(), http.MethodGet, "/utm-links"),
		"/utm-links must not route; API Gateway sends the /admin prefix")
}
