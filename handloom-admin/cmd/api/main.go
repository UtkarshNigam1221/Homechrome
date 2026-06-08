// Package main is the entry point for the API server
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/wire"
	"github.com/handloom/admin/pkg/slogx"
	"github.com/handloom/admin/pkg/telemetry"
)

func main() {
	cfg := config.Load()

	slogx.Setup(cfg.App.Debug)

	telShutdown := telemetry.MustInit(
		"handloom-monolith",
		cfg.Telemetry.ServiceVersion,
		cfg.Telemetry.Environment,
	)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		telShutdown(shutdownCtx)
	}()

	slog.Info("Starting handloom-admin API server", "environment", cfg.App.Environment)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := wire.InitializeMonolithDeps(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}
	defer deps.PostgresPool.Close()
	slog.Info("Dependencies initialized")

	r := createRouter(deps)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		slog.Info("Server listening", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}

func createRouter(d *wire.MonolithDeps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(telemetry.HTTPMiddleware("handloom-monolith"))
	r.Use(telemetry.TraceIDMiddleware)

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger())
	r.Use(middleware.Recoverer())
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Compress(5))

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(100, time.Minute))
			r.Mount("/pricing", d.PricingHandler.PublicRoutes())
		})
	})

	r.Route("/api/v1/store", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(30, time.Minute))
			r.Mount("/auth", d.StoreAuthHandler.Routes(d.CustomerAuthMiddleware.Authenticate))
		})
		r.Mount("/catalog", d.StoreCatalogHandler.Routes())
		r.Mount("/track", d.StoreTrackingHandler.Routes())
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(60, time.Minute))
			r.Mount("/events", d.StoreEventsHandler.Routes())
		})

		r.Mount("/webhooks", d.StoreWebhookHandler.Routes())

		r.Group(func(r chi.Router) {
			r.Use(d.OptionalCartAuth.Resolve)
			r.Mount("/cart", d.StoreCartHandler.CRUDRoutes())
		})

		r.Group(func(r chi.Router) {
			r.Use(d.CustomerAuthMiddleware.Authenticate)
			r.Mount("/me", d.StoreProfileHandler.Routes())
			r.Mount("/checkout", d.StoreCheckoutHandler.Routes())
			r.Mount("/orders", d.StoreOrderHandler.Routes())
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(20, time.Minute))
			r.Mount("/auth", d.AuthHandler.Routes(d.AuthMiddleware.Authenticate))
		})

		r.Group(func(r chi.Router) {
			r.Use(d.AuthMiddleware.Authenticate)

			r.Group(func(r chi.Router) {
				r.Use(d.AuthMiddleware.RequireRole(domain.UserRoleAdmin))
				r.Mount("/users", d.UserHandler.Routes())
			})

			r.Mount("/categories", d.CategoryHandler.Routes())
			r.Mount("/products", d.ProductHandler.Routes())
			r.Mount("/inventory", d.InventoryHandler.Routes())
			r.Mount("/pricing", d.PricingHandler.Routes())
			r.Mount("/orders", d.OrderHandler.Routes())
			r.Mount("/customers", d.CustomerHandler.Routes())

			r.Group(func(r chi.Router) {
				r.Use(d.AuthMiddleware.RequireRole(domain.UserRoleAdmin))
				r.Mount("/audit", auditRoutes(d.AuditHandler))
			})

			r.Mount("/notifications", d.NotificationHandler.Routes())
			r.Mount("/coupons", d.CouponHandler.Routes())
			r.Mount("/assets", d.AssetHandler.Routes())
			r.Mount("/reports", d.ReportHandler.Routes())
		})
	})

	return r
}

func auditRoutes(h *handler.AuditHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Get("/entity/{type}/{id}", h.GetByEntity)
	r.Get("/user/{id}", h.GetByUser)
	return r
}
