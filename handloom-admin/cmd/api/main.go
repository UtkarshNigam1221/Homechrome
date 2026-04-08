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

	"github.com/handloom/admin/internal/cache"
	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	eventhandlers "github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/gateway/shiprocket"
	"github.com/handloom/admin/internal/gateway/sms"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
	"github.com/handloom/admin/pkg/slogx"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize structured logger
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting handloom-admin API server", "environment", cfg.App.Environment)

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize DynamoDB client", "error", err)
		os.Exit(1)
	}
	slog.Info("DynamoDB client initialized")

	// Initialize S3 client
	s3c, err := s3client.New(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
	if err != nil {
		slog.Error("Failed to initialize S3 client", "error", err)
		os.Exit(1)
	}
	slog.Info("S3 client initialized")

	// Initialize PostgreSQL pool for catalog data
	pgPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		slog.Error("Failed to initialize PostgreSQL pool", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()
	slog.Info("PostgreSQL pool initialized")

	// Initialize repositories
	catalogCache := cache.New(5*time.Minute, 10*time.Minute)
	userRepo := dynamodb.NewUserRepository(dbClient)
	categoryRepo := postgres.NewCachedCategoryRepository(
		postgres.NewCategoryRepository(pgPool), catalogCache,
	)
	productRepo := postgres.NewCachedProductRepository(
		postgres.NewProductRepository(pgPool), catalogCache,
	)
	pricingRuleRepo := dynamodb.NewPricingRuleRepository(dbClient)
	priceQuoteRepo := dynamodb.NewPriceQuoteRepository(dbClient)
	inventoryRepo := postgres.NewInventoryRepository(pgPool)
	orderRepo := dynamodb.NewOrderRepository(dbClient)
	customerRepo := dynamodb.NewCustomerRepository(dbClient)
	auditRepo := dynamodb.NewAuditRepository(dbClient)
	tokenStore := dynamodb.NewTokenStore(dbClient)
	notificationRepo := dynamodb.NewNotificationRepository(dbClient)
	couponRepo := dynamodb.NewCouponRepository(dbClient)
	analyticsRepo := dynamodb.NewAnalyticsRepository(dbClient)
	reportRepo := dynamodb.NewReportRepository(dbClient)

	// B2C repositories
	otpRepo := dynamodb.NewOTPRepository(dbClient)
	customerTokenStore := dynamodb.NewCustomerTokenStore(dbClient)
	cartRepo := dynamodb.NewCartRepository(dbClient)
	paymentRepo := dynamodb.NewPaymentRepository(dbClient)
	shipmentRepo := dynamodb.NewShipmentRepository(dbClient)
	eventsRepo := dynamodb.NewEventsRepository(dbClient)

	// Initialize event handlers and publisher (LocalPublisher for monolith dev mode)
	notifEventHandler := eventhandlers.NewNotificationHandler()
	reportEventHandler := eventhandlers.NewReportHandler()
	analyticsAggregator := service.NewAnalyticsAggregator(eventsRepo, analyticsRepo)
	analyticsEventHandler := eventhandlers.NewAnalyticsHandler(eventsRepo, analyticsRepo, analyticsAggregator)
	auditEventHandler := eventhandlers.NewAuditHandler()
	publisher := event.NewLocalPublisher(notifEventHandler, reportEventHandler, analyticsEventHandler, auditEventHandler)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		tokenStore,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)

	userService := service.NewUserService(userRepo, tokenStore)
	assetService := service.NewAssetService(s3c, cfg.AWS.S3Bucket, cfg.AWS.Region, cfg.AWS.CDNUrl, cfg.AWS.Endpoint)
	inventoryService := service.NewInventoryService(inventoryRepo, catalogCache, publisher)
	categoryService := service.NewCategoryService(categoryRepo, productRepo, assetService)
	productService := service.NewProductService(
		productRepo,
		categoryRepo,
		inventoryRepo,
		assetService,
		publisher,
	)
	pricingService := service.NewPricingService(
		pricingRuleRepo,
		priceQuoteRepo,
		categoryRepo,
		productRepo,
		cfg.App.QuoteValidityHrs,
	)
	orderService := service.NewOrderService(
		orderRepo,
		customerRepo,
		productRepo,
		inventoryRepo,
		priceQuoteRepo,
		pricingService,
	)
	customerService := service.NewCustomerService(customerRepo, orderRepo)
	auditService := service.NewAuditService(auditRepo)
	notificationService := service.NewNotificationService(notificationRepo, userRepo)
	couponService := service.NewCouponService(couponRepo)
	analyticsService := service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo)
	reportService := service.NewReportService(reportRepo, orderService, productService, customerService, inventoryService, analyticsService)

	// Gateway clients
	var smsGateway interface {
		SendOTP(ctx context.Context, phone, code string) error
	}
	if cfg.Store.MSG91AuthKey == "" || cfg.Store.MSG91OTPTemplateID == "" {
		smsGateway = sms.NewDevClient()
		slog.Info("Using dev SMS gateway (OTPs printed to console)")
	} else {
		smsGateway = sms.NewClient(sms.Config{
			AuthKey:       cfg.Store.MSG91AuthKey,
			OTPTemplateID: cfg.Store.MSG91OTPTemplateID,
			BaseURL:       cfg.Store.MSG91BaseURL,
		})
	}

	var phonePeClient phonepe.Gateway
	if cfg.Store.PhonePeClientID == "" || cfg.Store.PhonePeClientSecret == "" {
		phonePeClient = phonepe.NewDevClient(cfg.Store.PhonePeRedirectURL)
		slog.Info("Using dev PhonePe gateway (mock payments)")
	} else {
		phonePeClient = phonepe.NewClient(phonepe.Config{
			ClientID:      cfg.Store.PhonePeClientID,
			ClientSecret:  cfg.Store.PhonePeClientSecret,
			ClientVersion: cfg.Store.PhonePeClientVersion,
			BaseURL:       cfg.Store.PhonePeBaseURL,
			CallbackURL:   cfg.Store.PhonePeCallbackURL,
			RedirectURL:   cfg.Store.PhonePeRedirectURL,
		})
	}

	var shiprocketClient shiprocket.Gateway
	if cfg.Store.ShiprocketEmail == "" || cfg.Store.ShiprocketPassword == "" {
		shiprocketClient = shiprocket.NewDevClient()
		slog.Info("Using dev Shiprocket gateway (mock courier data)")
	} else {
		shiprocketClient = shiprocket.NewClient(shiprocket.Config{
			Email:         cfg.Store.ShiprocketEmail,
			Password:      cfg.Store.ShiprocketPassword,
			BaseURL:       cfg.Store.ShiprocketBaseURL,
			PickupPincode: cfg.Store.ShiprocketPickupPincode,
		})
	}

	// B2C services
	customerAuthService := service.NewCustomerAuthService(
		otpRepo,
		customerRepo,
		customerTokenStore,
		smsGateway,
		publisher,
		service.CustomerAuthConfig{
			JWTSecret:            cfg.Store.CustomerJWTSecret,
			AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
			RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
			Issuer:               "handloom-store",
		},
	)

	cartService := service.NewCartService(cartRepo, productRepo, inventoryRepo)
	paymentService := service.NewPaymentService(paymentRepo, orderRepo, inventoryRepo, cartService, phonePeClient, publisher)
	shippingService := service.NewShippingService(shipmentRepo, orderRepo, shiprocketClient, cfg.Store.ShiprocketPickupPincode)
	checkoutService := service.NewCheckoutService(cartService, orderRepo, paymentService, shippingService, inventoryRepo, customerRepo, publisher)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, userService, validation)
	userHandler := handler.NewUserHandler(userService, validation)
	categoryHandler := handler.NewCategoryHandler(categoryService, validation)
	productHandler := handler.NewProductHandler(productService, inventoryService, validation)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	pricingHandler := handler.NewPricingHandler(pricingService, validation)
	orderHandler := handler.NewOrderHandler(orderService, paymentService, validation)
	customerHandler := handler.NewCustomerHandler(customerService, validation)
	auditHandler := handler.NewAuditHandler(auditService)
	notificationHandler := handler.NewNotificationHandler(notificationService, validation)
	couponHandler := handler.NewCouponHandler(couponService, validation)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	assetHandler := handler.NewAssetHandler(assetService, validation)
	reportHandler := handler.NewReportHandler(reportService, validation)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService)

	// B2C handlers
	storeAuthHandler := store.NewAuthHandler(customerAuthService, cartService, validation)
	storeCatalogHandler := store.NewCatalogHandler(productService, categoryService, inventoryService)
	storeCartHandler := store.NewCartHandler(cartService, validation)
	storeCheckoutHandler := store.NewCheckoutHandler(checkoutService, validation)
	storeOrderHandler := store.NewOrderHandler(orderService, orderRepo)
	storeTrackingHandler := store.NewTrackingHandler(orderRepo, shipmentRepo)
	storeProfileHandler := store.NewProfileHandler(customerRepo, validation)
	storeWebhookHandler := store.NewWebhookHandler(paymentService, phonePeClient, cfg.Store.PhonePeWebhookUsername, cfg.Store.PhonePeWebhookPassword)
	storeEventsHandler := store.NewEventsHandler(eventsRepo, analyticsRepo, validation)

	// Customer auth middleware
	customerAuthMiddleware := middleware.NewCustomerAuth(customerAuthService)
	optionalCartAuth := middleware.NewOptionalCartAuth(customerAuthService)

	// Create router
	r := createRouter(cfg, authMiddleware,
		authHandler, userHandler, categoryHandler,
		productHandler, inventoryHandler, pricingHandler, orderHandler,
		customerHandler, auditHandler, notificationHandler, couponHandler,
		analyticsHandler, assetHandler, reportHandler,
		// B2C store handlers
		storeAuthHandler, storeCatalogHandler, storeCartHandler,
		storeCheckoutHandler, storeOrderHandler, storeTrackingHandler,
		storeProfileHandler, storeWebhookHandler, storeEventsHandler,
		customerAuthMiddleware, optionalCartAuth,
	)

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
		slog.Info("Server listening", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}

