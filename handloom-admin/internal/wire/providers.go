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
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
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

// ProvideS3Client creates a new S3 client
func ProvideS3Client(ctx context.Context, cfg *config.Config) (*s3client.S3Client, error) {
	return s3client.New(ctx, cfg.AWS.Region)
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
	tokenStore domain.TokenStore,
	log *logger.Logger,
) *service.UserService {
	return service.NewUserService(userRepo, tokenStore, log)
}

// ProvideCategoryService creates a new CategoryService
func ProvideCategoryService(
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	assetService *service.AssetService,
	log *logger.Logger,
) *service.CategoryService {
	return service.NewCategoryService(categoryRepo, productRepo, assetService, log)
}

// ProvideProductService creates a new ProductService
func ProvideProductService(
	productRepo domain.ProductRepository,
	categoryRepo domain.CategoryRepository,
	inventoryRepo domain.InventoryRepository,
	assetService *service.AssetService,
	log *logger.Logger,
) *service.ProductService {
	return service.NewProductService(productRepo, categoryRepo, inventoryRepo, assetService, log)
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

// ProvideAssetService creates a new AssetService
func ProvideAssetService(
	log *logger.Logger,
	s3Client *s3client.S3Client,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(log, s3Client, cfg.AWS.S3Bucket)
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
	ProvideProductService,
	ProvideInventoryService,
	ProvideOrderService,
	ProvideCustomerService,
	ProvidePricingService,
	ProvideAnalyticsService,
	ProvideNotificationService,
	ProvideCouponService,
	ProvideArtisanService,
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
	userService *service.UserService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.AuthHandler {
	return handler.NewAuthHandler(authService, userService, log, validation)
}

// ProvideUserHandler creates a new UserHandler
func ProvideUserHandler(
	userService *service.UserService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.UserHandler {
	return handler.NewUserHandler(userService, log, validation)
}

// ProvideCategoryHandler creates a new CategoryHandler
func ProvideCategoryHandler(
	categoryService *service.CategoryService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.CategoryHandler {
	return handler.NewCategoryHandler(categoryService, log, validation)
}

// ProvideProductHandler creates a new ProductHandler
func ProvideProductHandler(
	productService *service.ProductService,
	inventoryService *service.InventoryService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.ProductHandler {
	return handler.NewProductHandler(productService, inventoryService, log, validation)
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
	validation *middleware.Validation,
) *handler.OrderHandler {
	return handler.NewOrderHandler(orderService, log, validation)
}

// ProvideCustomerHandler creates a new CustomerHandler
func ProvideCustomerHandler(
	customerService *service.CustomerService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.CustomerHandler {
	return handler.NewCustomerHandler(customerService, log, validation)
}

// ProvidePricingHandler creates a new PricingHandler
func ProvidePricingHandler(
	pricingService *service.PricingService,
	log *logger.Logger,
	validation *middleware.Validation,
) *handler.PricingHandler {
	return handler.NewPricingHandler(pricingService, log, validation)
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
	validation *middleware.Validation,
) *handler.NotificationHandler {
	return handler.NewNotificationHandler(notificationService, validation)
}

// ProvideCouponHandler creates a new CouponHandler
func ProvideCouponHandler(
	couponService *service.CouponService,
	validation *middleware.Validation,
) *handler.CouponHandler {
	return handler.NewCouponHandler(couponService, validation)
}

// ProvideArtisanHandler creates a new ArtisanHandler
func ProvideArtisanHandler(
	artisanService *service.ArtisanService,
	validation *middleware.Validation,
) *handler.ArtisanHandler {
	return handler.NewArtisanHandler(artisanService, validation)
}

// ProvideAssetHandler creates a new AssetHandler
func ProvideAssetHandler(
	assetService *service.AssetService,
	validation *middleware.Validation,
) *handler.AssetHandler {
	return handler.NewAssetHandler(assetService, validation)
}

// ProvideReportHandler creates a new ReportHandler
func ProvideReportHandler(
	reportService *service.ReportService,
	validation *middleware.Validation,
) *handler.ReportHandler {
	return handler.NewReportHandler(reportService, validation)
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
	ProvideProductHandler,
	ProvideInventoryHandler,
	ProvideOrderHandler,
	ProvideCustomerHandler,
	ProvidePricingHandler,
	ProvideAnalyticsHandler,
	ProvideNotificationHandler,
	ProvideCouponHandler,
	ProvideArtisanHandler,
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

// ProvideValidation creates the Validation middleware
func ProvideValidation(v *validator.Service) *middleware.Validation {
	return middleware.NewValidation(v, middleware.ValidationConfig{})
}

// ProvideValidator creates the validator service
func ProvideValidator() *validator.Service {
	return validator.New()
}

// MiddlewareSet contains all middleware providers
var MiddlewareSet = wire.NewSet(
	ProvideAuthMiddleware,
	ProvideValidator,
	ProvideValidation,
)
