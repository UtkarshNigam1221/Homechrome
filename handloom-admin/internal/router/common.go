// Package router provides shared routing utilities for Lambda functions
package router

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/handloom/admin/internal/middleware"
	metricsmw "github.com/handloom/admin/pkg/metrics/middleware"
	"github.com/handloom/admin/pkg/telemetry"
)

// Config contains router configuration
type Config struct {
	AllowedOrigins []string
	Debug          bool
}

// ApplyObservability installs the provider-agnostic middleware stack every
// HTTP entry point should have: server tracing, request ID, metrics, geo, logs,
// panic recovery. CORS and auth are intentionally left out — those are
// per-router concerns (admin JWT, store JWT, embedder HMAC). Call this from any
// chi router (NewBaseRouter, or a hand-rolled one like the embedder's) so the
// observability stack can't drift between services.
func ApplyObservability(r chi.Router) {
	// OTel HTTP server middleware — creates a SERVER span per request.
	// Without this, Lambda invocations show no traces in Tempo.
	// Service name comes from OTEL_SERVICE_NAME injected by CDK applyTelemetry.
	otelServiceName := os.Getenv("OTEL_SERVICE_NAME")
	if otelServiceName == "" {
		otelServiceName = "handloom-lambda"
	}
	r.Use(telemetry.HTTPMiddleware(otelServiceName))
	r.Use(telemetry.TraceIDMiddleware)

	r.Use(middleware.RequestID)
	r.Use(metricsmw.Buffer)                      // injects metrics buffer + defers flush
	r.Use(metricsmw.HTTPServer(otelServiceName)) // emits http_request{} + duration
	r.Use(middleware.GeoExtractor)               // reads the X-Hc-Visitor header into ctx
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
}

// NewBaseRouter creates a base router with common middleware
// Set addHealthCheck to true to add the /health endpoint (for unauthenticated routers)
func NewBaseRouter(cfg Config, addHealthCheck bool) *chi.Mux {
	r := chi.NewRouter()

	ApplyObservability(r)

	// Skip compression in Lambda — the gzipped body corrupts in the
	// APIGatewayProxyResponse string field. API Gateway handles compression.
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		r.Use(chimiddleware.Compress(5))
	}

	// CORS — AllowCredentials requires a specific origin, not "*".
	// When explicit origins are configured, use them; otherwise allow all by
	// reflecting the request Origin back (AllowOriginFunc).
	corsOpts := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", middleware.VisitorHeader},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if len(cfg.AllowedOrigins) > 0 && cfg.AllowedOrigins[0] != "*" {
		corsOpts.AllowedOrigins = cfg.AllowedOrigins
	} else {
		corsOpts.AllowOriginFunc = func(r *http.Request, origin string) bool { return true }
	}
	r.Use(cors.Handler(corsOpts))

	// Health check for Lambda warm-up (only for unauthenticated routers)
	if addHealthCheck {
		r.Get("/health", healthHandler)
	}

	return r
}

// NewAuthenticatedRouter creates a router that requires authentication
func NewAuthenticatedRouter(cfg Config, authMiddleware *middleware.Auth) *chi.Mux {
	// Don't add health check yet - we need to add auth middleware first
	r := NewBaseRouter(cfg, false)

	// Apply auth middleware to all routes
	r.Use(authMiddleware.Authenticate)

	// Add health check after middleware (it will require auth)
	// Or we can add an unauthenticated health check in a group
	r.Get("/health", healthHandler)

	return r
}

// DefaultTimeout returns the default Lambda timeout
func DefaultTimeout() time.Duration {
	return 29 * time.Second
}
