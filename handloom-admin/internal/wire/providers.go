// Package wire provides dependency injection using Google Wire
package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/logger"
)

// ============================================================================
// CORE PROVIDERS - Shared across all services
// ============================================================================

// ProvideLogger creates a new logger
func ProvideLogger(cfg *config.Config) *logger.Logger {
	return logger.New(cfg.App.Debug)
}

// ProvideDynamoDBClient creates a new DynamoDB client
func ProvideDynamoDBClient(ctx context.Context, cfg *config.Config) (*dynamodb.Client, error) {
	return dynamodb.NewClient(ctx, cfg)
}

// CoreSet contains core providers used by all services
var CoreSet = wire.NewSet(
	ProvideLogger,
	ProvideDynamoDBClient,
)

// ============================================================================
// REPOSITORY PROVIDERS
// ============================================================================

// ProvideUserRepository creates a new UserRepository
func ProvideUserRepository(client *dynamodb.Client) domain.UserRepository {
	return dynamodb.NewUserRepository(client)
}

// ProvideTokenStore creates a new TokenStore
func ProvideTokenStore(client *dynamodb.Client) domain.TokenStore {
	return dynamodb.NewTokenStore(client)
}

// ProvideCategoryRepository creates a new CategoryRepository
func ProvideCategoryRepository(client *dynamodb.Client) domain.CategoryRepository {
	return dynamodb.NewCategoryRepository(client)
}

// ProvideDesignRepository creates a new DesignRepository
func ProvideDesignRepository(client *dynamodb.Client) domain.DesignRepository {
	return dynamodb.NewDesignRepository(client)
}

// ProvideProductRepository creates a new ProductRepository
func ProvideProductRepository(client *dynamodb.Client) domain.ProductRepository {
	return dynamodb.NewProductRepository(client)
}

// ProvideInventoryRepository creates a new InventoryRepository
func ProvideInventoryRepository(client *dynamodb.Client) domain.InventoryRepository {
	return dynamodb.NewInventoryRepository(client)
}

// ProvideOrderRepository creates a new OrderRepository
func ProvideOrderRepository(client *dynamodb.Client) domain.OrderRepository {
	return dynamodb.NewOrderRepository(client)
}

// ProvideCustomerRepository creates a new CustomerRepository
func ProvideCustomerRepository(client *dynamodb.Client) domain.CustomerRepository {
	return dynamodb.NewCustomerRepository(client)
}

// ProvidePricingRuleRepository creates a new PricingRuleRepository
func ProvidePricingRuleRepository(client *dynamodb.Client) domain.PricingRuleRepository {
	return dynamodb.NewPricingRuleRepository(client)
}

// ProvidePriceQuoteRepository creates a new PriceQuoteRepository
func ProvidePriceQuoteRepository(client *dynamodb.Client) domain.PriceQuoteRepository {
	return dynamodb.NewPriceQuoteRepository(client)
}

// ProvideAnalyticsRepository creates a new AnalyticsRepository
func ProvideAnalyticsRepository(client *dynamodb.Client) domain.AnalyticsRepository {
	return dynamodb.NewAnalyticsRepository(client)
}

// ProvideNotificationRepository creates a new NotificationRepository
func ProvideNotificationRepository(client *dynamodb.Client) domain.NotificationRepository {
	return dynamodb.NewNotificationRepository(client)
}

// ProvideCouponRepository creates a new CouponRepository
func ProvideCouponRepository(client *dynamodb.Client) domain.CouponRepository {
	return dynamodb.NewCouponRepository(client)
}

// ProvideArtisanRepository creates a new ArtisanRepository
func ProvideArtisanRepository(client *dynamodb.Client) domain.ArtisanRepository {
	return dynamodb.NewArtisanRepository(client)
}

// ProvideBulkOperationRepository creates a new BulkOperationRepository
func ProvideBulkOperationRepository(client *dynamodb.Client) domain.BulkOperationRepository {
	return dynamodb.NewBulkOperationRepository(client)
}

