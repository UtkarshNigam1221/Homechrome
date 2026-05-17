// Package wire provides dependency injection using Google Wire
package wire

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/cache"
	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	eventhandlers "github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/internal/gateway/delhivery"
	"github.com/handloom/admin/internal/gateway/lambdainvoker"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/gateway/sms"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/handler/cron"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
)

// ============================================================================
// CORE PROVIDERS - Shared across all services
// ============================================================================

// ProvideDynamoDBClient creates a new DynamoDB client
func ProvideDynamoDBClient(ctx context.Context, cfg *config.Config) (*dynamodb.Client, error) {
	return dynamodb.NewClient(ctx, cfg)
}

// ProvideS3Client creates a new S3 client
func ProvideS3Client(ctx context.Context, cfg *config.Config) (*s3client.S3Client, error) {
	return s3client.New(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
}

// ProvidePostgresPool creates a new PostgreSQL connection pool for catalog data
func ProvidePostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	return postgres.NewPool(ctx, &cfg.Postgres)
}

// ProvideCatalogCache creates an in-process cache for catalog data.
// TTL matches the documented 2-5 min freshness window for catalog data.
func ProvideCatalogCache() *cache.Cache {
	return cache.New(5*time.Minute, 10*time.Minute)
}

// CoreSet contains core providers used by all services
var CoreSet = wire.NewSet(
	ProvideDynamoDBClient,
	ProvidePostgresPool,
	ProvideCatalogCache,
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

// ProvideCategoryRepository creates a new CategoryRepository backed by PostgreSQL with cache
func ProvideCategoryRepository(pool *pgxpool.Pool, c *cache.Cache) domain.CategoryRepository {
	return postgres.NewCachedCategoryRepository(
		postgres.NewCategoryRepository(pool), c,
	)
}

// ProvideProductRepository creates a new ProductRepository backed by PostgreSQL with cache
func ProvideProductRepository(pool *pgxpool.Pool, c *cache.Cache) domain.ProductRepository {
	return postgres.NewCachedProductRepository(
		postgres.NewProductRepository(pool), c,
	)
}

// ProvideInventoryRepository creates a new InventoryRepository backed by PostgreSQL
func ProvideInventoryRepository(pool *pgxpool.Pool) domain.InventoryRepository {
	return postgres.NewInventoryRepository(pool)
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

// ProvideReportRepository creates a new ReportRepository
func ProvideReportRepository(client *dynamodb.Client) domain.ReportRepository {
	return dynamodb.NewReportRepository(client)
}

// ProvideAuditRepository creates a new AuditRepository
func ProvideAuditRepository(client *dynamodb.Client) domain.AuditRepository {
	return dynamodb.NewAuditRepository(client)
}

// ProvideEventsRepository creates a new EventsRepository
func ProvideEventsRepository(client *dynamodb.Client) domain.EventsRepository {
	return dynamodb.NewEventsRepository(client)
}

// ProvideShippingRateRepository creates a new ShippingRateRepository
func ProvideShippingRateRepository(client *dynamodb.Client) domain.ShippingRateRepository {
	return dynamodb.NewShippingRateRepository(client)
}

// ProvidePincodeRepository creates a new PincodeRepository
func ProvidePincodeRepository(client *dynamodb.Client) domain.PincodeRepository {
	return dynamodb.NewPincodeRepository(client)
}

// ProvideCODRemittanceRepository creates a new CODRemittanceRepository
func ProvideCODRemittanceRepository(client *dynamodb.Client) domain.CODRemittanceRepository {
	return dynamodb.NewCODRemittanceRepository(client)
}

// ProvideReturnRepository creates a new ReturnRepository
func ProvideReturnRepository(client *dynamodb.Client) domain.ReturnRepository {
	return dynamodb.NewReturnRepository(client)
}

// ProvideManifestRepository creates a new ManifestRepository
func ProvideManifestRepository(client *dynamodb.Client) domain.ManifestRepository {
	return dynamodb.NewManifestRepository(client)
}

// RepositorySet contains all repository providers
var RepositorySet = wire.NewSet(
	ProvideUserRepository,
	ProvideTokenStore,
	ProvideCategoryRepository,
	ProvideProductRepository,
	ProvideInventoryRepository,
	ProvideOrderRepository,
	ProvidePaymentRepository,
	ProvideCartRepository,
	ProvideCustomerRepository,
	ProvidePricingRuleRepository,
	ProvidePriceQuoteRepository,
	ProvideAnalyticsRepository,
	ProvideNotificationRepository,
	ProvideCouponRepository,
	ProvideReportRepository,
	ProvideAuditRepository,
	ProvideEventsRepository,
	ProvideShippingRateRepository,
	ProvidePincodeRepository,
	ProvideCODRemittanceRepository,
	ProvideReturnRepository,
	ProvideManifestRepository,
)

// ============================================================================
// SERVICE PROVIDERS
// ============================================================================

// ProvideAuthService creates a new AuthService
func ProvideAuthService(
	userRepo domain.UserRepository,
	tokenStore domain.TokenStore,
	cfg *config.Config,
) *service.AuthService {
	return service.NewAuthService(
		userRepo,
		tokenStore,
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
) *service.UserService {
	return service.NewUserService(userRepo, tokenStore)
}

// ProvideCategoryService creates a new CategoryService
func ProvideCategoryService(
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	assetService *service.AssetService,
) *service.CategoryService {
	return service.NewCategoryService(categoryRepo, productRepo, assetService)
}

// ProvideProductService creates a new ProductService
func ProvideProductService(
	productRepo domain.ProductRepository,
	categoryRepo domain.CategoryRepository,
	inventoryRepo domain.InventoryRepository,
	assetService *service.AssetService,
	publisher event.EventPublisher,
) *service.ProductService {
	return service.NewProductService(productRepo, categoryRepo, inventoryRepo, assetService, publisher)
}

// ProvideInventoryService creates a new InventoryService
func ProvideInventoryService(
	inventoryRepo domain.InventoryRepository,
	c *cache.Cache,
	publisher event.EventPublisher,
) *service.InventoryService {
	return service.NewInventoryService(inventoryRepo, c, publisher)
}

// ProvidePricingService creates a new PricingService
func ProvidePricingService(
	pricingRuleRepo domain.PricingRuleRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	cfg *config.Config,
) *service.PricingService {
	return service.NewPricingService(
		pricingRuleRepo,
		priceQuoteRepo,
		categoryRepo,
		productRepo,
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
) *service.OrderService {
	return service.NewOrderService(orderRepo, customerRepo, productRepo, inventoryRepo, priceQuoteRepo, pricingService)
}

// ProvideCustomerService creates a new CustomerService
func ProvideCustomerService(
	customerRepo domain.CustomerRepository,
	orderRepo domain.OrderRepository,
) *service.CustomerService {
	return service.NewCustomerService(customerRepo, orderRepo)
}

// ProvideAnalyticsService creates a new AnalyticsService
func ProvideAnalyticsService(
	analyticsRepo domain.AnalyticsRepository,
	orderRepo domain.OrderRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
) *service.AnalyticsService {
	return service.NewAnalyticsService(analyticsRepo, orderRepo, productRepo, inventoryRepo)
}

// ProvideNotificationService creates a new NotificationService
func ProvideNotificationService(
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
) *service.NotificationService {
	return service.NewNotificationService(notificationRepo, userRepo)
}

// ProvideCouponService creates a new CouponService
func ProvideCouponService(
	couponRepo domain.CouponRepository,
) *service.CouponService {
	return service.NewCouponService(couponRepo)
}

// ProvideAssetService creates a new AssetService
func ProvideAssetService(
	s3Client *s3client.S3Client,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(s3Client, cfg.AWS.S3Bucket, cfg.AWS.Region, cfg.AWS.CDNUrl, cfg.AWS.Endpoint)
}

// ProvideReportService creates a new ReportService
func ProvideReportService(
	reportRepo domain.ReportRepository,
	orderService *service.OrderService,
	productService *service.ProductService,
	customerService *service.CustomerService,
	inventoryService *service.InventoryService,
	analyticsService *service.AnalyticsService,
) *service.ReportService {
	return service.NewReportService(reportRepo, orderService, productService, customerService, inventoryService, analyticsService)
}

// ProvideAuditService creates a new AuditService
func ProvideAuditService(
	auditRepo domain.AuditRepository,
) *service.AuditService {
	return service.NewAuditService(auditRepo)
}

// ProvideEventPublisher creates an EventPublisher for Lambda mode.
// Returns SNSPublisher when events are enabled, NoopPublisher otherwise.
// Monolith mode uses ProvideLocalEventPublisher instead.
func ProvideEventPublisher(ctx context.Context, cfg *config.Config) (event.EventPublisher, error) {
	if cfg.Event.Enabled {
		return event.NewSNSPublisher(ctx, cfg.Event.SNSTopicARN, cfg.AWS.Region, cfg.AWS.Endpoint)
	}
	return event.NewNoopPublisher(), nil
}

// LambdaPublisherSet provides the Lambda-mode event publisher (SNS or Noop).
var LambdaPublisherSet = wire.NewSet(
	ProvideEventPublisher,
)

// ServiceSet contains all service providers (publisher excluded — choose
// LambdaPublisherSet or MonolithPublisherSet per injector).
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
	ProvideAssetService,
	ProvideReportService,
	ProvideAuditService,
	ProvideCartService,
	ProvidePhonePeGateway,
	ProvideDelhiveryGateway,
	ProvidePaymentService,
	ProvideManifestService,
	ProvideNDRService,
	ProvideCODReconciliationService,
	ProvideRateTableService,
	ProvideReturnService,
)

// ============================================================================
// HANDLER PROVIDERS
// ============================================================================

// ProvideAuthHandler creates a new AuthHandler
func ProvideAuthHandler(
	authService *service.AuthService,
	userService *service.UserService,
	validation *middleware.Validation,
) *handler.AuthHandler {
	return handler.NewAuthHandler(authService, userService, validation)
}

// ProvideUserHandler creates a new UserHandler
func ProvideUserHandler(
	userService *service.UserService,
	validation *middleware.Validation,
) *handler.UserHandler {
	return handler.NewUserHandler(userService, validation)
}

// ProvideCategoryHandler creates a new CategoryHandler
func ProvideCategoryHandler(
	categoryService *service.CategoryService,
	validation *middleware.Validation,
) *handler.CategoryHandler {
	return handler.NewCategoryHandler(categoryService, validation)
}

// ProvideProductHandler creates a new ProductHandler
func ProvideProductHandler(
	productService *service.ProductService,
	inventoryService *service.InventoryService,
	validation *middleware.Validation,
) *handler.ProductHandler {
	return handler.NewProductHandler(productService, inventoryService, validation)
}

// ProvideInventoryHandler creates a new InventoryHandler
func ProvideInventoryHandler(
	inventoryService *service.InventoryService,
) *handler.InventoryHandler {
	return handler.NewInventoryHandler(inventoryService)
}

// ProvideOrderHandler creates a new OrderHandler
func ProvideOrderHandler(
	orderService *service.OrderService,
	paymentService *service.PaymentService,
	returnService *service.ReturnService,
	validation *middleware.Validation,
) *handler.OrderHandler {
	return handler.NewOrderHandler(orderService, paymentService, returnService, validation)
}

// ProvideCustomerHandler creates a new CustomerHandler
func ProvideCustomerHandler(
	customerService *service.CustomerService,
	validation *middleware.Validation,
) *handler.CustomerHandler {
	return handler.NewCustomerHandler(customerService, validation)
}

// ProvidePricingHandler creates a new PricingHandler
func ProvidePricingHandler(
	pricingService *service.PricingService,
	validation *middleware.Validation,
) *handler.PricingHandler {
	return handler.NewPricingHandler(pricingService, validation)
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

// ProvideAWSSDKConfig loads a default AWS SDK config scoped to the
// configured region. Used by callers (e.g. LambdaInvoker) that need raw
// SDK clients alongside the existing typed DynamoDB/S3 wrappers.
func ProvideAWSSDKConfig(ctx context.Context, cfg *config.Config) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWS.Region),
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// ProvideRateRefreshInvoker builds a Lambda invoker bound to the
// cron-rate-refresh function name (from RATE_REFRESH_LAMBDA_NAME).
func ProvideRateRefreshInvoker(awsCfg aws.Config, cfg *config.Config) *lambdainvoker.LambdaInvoker {
	client := awslambda.NewFromConfig(awsCfg)
	return lambdainvoker.NewLambdaInvoker(client, cfg.Delhivery.RateRefreshLambdaName)
}

// ProvideRateRefreshFn returns nil when no rate-refresh Lambda is
// configured so the handler falls back to synchronous refresh.
// Otherwise it returns the invoker's Invoke method.
func ProvideRateRefreshFn(invoker *lambdainvoker.LambdaInvoker, cfg *config.Config) func(ctx context.Context) error {
	if cfg.Delhivery.RateRefreshLambdaName == "" {
		return nil
	}
	return invoker.Invoke
}

// ProvideShippingAdminHandler creates a new ShippingAdminHandler.
// rateRefreshFn is nil when RATE_REFRESH_LAMBDA_NAME is unset — the handler
// falls back to running RateTableService.Refresh inline.
func ProvideShippingAdminHandler(
	rateRepo domain.ShippingRateRepository,
	rateTable *service.RateTableService,
	remRepo domain.CODRemittanceRepository,
	ndr *service.NDRService,
	shipmentRepo domain.ShipmentRepository,
	manifest *service.ManifestService,
	validation *middleware.Validation,
	rateRefreshFn func(ctx context.Context) error,
) *handler.ShippingAdminHandler {
	return handler.NewShippingAdminHandler(
		rateRepo, rateTable, remRepo, ndr, shipmentRepo, manifest, validation, rateRefreshFn,
	)
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
	ProvideAssetHandler,
	ProvideReportHandler,
	ProvideAuditHandler,
	ProvideShippingAdminHandler,
)

// ============================================================================
// MIDDLEWARE PROVIDERS
// ============================================================================

// ProvideAuthMiddleware creates the Auth middleware
func ProvideAuthMiddleware(
	authService *service.AuthService,
) *middleware.Auth {
	return middleware.NewAuth(authService)
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

// ============================================================================
// B2C STORE REPOSITORY PROVIDERS
// ============================================================================

// ProvideOTPRepository creates a new OTPRepository
func ProvideOTPRepository(client *dynamodb.Client) domain.OTPRepository {
	return dynamodb.NewOTPRepository(client)
}

// ProvideCustomerTokenStore creates a new CustomerTokenStore
func ProvideCustomerTokenStore(client *dynamodb.Client) domain.CustomerTokenStore {
	return dynamodb.NewCustomerTokenStore(client)
}

// ProvideCartRepository creates a new CartRepository
func ProvideCartRepository(client *dynamodb.Client) domain.CartRepository {
	return dynamodb.NewCartRepository(client)
}

// ProvidePaymentRepository creates a new PaymentRepository
func ProvidePaymentRepository(client *dynamodb.Client) domain.PaymentRepository {
	return dynamodb.NewPaymentRepository(client)
}

// ProvideShipmentRepository creates a new ShipmentRepository
func ProvideShipmentRepository(client *dynamodb.Client) domain.ShipmentRepository {
	return dynamodb.NewShipmentRepository(client)
}

// ============================================================================
// B2C STORE SERVICE PROVIDERS
// ============================================================================

// ProvideCustomerAuthService creates a new CustomerAuthService with SMS gateway
func ProvideCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	publisher event.EventPublisher,
	cfg *config.Config,
) *service.CustomerAuthService {
	var smsGateway domain.SMSGateway
	if cfg.Store.MSG91AuthKey == "" || cfg.Store.MSG91OTPTemplateID == "" {
		smsGateway = sms.NewDevClient()
	} else {
		smsGateway = sms.NewClient(sms.Config{
			AuthKey:       cfg.Store.MSG91AuthKey,
			OTPTemplateID: cfg.Store.MSG91OTPTemplateID,
			BaseURL:       cfg.Store.MSG91BaseURL,
		})
	}
	return service.NewCustomerAuthService(
		otpRepo, customerRepo, tokenStore, smsGateway, publisher,
		service.CustomerAuthConfig{
			JWTSecret:            cfg.Store.CustomerJWTSecret,
			AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
			RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
			Issuer:               "handloom-store",
		},
	)
}

// ProvideCartService creates a new CartService
func ProvideCartService(
	cartRepo domain.CartRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
) *service.CartService {
	return service.NewCartService(cartRepo, productRepo, inventoryRepo)
}

// ProvidePhonePeGateway creates the PhonePe gateway (real or dev client)
func ProvidePhonePeGateway(cfg *config.Config) phonepe.Gateway {
	if cfg.Store.PhonePeClientID == "" || cfg.Store.PhonePeClientSecret == "" {
		return phonepe.NewDevClient(cfg.Store.PhonePeRedirectURL)
	}
	return phonepe.NewClient(phonepe.Config{
		ClientID:      cfg.Store.PhonePeClientID,
		ClientSecret:  cfg.Store.PhonePeClientSecret,
		ClientVersion: cfg.Store.PhonePeClientVersion,
		BaseURL:       cfg.Store.PhonePeBaseURL,
		CallbackURL:   cfg.Store.PhonePeCallbackURL,
		RedirectURL:   cfg.Store.PhonePeRedirectURL,
	})
}

// ProvideDelhiveryGateway returns a courier.Gateway, choosing DevClient when API token is empty.
func ProvideDelhiveryGateway(cfg *config.Config) courier.Gateway {
	if cfg.Delhivery.APIToken == "" {
		return delhivery.NewDevClient()
	}
	return delhivery.NewClient(delhivery.Config{
		APIToken:       cfg.Delhivery.APIToken,
		BaseURL:        cfg.Delhivery.BaseURL,
		ClientName:     cfg.Delhivery.ClientName,
		WebhookToken:   cfg.Delhivery.WebhookToken,
		PickupLocation: cfg.Delhivery.PickupLocation,
	})
}

// ProvidePaymentService creates a new PaymentService
func ProvidePaymentService(
	paymentRepo domain.PaymentRepository,
	orderRepo domain.OrderRepository,
	inventoryRepo domain.InventoryRepository,
	cartService *service.CartService,
	phonePeClient phonepe.Gateway,
	publisher event.EventPublisher,
) *service.PaymentService {
	return service.NewPaymentService(paymentRepo, orderRepo, inventoryRepo, cartService, phonePeClient, publisher)
}

// ProvideShippingService creates a new ShippingService backed by a courier.Gateway.
func ProvideShippingService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	pincodeRepo domain.PincodeRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
	returnService *service.ReturnService,
	cfg *config.Config,
) *service.ShippingService {
	return service.NewShippingService(
		shipmentRepo, orderRepo, pincodeRepo, gw, pub, returnService, cfg.Delhivery.PickupLocation,
	)
}

// ProvideManifestService creates a new ManifestService.
func ProvideManifestService(
	shipmentRepo domain.ShipmentRepository,
	manifestRepo domain.ManifestRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
	cfg *config.Config,
) *service.ManifestService {
	return service.NewManifestService(shipmentRepo, manifestRepo, gw, pub, cfg.Delhivery.PickupLocation)
}

// ProvideNDRService creates a new NDRService.
func ProvideNDRService(
	shipmentRepo domain.ShipmentRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
	cfg *config.Config,
) *service.NDRService {
	maxAttempts := cfg.Delhivery.NDRLimit
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return service.NewNDRService(shipmentRepo, gw, pub, maxAttempts)
}

// ProvideCODReconciliationService creates a new CODReconciliationService.
func ProvideCODReconciliationService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	remRepo domain.CODRemittanceRepository,
	gw courier.Gateway,
	pub event.EventPublisher,
) *service.CODReconciliationService {
	return service.NewCODReconciliationService(shipmentRepo, orderRepo, remRepo, gw, pub)
}

// ProvideRateTableService creates a new RateTableService.
func ProvideRateTableService(
	rateRepo domain.ShippingRateRepository,
	pincodeRepo domain.PincodeRepository,
	gw courier.Gateway,
) *service.RateTableService {
	return service.NewRateTableService(rateRepo, pincodeRepo, gw)
}

// ProvideReturnService creates a new ReturnService.
func ProvideReturnService(
	orderRepo domain.OrderRepository,
	shipmentRepo domain.ShipmentRepository,
	returnRepo domain.ReturnRepository,
	paymentSvc *service.PaymentService,
	gw courier.Gateway,
	pub event.EventPublisher,
	cfg *config.Config,
) *service.ReturnService {
	window := cfg.Delhivery.ReturnWindowDays
	if window <= 0 {
		window = 7
	}
	return service.NewReturnService(
		orderRepo, shipmentRepo, returnRepo, paymentSvc, gw, pub, cfg.Delhivery.PickupLocation, window,
	)
}

// ProvideCheckoutService creates a new CheckoutService
func ProvideCheckoutService(
	cartService *service.CartService,
	orderRepo domain.OrderRepository,
	paymentService *service.PaymentService,
	shippingService *service.ShippingService,
	rateTable *service.RateTableService,
	inventoryRepo domain.InventoryRepository,
	customerRepo domain.CustomerRepository,
	publisher event.EventPublisher,
) *service.CheckoutService {
	return service.NewCheckoutService(cartService, orderRepo, paymentService, shippingService, rateTable, inventoryRepo, customerRepo, publisher)
}

// ============================================================================
// B2C STORE HANDLER PROVIDERS
// ============================================================================

// ProvideStoreAuthHandler creates a new store AuthHandler
func ProvideStoreAuthHandler(
	customerAuthService *service.CustomerAuthService,
	cartService *service.CartService,
	validation *middleware.Validation,
) *store.AuthHandler {
	return store.NewAuthHandler(customerAuthService, cartService, validation)
}

// ProvideStoreCatalogHandler creates a new store CatalogHandler
func ProvideStoreCatalogHandler(
	productService *service.ProductService,
	categoryService *service.CategoryService,
	inventoryService *service.InventoryService,
	shippingService *service.ShippingService,
) *store.CatalogHandler {
	return store.NewCatalogHandler(productService, categoryService, inventoryService, shippingService)
}

// ProvideStoreCartHandler creates a new store CartHandler
func ProvideStoreCartHandler(
	cartService *service.CartService,
	validation *middleware.Validation,
) *store.CartHandler {
	return store.NewCartHandler(cartService, validation)
}

// ProvideStoreCheckoutHandler creates a new store CheckoutHandler
func ProvideStoreCheckoutHandler(
	checkoutService *service.CheckoutService,
	validation *middleware.Validation,
) *store.CheckoutHandler {
	return store.NewCheckoutHandler(checkoutService, validation)
}

// ProvideStoreOrderHandler creates a new store OrderHandler
func ProvideStoreOrderHandler(
	orderService *service.OrderService,
	orderRepo domain.OrderRepository,
) *store.OrderHandler {
	return store.NewOrderHandler(orderService, orderRepo)
}

// ProvideStoreTrackingHandler creates a new store TrackingHandler
func ProvideStoreTrackingHandler(
	orderRepo domain.OrderRepository,
	shipmentRepo domain.ShipmentRepository,
) *store.TrackingHandler {
	return store.NewTrackingHandler(orderRepo, shipmentRepo)
}

// ProvideStoreProfileHandler creates a new store ProfileHandler
func ProvideStoreProfileHandler(
	customerRepo domain.CustomerRepository,
	validation *middleware.Validation,
) *store.ProfileHandler {
	return store.NewProfileHandler(customerRepo, validation)
}

// ProvideStoreWebhookHandler creates a new store WebhookHandler
func ProvideStoreWebhookHandler(
	paymentService *service.PaymentService,
	shippingService *service.ShippingService,
	phonePe phonepe.Gateway,
	cfg *config.Config,
) *store.WebhookHandler {
	return store.NewWebhookHandler(paymentService, shippingService, phonePe, cfg.Store.PhonePeWebhookUsername, cfg.Store.PhonePeWebhookPassword)
}

// ProvideStoreEventsHandler creates a new store EventsHandler
func ProvideStoreEventsHandler(
	eventsRepo domain.EventsRepository,
	analyticsRepo domain.AnalyticsRepository,
	validation *middleware.Validation,
) *store.EventsHandler {
	return store.NewEventsHandler(eventsRepo, analyticsRepo, validation)
}

// ============================================================================
// B2C STORE MIDDLEWARE PROVIDERS
// ============================================================================

// ProvideCustomerAuthMiddleware creates the CustomerAuth middleware
func ProvideCustomerAuthMiddleware(
	customerAuthService *service.CustomerAuthService,
) *middleware.CustomerAuth {
	return middleware.NewCustomerAuth(customerAuthService)
}

// ProvideOptionalCartAuth creates the OptionalCartAuth middleware
func ProvideOptionalCartAuth(
	customerAuthService *service.CustomerAuthService,
) *middleware.OptionalCartAuth {
	return middleware.NewOptionalCartAuth(customerAuthService)
}

// Store-only providers that fill gaps in the admin RepositorySet/ServiceSet/
// HandlerSet/MiddlewareSet so the monolith can wire admin + store in one graph.

var StoreRepositorySet = wire.NewSet(
	ProvideShipmentRepository,
	ProvideOTPRepository,
	ProvideCustomerTokenStore,
)

var StoreServiceSet = wire.NewSet(
	ProvideCustomerAuthService,
	ProvideShippingService,
	ProvideCheckoutService,
)

var StoreHandlerSet = wire.NewSet(
	ProvideStoreAuthHandler,
	ProvideStoreCatalogHandler,
	ProvideStoreCartHandler,
	ProvideStoreCheckoutHandler,
	ProvideStoreOrderHandler,
	ProvideStoreTrackingHandler,
	ProvideStoreProfileHandler,
	ProvideStoreWebhookHandler,
	ProvideStoreEventsHandler,
)

var StoreMiddlewareSet = wire.NewSet(
	ProvideCustomerAuthMiddleware,
	ProvideOptionalCartAuth,
)

func ProvideNotificationEventHandler() *eventhandlers.NotificationHandler {
	return eventhandlers.NewNotificationHandler()
}

func ProvideReportEventHandler() *eventhandlers.ReportHandler {
	return eventhandlers.NewReportHandler()
}

func ProvideAuditEventHandler() *eventhandlers.AuditHandler {
	return eventhandlers.NewAuditHandler()
}

func ProvideAnalyticsAggregator(
	eventsRepo domain.EventsRepository,
	analyticsRepo domain.AnalyticsRepository,
) *service.AnalyticsAggregator {
	return service.NewAnalyticsAggregator(eventsRepo, analyticsRepo)
}

func ProvideAnalyticsEventHandler(
	eventsRepo domain.EventsRepository,
	analyticsRepo domain.AnalyticsRepository,
	aggregator *service.AnalyticsAggregator,
) *eventhandlers.AnalyticsHandler {
	return eventhandlers.NewAnalyticsHandler(eventsRepo, analyticsRepo, aggregator)
}

// ProvideLocalEventPublisher dispatches events in-process (monolith dev mode).
// Lambda mode uses ProvideEventPublisher instead.
func ProvideLocalEventPublisher(
	notif *eventhandlers.NotificationHandler,
	report *eventhandlers.ReportHandler,
	analytics *eventhandlers.AnalyticsHandler,
	audit *eventhandlers.AuditHandler,
) event.EventPublisher {
	return event.NewLocalPublisher(notif, report, analytics, audit)
}

// ============================================================================
// CRON HANDLER PROVIDERS
// ============================================================================

// ProvidePickupBatchHandler creates a new cron PickupBatchHandler.
func ProvidePickupBatchHandler(m *service.ManifestService) *cron.PickupBatchHandler {
	return cron.NewPickupBatchHandler(m)
}

// ProvideCODRemittanceHandler creates a new cron CODRemittanceHandler.
func ProvideCODRemittanceHandler(c *service.CODReconciliationService) *cron.CODRemittanceHandler {
	return cron.NewCODRemittanceHandler(c)
}

// ProvideRateRefreshHandler creates a new cron RateRefreshHandler.
func ProvideRateRefreshHandler(r *service.RateTableService) *cron.RateRefreshHandler {
	return cron.NewRateRefreshHandler(r)
}

var MonolithPublisherSet = wire.NewSet(
	ProvideNotificationEventHandler,
	ProvideReportEventHandler,
	ProvideAuditEventHandler,
	ProvideAnalyticsAggregator,
	ProvideAnalyticsEventHandler,
	ProvideLocalEventPublisher,
)
