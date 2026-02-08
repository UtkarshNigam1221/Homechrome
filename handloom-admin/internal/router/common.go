// Package router provides shared routing utilities for Lambda functions
package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
)

// Config contains router configuration
type Config struct {
	AllowedOrigins []string
	Debug          bool
}

// NewBaseRouter creates a base router with common middleware
// Set addHealthCheck to true to add the /health endpoint (for unauthenticated routers)
func NewBaseRouter(cfg Config, log *logger.Logger, addHealthCheck bool) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recoverer(log))
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Compress(5))

	// CORS
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check for Lambda warm-up (only for unauthenticated routers)
	if addHealthCheck {
		r.Get("/health", healthHandler)
	}

	return r
}

// NewAuthenticatedRouter creates a router that requires authentication
func NewAuthenticatedRouter(cfg Config, log *logger.Logger, authMiddleware *middleware.Auth) *chi.Mux {
	// Don't add health check yet - we need to add auth middleware first
	r := NewBaseRouter(cfg, log, false)

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