// ProvideAssetRepository creates a new AssetRepository
func ProvideAssetRepository(client *dynamodb.Client) domain.AssetRepository {
	return dynamodb.NewAssetRepository(client)
}

// ProvideReportRepository creates a new ReportRepository
func ProvideReportRepository(client *dynamodb.Client) domain.ReportRepository {
	return dynamodb.NewReportRepository(client)
}

// ProvideAuditRepository creates a new AuditRepository
func ProvideAuditRepository(client *dynamodb.Client) domain.AuditRepository {
	return dynamodb.NewAuditRepository(client)
}

// RepositorySet contains all repository providers
var RepositorySet = wire.NewSet(
	ProvideUserRepository,
	ProvideTokenStore,
	ProvideCategoryRepository,
	ProvideDesignRepository,
	ProvideProductRepository,
	ProvideInventoryRepository,
	ProvideOrderRepository,
	ProvideCustomerRepository,
	ProvidePricingRuleRepository,
	ProvidePriceQuoteRepository,
	ProvideAnalyticsRepository,
	ProvideNotificationRepository,
	ProvideCouponRepository,
	ProvideArtisanRepository,
	ProvideBulkOperationRepository,
	ProvideAssetRepository,
	ProvideReportRepository,
	ProvideAuditRepository,
)

// ============================================================================
// SERVICE PROVIDERS
// ============================================================================

// ProvideAuthService creates a new AuthService
func ProvideAuthService(
	userRepo domain.UserRepository,
	tokenStore domain.TokenStore,
	log *logger.Logger,
	cfg *config.Config,
) *service.AuthService {
	return service.NewAuthService(
		userRepo,
		tokenStore,
		log,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)
}

// ProvideUserService creates a new UserService
func ProvideUserService(
	userRepo domain.UserRepository,
	log *logger.Logger,
) *service.UserService {
	return service.NewUserService(userRepo, log)
}

// ProvideCategoryService creates a new CategoryService
func ProvideCategoryService(
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	log *logger.Logger,
) *service.CategoryService {
	return service.NewCategoryService(categoryRepo, productRepo, log)
}

// ProvideDesignService creates a new DesignService
func ProvideDesignService(
	designRepo domain.DesignRepository,
	categoryRepo domain.CategoryRepository,
	log *logger.Logger,
) *service.DesignService {
	return service.NewDesignService(designRepo, categoryRepo, log)
}

// ProvideProductService creates a new ProductService
func ProvideProductService(
	productRepo domain.ProductRepository,
	categoryRepo domain.CategoryRepository,
	designRepo domain.DesignRepository,
	inventoryRepo domain.InventoryRepository,
	log *logger.Logger,
) *service.ProductService {
	return service.NewProductService(productRepo, categoryRepo, designRepo, inventoryRepo, log)
}

// ProvideInventoryService creates a new InventoryService
func ProvideInventoryService(
	inventoryRepo domain.InventoryRepository,
	productRepo domain.ProductRepository,
	log *logger.Logger,
) *service.InventoryService {
	return service.NewInventoryService(inventoryRepo, productRepo, log)
}

// ProvidePricingService creates a new PricingService
func ProvidePricingService(
	pricingRuleRepo domain.PricingRuleRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	log *logger.Logger,
	cfg *config.Config,
) *service.PricingService {
	return service.NewPricingService(
		pricingRuleRepo,
		priceQuoteRepo,
		categoryRepo,
		productRepo,
		log,
		cfg.App.QuoteValidityHrs,
	)
}

// ProvideOrderService creates a new OrderService
func ProvideOrderService(
	orderRepo domain.OrderRepository,
	customerRepo domain.CustomerRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	pricingService *service.PricingService,
	log *logger.Logger,
) *service.OrderService {
	return service.NewOrderService(orderRepo, customerRepo, productRepo, inventoryRepo, priceQuoteRepo, pricingService, log)
}

// ProvideCustomerService creates a new CustomerService
func ProvideCustomerService(
	customerRepo domain.CustomerRepository,
	orderRepo domain.OrderRepository,
	log *logger.Logger,
) *service.CustomerService {
	return service.NewCustomerService(customerRepo, orderRepo, log)
}

