// Package main is the entry point for the API server
package main

import (
	"context"
	"fmt"
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
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.App.Debug)
	log.Infof("Starting handloom-admin API server in %s mode", cfg.App.Environment)

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}
	log.Info("DynamoDB client initialized")

	// Initialize S3 client
	s3c, err := s3client.New(ctx, cfg.AWS.Region)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
	log.Info("S3 client initialized")

	// Initialize repositories
	userRepo := dynamodb.NewUserRepository(dbClient)
	categoryRepo := dynamodb.NewCategoryRepository(dbClient)
	productRepo := dynamodb.NewProductRepository(dbClient)
	pricingRuleRepo := dynamodb.NewPricingRuleRepository(dbClient)
	priceQuoteRepo := dynamodb.NewPriceQuoteRepository(dbClient)
	inventoryRepo := dynamodb.NewInventoryRepository(dbClient)
	orderRepo := dynamodb.NewOrderRepository(dbClient)
	customerRepo := dynamodb.NewCustomerRepository(dbClient)
	auditRepo := dynamodb.NewAuditRepository(dbClient)
	tokenStore := dynamodb.NewTokenStore(dbClient)
	notificationRepo := dynamodb.NewNotificationRepository(dbClient)
	couponRepo := dynamodb.NewCouponRepository(dbClient)
	artisanRepo := dynamodb.NewArtisanRepository(dbClient)
	analyticsRepo := dynamodb.NewAnalyticsRepository(dbClient)
	reportRepo := dynamodb.NewReportRepository(dbClient)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		tokenStore,
		log,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)

	userService := service.NewUserService(userRepo, tokenStore, log)
	assetService := service.NewAssetService(log, s3c, cfg.AWS.S3Bucket)
	categoryService := service.NewCategoryService(categoryRepo, productRepo, assetService, log)
	inventoryService := service.NewInventoryService(inventoryRepo, productRepo, log)
	productService := service.NewProductService(
		productRepo,
		categoryRepo,
		inventoryRepo,
		assetService,
		log,
	)
	pricingService := service.NewPricingService(
		pricingRuleRepo,
		priceQuoteRepo,
		categoryRepo,
		productRepo,
		log,
		cfg.App.QuoteValidityHrs,
	)
	orderService := service.NewOrderService(
		orderRepo,
		customerRepo,
		productRepo,
		inventoryRepo,
		priceQuoteRepo,
		pricingService,
		log,
	)
	customerService := service.NewCustomerService(customerRepo, orderRepo, log)
	auditService := service.NewAuditService(auditRepo, log)
	notificationService := service.NewNotificationService(notificationRepo, userRepo, log)
	couponService := service.NewCouponService(couponRepo, log)
	artisanService := service.NewArtisanService(artisanRepo, log)
	analyticsService := service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo, log)
	reportService := service.NewReportService(reportRepo, orderService, productService, customerService, inventoryService, analyticsService, log)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, userService, log, validation)
	userHandler := handler.NewUserHandler(userService, log, validation)
	categoryHandler := handler.NewCategoryHandler(categoryService, log, validation)
	productHandler := handler.NewProductHandler(productService, inventoryService, log, validation)
	inventoryHandler := handler.NewInventoryHandler(inventoryService, log)
	pricingHandler := handler.NewPricingHandler(pricingService, log, validation)
	orderHandler := handler.NewOrderHandler(orderService, log, validation)
	customerHandler := handler.NewCustomerHandler(customerService, log, validation)
	auditHandler := handler.NewAuditHandler(auditService)
	notificationHandler := handler.NewNotificationHandler(notificationService, validation)
	couponHandler := handler.NewCouponHandler(couponService, validation)
	artisanHandler := handler.NewArtisanHandler(artisanService, validation)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	assetHandler := handler.NewAssetHandler(assetService, validation)
	reportHandler := handler.NewReportHandler(reportService, validation)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService, log)

	// Create router
	r := createRouter(cfg, log, authMiddleware,
		authHandler, userHandler, categoryHandler,
		productHandler, inventoryHandler, pricingHandler, orderHandler,
		customerHandler, auditHandler, notificationHandler, couponHandler,
		artisanHandler, analyticsHandler, assetHandler, reportHandler)

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Server listening on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}

func createRouter(
	cfg *config.Config,
	log *logger.Logger,
	authMiddleware *middleware.Auth,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	categoryHandler *handler.CategoryHandler,
	productHandler *handler.ProductHandler,
	inventoryHandler *handler.InventoryHandler,
	pricingHandler *handler.PricingHandler,
	orderHandler *handler.OrderHandler,
	customerHandler *handler.CustomerHandler,
	auditHandler *handler.AuditHandler,
	notificationHandler *handler.NotificationHandler,
	couponHandler *handler.CouponHandler,
	artisanHandler *handler.ArtisanHandler,
	analyticsHandler *handler.AnalyticsHandler,
	assetHandler *handler.AssetHandler,
	reportHandler *handler.ReportHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recoverer(log))
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Compress(5))

	// CORS — reflect origin to support credentials (cookies)
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// API v1 routes (public)
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes with rate limiting
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(100, time.Minute))

			// Pricing calculation (public/B2C)
			r.Mount("/pricing", pricingHandler.PublicRoutes())
		})
	})

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		// Auth routes (public + protected, split inside the handler)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(20, time.Minute))
			r.Mount("/auth", authHandler.Routes(authMiddleware.Authenticate))
		})

		// Protected admin routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Users (admin only)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
				r.Mount("/users", userHandler.Routes())
			})

			// Categories
			r.Mount("/categories", categoryHandler.Routes())

			// Products (includes inventory)
			r.Mount("/products", productHandler.Routes())

			// Inventory (low stock alerts)
			r.Mount("/inventory", inventoryHandler.Routes())

			// Pricing Rules
			r.Mount("/pricing", pricingHandler.Routes())

			// Orders
			r.Mount("/orders", orderHandler.Routes())

			// Customers
			r.Mount("/customers", customerHandler.Routes())

			// Audit Logs (admin only)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
				r.Mount("/audit", auditRoutes(auditHandler))
			})

			// Notifications
			r.Mount("/notifications", notificationHandler.Routes())

			// Coupons
			r.Mount("/coupons", couponHandler.Routes())

			// Artisans
			r.Mount("/artisans", artisanHandler.Routes())

			// Analytics/Dashboard
			r.Mount("/analytics", analyticsRoutes(analyticsHandler))

			// Assets/Media
			r.Mount("/assets", assetHandler.Routes())

			// Reports
			r.Mount("/reports", reportHandler.Routes())
		})
	})

	return r
}

// Audit routes
func auditRoutes(h *handler.AuditHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Get("/entity/{type}/{id}", h.GetByEntity)
	r.Get("/user/{id}", h.GetByUser)
	return r
}

// Analytics routes
func analyticsRoutes(h *handler.AnalyticsHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/dashboard", h.GetDashboardStats)
	r.Get("/sales", h.GetSalesAnalytics)
	r.Get("/top-products", h.GetTopProducts)
	r.Get("/top-categories", h.GetTopCategories)
	r.Get("/customers", h.GetCustomerAnalytics)
	r.Get("/inventory", h.GetInventoryAnalytics)
	return r
}