func createRouter(
	cfg *config.Config,
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
	analyticsHandler *handler.AnalyticsHandler,
	assetHandler *handler.AssetHandler,
	reportHandler *handler.ReportHandler,
	// B2C store handlers
	storeAuthHandler *store.AuthHandler,
	storeCatalogHandler *store.CatalogHandler,
	storeCartHandler *store.CartHandler,
	storeCheckoutHandler *store.CheckoutHandler,
	storeOrderHandler *store.OrderHandler,
	storeTrackingHandler *store.TrackingHandler,
	storeProfileHandler *store.ProfileHandler,
	storeWebhookHandler *store.WebhookHandler,
	storeEventsHandler *store.EventsHandler,
	customerAuthMiddleware *middleware.CustomerAuth,
	optionalCartAuth *middleware.OptionalCartAuth,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger())
	r.Use(middleware.Recoverer())
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

	// B2C Store routes
	r.Route("/api/v1/store", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(30, time.Minute))
			r.Mount("/auth", storeAuthHandler.Routes(customerAuthMiddleware.Authenticate))
		})
		r.Mount("/catalog", storeCatalogHandler.Routes())
		r.Mount("/track", storeTrackingHandler.Routes())
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(60, time.Minute))
			r.Mount("/events", storeEventsHandler.Routes())
		})

		// Webhook routes (signature-verified, not customer auth)
		r.Mount("/webhooks", storeWebhookHandler.Routes())

		// Guest-accessible cart routes
		r.Group(func(r chi.Router) {
			r.Use(optionalCartAuth.Resolve)
			r.Mount("/cart", storeCartHandler.CRUDRoutes())
		})

		// Customer-authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(customerAuthMiddleware.Authenticate)
			r.Mount("/me", storeProfileHandler.Routes())
			r.Mount("/checkout", storeCheckoutHandler.Routes())
			r.Mount("/orders", storeOrderHandler.Routes())
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