// ProvideAnalyticsService creates a new AnalyticsService
func ProvideAnalyticsService(
	analyticsRepo domain.AnalyticsRepository,
	orderRepo domain.OrderRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	log *logger.Logger,
) *service.AnalyticsService {
	return service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo, log)
}

// ProvideNotificationService creates a new NotificationService
func ProvideNotificationService(
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	log *logger.Logger,
) *service.NotificationService {
	return service.NewNotificationService(notificationRepo, userRepo, log)
}

// ProvideCouponService creates a new CouponService
func ProvideCouponService(
	couponRepo domain.CouponRepository,
	log *logger.Logger,
) *service.CouponService {
	return service.NewCouponService(couponRepo, log)
}

// ProvideArtisanService creates a new ArtisanService
func ProvideArtisanService(
	artisanRepo domain.ArtisanRepository,
	log *logger.Logger,
) *service.ArtisanService {
	return service.NewArtisanService(artisanRepo, log)
}

// ProvideBulkService creates a new BulkService
func ProvideBulkService(
	bulkRepo domain.BulkOperationRepository,
	productService *service.ProductService,
	inventoryService *service.InventoryService,
	log *logger.Logger,
) *service.BulkService {
	return service.NewBulkService(bulkRepo, productService, inventoryService, log)
}

// ProvideAssetService creates a new AssetService
func ProvideAssetService(
	assetRepo domain.AssetRepository,
	log *logger.Logger,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(assetRepo, log, cfg.AWS.S3Bucket, cfg.AWS.CDNUrl)
}

// ProvideReportService creates a new ReportService
func ProvideReportService(
	reportRepo domain.ReportRepository,
	orderService *service.OrderService,
	productService *service.ProductService,
	customerService *service.CustomerService,
	inventoryService *service.InventoryService,
	analyticsService *service.AnalyticsService,
	log *logger.Logger,
) *service.ReportService {
	return service.NewReportService(reportRepo, orderService, productService, customerService, inventoryService, analyticsService, log)
}

// ProvideAuditService creates a new AuditService
func ProvideAuditService(
	auditRepo domain.AuditRepository,
	log *logger.Logger,
) *service.AuditService {
	return service.NewAuditService(auditRepo, log)
}

// ServiceSet contains all service providers
var ServiceSet = wire.NewSet(
	ProvideAuthService,
	ProvideUserService,
	ProvideCategoryService,
	ProvideDesignService,
	ProvideProductService,
	ProvideInventoryService,
	ProvideOrderService,
	ProvideCustomerService,
	ProvidePricingService,
	ProvideAnalyticsService,
	ProvideNotificationService,
	ProvideCouponService,
	ProvideArtisanService,
	ProvideBulkService,
	ProvideAssetService,
	ProvideReportService,
	ProvideAuditService,
)

// ============================================================================
// HANDLER PROVIDERS
// ============================================================================

// ProvideAuthHandler creates a new AuthHandler
func ProvideAuthHandler(
	authService *service.AuthService,
	log *logger.Logger,
) *handler.AuthHandler {
	return handler.NewAuthHandler(authService, log)
}

// ProvideUserHandler creates a new UserHandler
func ProvideUserHandler(
	userService *service.UserService,
	log *logger.Logger,
) *handler.UserHandler {
	return handler.NewUserHandler(userService, log)
}

// ProvideCategoryHandler creates a new CategoryHandler
func ProvideCategoryHandler(
	categoryService *service.CategoryService,
	log *logger.Logger,
) *handler.CategoryHandler {
	return handler.NewCategoryHandler(categoryService, log)
}

// ProvideDesignHandler creates a new DesignHandler
func ProvideDesignHandler(
	designService *service.DesignService,
	log *logger.Logger,
) *handler.DesignHandler {
	return handler.NewDesignHandler(designService, log)
}

