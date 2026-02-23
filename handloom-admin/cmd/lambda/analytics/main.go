// Package main is the Lambda entry point for the Analytics service
package main

import (
	"context"
	"os"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Analytics Lambda")

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
	analyticsRepo := dynamodb.NewAnalyticsRepository(dbClient)
	orderRepo := dynamodb.NewOrderRepository(dbClient)
	productRepo := postgres.NewProductRepository(pgPool)
	inventoryRepo := postgres.NewInventoryRepository(pgPool)

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
	analyticsService := service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo, log)

	// Initialize handler
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService, log)

	// Create authenticated router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewAuthenticatedRouter(routerCfg, log, authMiddleware)

	// Register routes
	router.NewAnalyticsRouter(r, analyticsHandler)

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
