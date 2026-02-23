// Package main is the Lambda entry point for the Pricing service
package main

import (
	"context"
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
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Pricing Lambda")

	// Initialize context
	ctx := context.Background()

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}

	// Initialize PostgreSQL pool for catalog data
	pgPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL pool: %v", err)
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
		log,
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
		log,
		cfg.App.QuoteValidityHrs,
	)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handler
	pricingHandler := handler.NewPricingHandler(pricingService, log, validation)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService, log)

	// Create router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewBaseRouter(routerCfg, log, true)

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
