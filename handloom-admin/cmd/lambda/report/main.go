// Package main is the Lambda entry point for the Report service
package main

import (
	"context"
	"os"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/router"
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
	log.Info("Starting Report Lambda")

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
	reportRepo := dynamodb.NewReportRepository(dbClient)
	orderRepo := dynamodb.NewOrderRepository(dbClient)
	productRepo := dynamodb.NewProductRepository(dbClient)
	customerRepo := dynamodb.NewCustomerRepository(dbClient)
	inventoryRepo := dynamodb.NewInventoryRepository(dbClient)
	analyticsRepo := dynamodb.NewAnalyticsRepository(dbClient)
	categoryRepo := dynamodb.NewCategoryRepository(dbClient)
	priceQuoteRepo := dynamodb.NewPriceQuoteRepository(dbClient)
	pricingRuleRepo := dynamodb.NewPricingRuleRepository(dbClient)

	// Initialize S3 client
	s3c, err := s3client.New(ctx, cfg.AWS.Region)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

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
	assetService := service.NewAssetService(log, s3c, cfg.AWS.S3Bucket)
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
	analyticsService := service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo, log)
	reportService := service.NewReportService(reportRepo, orderService, productService, customerService, inventoryService, analyticsService, log)

	// Initialize validation middleware
	v := validator.New()
	validation := middleware.NewValidation(v, middleware.ValidationConfig{})

	// Initialize handler
	reportHandler := handler.NewReportHandler(reportService, validation)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService, log)

	// Create authenticated router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewAuthenticatedRouter(routerCfg, log, authMiddleware)

	// Register routes
	router.NewReportRouter(r, reportHandler)

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
