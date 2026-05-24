// Package wire provides dependency injection using Google Wire
package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	eventhandlers "github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/gateway/shiprocket"
	"github.com/handloom/admin/internal/gateway/sms"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/lambdaclient"
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

// ProvideLambdaClient creates a new Lambda client used for sync invocations.
func ProvideLambdaClient(ctx context.Context, cfg *config.Config) (*lambdaclient.LambdaClient, error) {
	return lambdaclient.New(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
}

// ProvidePostgresPool creates a new PostgreSQL connection pool for catalog data
func ProvidePostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	return postgres.NewPool(ctx, &cfg.Postgres)
}

// CoreSet contains core providers used by all services
var CoreSet = wire.NewSet(
	ProvideDynamoDBClient,
	ProvidePostgresPool,
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

// ProvideCategoryRepository creates a new CategoryRepository backed by PostgreSQL
func ProvideCategoryRepository(pool *pgxpool.Pool) domain.CategoryRepository {
	return postgres.NewCategoryRepository(pool)
}

// ProvideProductRepository creates a new ProductRepository backed by PostgreSQL
func ProvideProductRepository(pool *pgxpool.Pool) domain.ProductRepository {
	return postgres.NewProductRepository(pool)
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
	publisher event.EventPublisher,
) *service.InventoryService {
	return service.NewInventoryService(inventoryRepo, publisher)
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
	lambdaClient *lambdaclient.LambdaClient,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(
		s3Client,
		lambdaClient,
		cfg.AWS.S3Bucket,
		cfg.AWS.Region,
		cfg.AWS.CDNUrl,
		cfg.AWS.Endpoint,
		cfg.AWS.ImageResizerFn,
	)
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
	ProvidePaymentService,
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
	validation *middleware.Validation,
) *handler.OrderHandler {
	return handler.NewOrderHandler(orderService, paymentService, validation)
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

// ProvideShippingService creates a new ShippingService with Shiprocket gateway
func ProvideShippingService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	cfg *config.Config,
) *service.ShippingService {
	var shiprocketClient shiprocket.Gateway
	if cfg.Store.ShiprocketEmail == "" || cfg.Store.ShiprocketPassword == "" {
		shiprocketClient = shiprocket.NewDevClient()
	} else {
		shiprocketClient = shiprocket.NewClient(shiprocket.Config{
			Email:         cfg.Store.ShiprocketEmail,
			Password:      cfg.Store.ShiprocketPassword,
			BaseURL:       cfg.Store.ShiprocketBaseURL,
			PickupPincode: cfg.Store.ShiprocketPickupPincode,
		})
	}
	return service.NewShippingService(shipmentRepo, orderRepo, shiprocketClient, cfg.Store.ShiprocketPickupPincode)
}

// ProvideCheckoutService creates a new CheckoutService
func ProvideCheckoutService(
	cartService *service.CartService,
	orderRepo domain.OrderRepository,
	paymentService *service.PaymentService,
	shippingService *service.ShippingService,
	inventoryRepo domain.InventoryRepository,
	customerRepo domain.CustomerRepository,
	publisher event.EventPublisher,
) *service.CheckoutService {
	return service.NewCheckoutService(cartService, orderRepo, paymentService, shippingService, inventoryRepo, customerRepo, publisher)
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
) *store.CatalogHandler {
	return store.NewCatalogHandler(productService, categoryService, inventoryService)
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
	phonePe phonepe.Gateway,
	cfg *config.Config,
) *store.WebhookHandler {
	return store.NewWebhookHandler(paymentService, phonePe, cfg.Store.PhonePeWebhookUsername, cfg.Store.PhonePeWebhookPassword)
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

var MonolithPublisherSet = wire.NewSet(
	ProvideNotificationEventHandler,
	ProvideReportEventHandler,
	ProvideAuditEventHandler,
	ProvideAnalyticsAggregator,
	ProvideAnalyticsEventHandler,
	ProvideLocalEventPublisher,
)