// ProvideProductHandler creates a new ProductHandler
func ProvideProductHandler(
	productService *service.ProductService,
	inventoryService *service.InventoryService,
	log *logger.Logger,
) *handler.ProductHandler {
	return handler.NewProductHandler(productService, inventoryService, log)
}

// ProvideInventoryHandler creates a new InventoryHandler
func ProvideInventoryHandler(
	inventoryService *service.InventoryService,
	log *logger.Logger,
) *handler.InventoryHandler {
	return handler.NewInventoryHandler(inventoryService, log)
}

// ProvideOrderHandler creates a new OrderHandler
func ProvideOrderHandler(
	orderService *service.OrderService,
	log *logger.Logger,
) *handler.OrderHandler {
	return handler.NewOrderHandler(orderService, log)
}

// ProvideCustomerHandler creates a new CustomerHandler
func ProvideCustomerHandler(
	customerService *service.CustomerService,
	log *logger.Logger,
) *handler.CustomerHandler {
	return handler.NewCustomerHandler(customerService, log)
}

// ProvidePricingHandler creates a new PricingHandler
func ProvidePricingHandler(
	pricingService *service.PricingService,
	log *logger.Logger,
) *handler.PricingHandler {
	return handler.NewPricingHandler(pricingService, log)
}

// ProvideAnalyticsHandler creates a new AnalyticsHandler
func ProvideAnalyticsHandler(
	analyticsService *service.AnalyticsService,
) *handler.AnalyticsHandler {
	return handler.NewAnalyticsHandler(analyticsService)
}

// ProvideNotificationHandler creates a new NotificationHandler
func ProvideNotificationHandler(
	notificationService *service.NotificationService,
) *handler.NotificationHandler {
	return handler.NewNotificationHandler(notificationService)
}

// ProvideCouponHandler creates a new CouponHandler
func ProvideCouponHandler(
	couponService *service.CouponService,
) *handler.CouponHandler {
	return handler.NewCouponHandler(couponService)
}

// ProvideArtisanHandler creates a new ArtisanHandler
func ProvideArtisanHandler(
	artisanService *service.ArtisanService,
) *handler.ArtisanHandler {
	return handler.NewArtisanHandler(artisanService)
}

// ProvideBulkHandler creates a new BulkHandler
func ProvideBulkHandler(
	bulkService *service.BulkService,
) *handler.BulkHandler {
	return handler.NewBulkHandler(bulkService)
}

// ProvideAssetHandler creates a new AssetHandler
func ProvideAssetHandler(
	assetService *service.AssetService,
) *handler.AssetHandler {
	return handler.NewAssetHandler(assetService)
}

// ProvideReportHandler creates a new ReportHandler
func ProvideReportHandler(
	reportService *service.ReportService,
) *handler.ReportHandler {
	return handler.NewReportHandler(reportService)
}

// ProvideAuditHandler creates a new AuditHandler
func ProvideAuditHandler(
	auditService *service.AuditService,
) *handler.AuditHandler {
	return handler.NewAuditHandler(auditService)
}

// HandlerSet contains all handler providers
var HandlerSet = wire.NewSet(
	ProvideAuthHandler,
	ProvideUserHandler,
	ProvideCategoryHandler,
	ProvideDesignHandler,
	ProvideProductHandler,
	ProvideInventoryHandler,
	ProvideOrderHandler,
	ProvideCustomerHandler,
	ProvidePricingHandler,
	ProvideAnalyticsHandler,
	ProvideNotificationHandler,
	ProvideCouponHandler,
	ProvideArtisanHandler,
	ProvideBulkHandler,
	ProvideAssetHandler,
	ProvideReportHandler,
	ProvideAuditHandler,
)

// ============================================================================
// MIDDLEWARE PROVIDERS
// ============================================================================

// ProvideAuthMiddleware creates the Auth middleware
func ProvideAuthMiddleware(
	authService *service.AuthService,
	log *logger.Logger,
) *middleware.Auth {
	return middleware.NewAuth(authService, log)
}

// MiddlewareSet contains all middleware providers
var MiddlewareSet = wire.NewSet(
	ProvideAuthMiddleware,
)
