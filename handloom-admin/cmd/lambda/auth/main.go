// Package main is the Lambda entry point for the Auth service
package main

import (
	"context"
	"os"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
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
	log.Info("Starting Auth Lambda")

	// Initialize context
	ctx := context.Background()

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}

	// Initialize repositories
	userRepo := dynamodb.NewUserRepository(dbClient)
	tokenStore := dynamodb.NewTokenStore(dbClient)

	// Initialize service
	authService := service.NewAuthService(
		userRepo,
		tokenStore,
		log,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handler
	authHandler := handler.NewAuthHandler(authService, log, validation)

	// Create router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewBaseRouter(routerCfg, log, true) // true = add health check

	// Register routes
	router.NewAuthRouter(r, authHandler)

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
