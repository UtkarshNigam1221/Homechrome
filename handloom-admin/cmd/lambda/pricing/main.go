// Package main is the Lambda entry point for the Pricing service
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
	"github.com/handloom/admin/pkg/slogx"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize structured logger
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting Pricing Lambda")

	// Initialize context
	ctx := context.Background()

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize DynamoDB client", "error", err)
		os.Exit(1)
	}

	// Initialize PostgreSQL pool for catalog data
	pgPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		slog.Error("Failed to initialize PostgreSQL pool", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	// Initialize repositories
	userRepo := dynamodb.NewUserRepository(dbClient)
	tokenStore := dynamodb.NewTokenStore(dbClient)
	pricingRuleRepo := dynamodb.NewPricingRuleRepository(dbClient)
	priceQuoteRepo := dynamodb.NewPriceQuoteRepository(dbClient)
	categoryRepo := postgres.NewCategoryRepository(pgPool)
	productRepo := postgres.NewProductRepository(pgPool)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		tokenStore,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)
	pricingService := service.NewPricingService(
		pricingRuleRepo,
		priceQuoteRepo,
		categoryRepo,
		productRepo,
		cfg.App.QuoteValidityHrs,
	)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handler
	pricingHandler := handler.NewPricingHandler(pricingService, validation)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService)

	// Create router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewBaseRouter(routerCfg, true)

	// Public pricing routes (no auth required) - for B2C
	r.Route("/api/v1/pricing", func(r chi.Router) {
		r.Mount("/", pricingHandler.PublicRoutes())
	})

	// Admin pricing routes (with auth)
	r.Route("/admin/pricing", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Mount("/", pricingHandler.Routes())
	})

	// Start Lambda
	adapter := router.NewLambdaAdapter(r)
	adapter.Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
