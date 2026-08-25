//go:build wireinject
// +build wireinject

// Package wire provides dependency injection using Google Wire
package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/embedder"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
	metricsworker "github.com/handloom/admin/internal/worker/metrics"
)

// ============================================================================
// ADMIN SERVICE DEPENDENCY CONTAINERS
// ============================================================================

// AuthDeps holds dependencies for the Auth Lambda
type AuthDeps struct {
	Config         *config.Config
	Handler        *handler.AuthHandler
	AuthMiddleware *middleware.Auth
}

// UserDeps holds dependencies for the User Lambda
type UserDeps struct {
	Config         *config.Config
	Handler        *handler.UserHandler
	AuthMiddleware *middleware.Auth
}

// CatalogDeps holds dependencies for the Catalog Lambda
type CatalogDeps struct {
	Config          *config.Config
	CategoryHandler *handler.CategoryHandler
	ProductHandler  *handler.ProductHandler
	AuthMiddleware  *middleware.Auth
	EmbedderClient  *embedder.Client
}

// OrderDeps holds dependencies for the Order Lambda
type OrderDeps struct {
	Config          *config.Config
	OrderHandler    *handler.OrderHandler
	CustomerHandler *handler.CustomerHandler
	AuthMiddleware  *middleware.Auth
}

// PricingDeps holds dependencies for the Pricing Lambda
type PricingDeps struct {
	Config         *config.Config
	Handler        *handler.PricingHandler
	AuthMiddleware *middleware.Auth
}

// InventoryDeps holds dependencies for the Inventory Lambda
type InventoryDeps struct {
	Config         *config.Config
	Handler        *handler.InventoryHandler
	AuthMiddleware *middleware.Auth
}

// NotificationDeps holds dependencies for the Notification Lambda
type NotificationDeps struct {
	Config         *config.Config
	Handler        *handler.NotificationHandler
	AuthMiddleware *middleware.Auth
}

// CouponDeps holds dependencies for the Coupon Lambda
type CouponDeps struct {
	Config         *config.Config
	Handler        *handler.CouponHandler
	AuthMiddleware *middleware.Auth
}

// UTMDeps holds dependencies for the UTM link Lambda
type UTMDeps struct {
	Config         *config.Config
	Handler        *handler.UTMLinkHandler
	AuthMiddleware *middleware.Auth
}

// AssetDeps holds dependencies for the Asset Lambda
type AssetDeps struct {
	Config         *config.Config
	Handler        *handler.AssetHandler
	AuthMiddleware *middleware.Auth
}

// ReportDeps holds dependencies for the Report Lambda
type ReportDeps struct {
	Config         *config.Config
	Handler        *handler.ReportHandler
	AuthMiddleware *middleware.Auth
}

// AuditDeps holds dependencies for the Audit Lambda
type AuditDeps struct {
	Config         *config.Config
	Handler        *handler.AuditHandler
	AuthMiddleware *middleware.Auth
}

// ============================================================================
// ADMIN INJECTOR FUNCTIONS
// ============================================================================

// InitializeAuthDeps creates Auth Lambda dependencies
func InitializeAuthDeps(ctx context.Context, cfg *config.Config) (*AuthDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuthService,
		ProvideUserService,
		ProvideAuthHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(AuthDeps), "*"),
	)
	return nil, nil
}

// InitializeUserDeps creates User Lambda dependencies
func InitializeUserDeps(ctx context.Context, cfg *config.Config) (*UserDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuthService,
		ProvideUserService,
		ProvideUserHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(UserDeps), "*"),
	)
	return nil, nil
}

// InitializeCatalogDeps creates Catalog Lambda dependencies
func InitializeCatalogDeps(ctx context.Context, cfg *config.Config) (*CatalogDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideS3Client,
		ProvideLambdaClient,
		ProvideEmbedderClient,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideCategoryRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideAuthService,
		ProvideAssetService,
		ProvideCategoryService,
		ProvideProductService,
		ProvideInventoryService,
		ProvideCategoryHandler,
		ProvideProductHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(CatalogDeps), "*"),
	)
	return nil, nil
}

// InitializeOrderDeps creates Order Lambda dependencies
func InitializeOrderDeps(ctx context.Context, cfg *config.Config) (*OrderDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvidePaymentRepository,
		ProvideCartRepository,
		ProvidePriceQuoteRepository,
		ProvidePricingRuleRepository,
		ProvideCategoryRepository,
		ProvideAuthService,
		ProvidePricingService,
		ProvideOrderService,
		ProvideCustomerService,
		ProvideCartService,
		ProvidePhonePeGateway,
		ProvidePaymentService,
		ProvideRefundRepository,
		ProvideAuditRepository,
		ProvideAuditService,
		ProvideNotificationRepository,
		ProvideNotificationService,
		ProvideRefundService,
		ProvideOrderHandler,
		ProvideCustomerHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(OrderDeps), "*"),
	)
	return nil, nil
}

// InitializePricingDeps creates Pricing Lambda dependencies
func InitializePricingDeps(ctx context.Context, cfg *config.Config) (*PricingDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvidePricingRuleRepository,
		ProvidePriceQuoteRepository,
		ProvideCategoryRepository,
		ProvideProductRepository,
		ProvideAuthService,
		ProvidePricingService,
		ProvidePricingHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(PricingDeps), "*"),
	)
	return nil, nil
}

// InitializeInventoryDeps creates Inventory Lambda dependencies
func InitializeInventoryDeps(ctx context.Context, cfg *config.Config) (*InventoryDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideInventoryRepository,
		ProvideAuthService,
		ProvideInventoryService,
		ProvideInventoryHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(InventoryDeps), "*"),
	)
	return nil, nil
}

// InitializeNotificationDeps creates Notification Lambda dependencies
func InitializeNotificationDeps(ctx context.Context, cfg *config.Config) (*NotificationDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideNotificationRepository,
		ProvideAuthService,
		ProvideNotificationService,
		ProvideNotificationHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(NotificationDeps), "*"),
	)
	return nil, nil
}

// InitializeCouponDeps creates Coupon Lambda dependencies
func InitializeCouponDeps(ctx context.Context, cfg *config.Config) (*CouponDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideCouponRepository,
		ProvideAuthService,
		ProvideCouponService,
		ProvideCouponHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(CouponDeps), "*"),
	)
	return nil, nil
}

// InitializeUTMDeps creates UTM link Lambda dependencies
func InitializeUTMDeps(ctx context.Context, cfg *config.Config) (*UTMDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideUTMLinkRepository,
		ProvideAuthService,
		ProvideUTMLinkService,
		ProvideUTMLinkHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(UTMDeps), "*"),
	)
	return nil, nil
}


// InitializeAssetDeps creates Asset Lambda dependencies
func InitializeAssetDeps(ctx context.Context, cfg *config.Config) (*AssetDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideS3Client,
		ProvideLambdaClient,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuthService,
		ProvideAssetService,
		ProvideAssetHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(AssetDeps), "*"),
	)
	return nil, nil
}

// InitializeReportDeps creates Report Lambda dependencies
func InitializeReportDeps(ctx context.Context, cfg *config.Config) (*ReportDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideS3Client,
		ProvideLambdaClient,
		ProvideEmbedderClient,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideReportRepository,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvidePriceQuoteRepository,
		ProvidePricingRuleRepository,
		ProvideCategoryRepository,
		ProvideAuthService,
		ProvideAssetService,
		ProvidePricingService,
		ProvidePaymentRepository,
		ProvideOrderService,
		ProvideProductService,
		ProvideCustomerService,
		ProvideInventoryService,
		ProvideReportService,
		ProvideReportHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(ReportDeps), "*"),
	)
	return nil, nil
}

// InitializeAuditDeps creates Audit Lambda dependencies
func InitializeAuditDeps(ctx context.Context, cfg *config.Config) (*AuditDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuditRepository,
		ProvideAuthService,
		ProvideAuditService,
		ProvideAuditHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(AuditDeps), "*"),
	)
	return nil, nil
}

// ============================================================================
// B2C STORE SERVICE DEPENDENCY CONTAINERS
// ============================================================================

// StoreAuthDeps holds dependencies for the Store Auth Lambda
type StoreAuthDeps struct {
	Config                 *config.Config
	Handler                *store.AuthHandler
	CustomerAuthMiddleware *middleware.CustomerAuth
}

// StoreCatalogDeps holds dependencies for the Store Catalog Lambda
type StoreCatalogDeps struct {
	Config  *config.Config
	Handler *store.CatalogHandler
}

// StoreCartDeps holds dependencies for the Store Cart Lambda
type StoreCartDeps struct {
	Config           *config.Config
	Handler          *store.CartHandler
	OptionalCartAuth *middleware.OptionalCartAuth
}

// StoreCheckoutDeps holds dependencies for the Store Checkout Lambda
type StoreCheckoutDeps struct {
	Config                 *config.Config
	Handler                *store.CheckoutHandler
	CustomerAuthMiddleware *middleware.CustomerAuth
}

// StoreOrdersDeps holds dependencies for the Store Orders Lambda
type StoreOrdersDeps struct {
	Config                 *config.Config
	Handler                *store.OrderHandler
	CustomerAuthMiddleware *middleware.CustomerAuth
}

// StoreTrackingDeps holds dependencies for the Store Tracking Lambda
type StoreTrackingDeps struct {
	Config  *config.Config
	Handler *store.TrackingHandler
}

// StoreProfileDeps holds dependencies for the Store Profile Lambda
type StoreProfileDeps struct {
	Config                 *config.Config
	Handler                *store.ProfileHandler
	CustomerAuthMiddleware *middleware.CustomerAuth
}

// StoreWebhooksDeps holds dependencies for the Store Webhooks Lambda
type StoreWebhooksDeps struct {
	Config  *config.Config
	Handler *store.WebhookHandler
}

// StoreEventsDeps holds dependencies for the Store Events Lambda
type StoreEventsDeps struct {
	Config  *config.Config
	Handler *store.EventsHandler
}

// ============================================================================
// B2C STORE INJECTOR FUNCTIONS
// ============================================================================

// InitializeStoreAuthDeps creates Store Auth Lambda dependencies
func InitializeStoreAuthDeps(ctx context.Context, cfg *config.Config) (*StoreAuthDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideOTPRepository,
		ProvideCustomerRepository,
		ProvideCustomerTokenStore,
		ProvideCustomerAuthService,
		// Cart deps needed for guest merge in auth handler
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideCartService,
		ProvideStoreAuthHandler,
		ProvideCustomerAuthMiddleware,
		wire.Struct(new(StoreAuthDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreCatalogDeps creates Store Catalog Lambda dependencies
func InitializeStoreCatalogDeps(ctx context.Context, cfg *config.Config) (*StoreCatalogDeps, error) {
	wire.Build(
		CoreSet,
		ProvideCategoryRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideS3Client,
		ProvideLambdaClient,
		ProvideEmbedderClient,
		ProvideAssetService,
		ProvideCategoryService,
		ProvideProductService,
		ProvideStoreInventoryService,
		ProvideStoreCatalogHandler,
		wire.Struct(new(StoreCatalogDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreCartDeps creates Store Cart Lambda dependencies
func InitializeStoreCartDeps(ctx context.Context, cfg *config.Config) (*StoreCartDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideOTPRepository,
		ProvideCustomerRepository,
		ProvideCustomerTokenStore,
		ProvideCustomerAuthService,
		ProvideCartService,
		ProvideStoreCartHandler,
		ProvideOptionalCartAuth,
		wire.Struct(new(StoreCartDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreCheckoutDeps creates Store Checkout Lambda dependencies
func InitializeStoreCheckoutDeps(ctx context.Context, cfg *config.Config) (*StoreCheckoutDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvidePaymentRepository,
		ProvideOTPRepository,
		ProvideCustomerTokenStore,
		ProvideCustomerAuthService,
		ProvideCartService,
		ProvidePhonePeGateway,
		ProvidePaymentService,
		ProvideCheckoutService,
		ProvideStoreCheckoutHandler,
		ProvideCustomerAuthMiddleware,
		wire.Struct(new(StoreCheckoutDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreOrdersDeps creates Store Orders Lambda dependencies
func InitializeStoreOrdersDeps(ctx context.Context, cfg *config.Config) (*StoreOrdersDeps, error) {
	wire.Build(
		CoreSet,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvidePricingRuleRepository,
		ProvidePriceQuoteRepository,
		ProvideCategoryRepository,
		ProvideOTPRepository,
		ProvideCustomerTokenStore,
		ProvideCustomerAuthService,
		ProvidePricingService,
		ProvidePaymentRepository,
		ProvideOrderService,
		ProvideStoreOrderHandler,
		ProvideCustomerAuthMiddleware,
		wire.Struct(new(StoreOrdersDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreTrackingDeps creates Store Tracking Lambda dependencies
func InitializeStoreTrackingDeps(ctx context.Context, cfg *config.Config) (*StoreTrackingDeps, error) {
	wire.Build(
		CoreSet,
		ProvideOrderRepository,
		ProvideShipmentRepository,
		ProvideStoreTrackingHandler,
		wire.Struct(new(StoreTrackingDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreProfileDeps creates Store Profile Lambda dependencies
func InitializeStoreProfileDeps(ctx context.Context, cfg *config.Config) (*StoreProfileDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideCustomerRepository,
		ProvideOTPRepository,
		ProvideCustomerTokenStore,
		ProvideCustomerAuthService,
		ProvideStoreProfileHandler,
		ProvideCustomerAuthMiddleware,
		wire.Struct(new(StoreProfileDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreWebhooksDeps creates Store Webhooks Lambda dependencies
func InitializeStoreWebhooksDeps(ctx context.Context, cfg *config.Config) (*StoreWebhooksDeps, error) {
	wire.Build(
		CoreSet,
		ProvidePaymentRepository,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideInventoryRepository,
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideCartService,
		ProvidePhonePeGateway,
		ProvidePaymentService,
		ProvideRefundRepository,
		ProvideUserRepository,
		ProvideNotificationRepository,
		ProvideNotificationService,
		ProvideStoreRefundService,
		ProvideStoreWebhookHandler,
		wire.Struct(new(StoreWebhooksDeps), "*"),
	)
	return nil, nil
}

// InitializeStoreEventsDeps creates Store Events Lambda dependencies.
// Needs PG for lazy-upsert into city_centroids on first sighting of a
// (city, country) pair. Metric counters themselves still flow via the
// SQS publisher initialised in bootstrap.
func InitializeStoreEventsDeps(ctx context.Context, cfg *config.Config) (*StoreEventsDeps, error) {
	wire.Build(
		ProvidePostgresPool,
		ProvideValidator,
		ProvideValidation,
		ProvideCentroidsRepository,
		ProvideStoreEventsHandler,
		wire.Struct(new(StoreEventsDeps), "*"),
	)
	return nil, nil
}

// MonolithDeps contains every dependency the monolith API server needs.
type MonolithDeps struct {
	// PostgresPool retained for graceful shutdown — DynamoDB SDK v2 needs none.
	PostgresPool *pgxpool.Pool

	// Admin handlers
	AuthHandler         *handler.AuthHandler
	UserHandler         *handler.UserHandler
	CategoryHandler     *handler.CategoryHandler
	ProductHandler      *handler.ProductHandler
	InventoryHandler    *handler.InventoryHandler
	PricingHandler      *handler.PricingHandler
	OrderHandler        *handler.OrderHandler
	CustomerHandler     *handler.CustomerHandler
	AuditHandler        *handler.AuditHandler
	NotificationHandler *handler.NotificationHandler
	CouponHandler       *handler.CouponHandler
	UTMLinkHandler      *handler.UTMLinkHandler

	AssetHandler  *handler.AssetHandler
	ReportHandler *handler.ReportHandler

	// Store handlers
	StoreAuthHandler     *store.AuthHandler
	StoreCatalogHandler  *store.CatalogHandler
	StoreCartHandler     *store.CartHandler
	StoreCheckoutHandler *store.CheckoutHandler
	StoreOrderHandler    *store.OrderHandler
	StoreTrackingHandler *store.TrackingHandler
	StoreProfileHandler  *store.ProfileHandler
	StoreWebhookHandler  *store.WebhookHandler
	StoreEventsHandler   *store.EventsHandler

	// Middleware
	AuthMiddleware         *middleware.Auth
	CustomerAuthMiddleware *middleware.CustomerAuth
	OptionalCartAuth       *middleware.OptionalCartAuth
}

// InitializeMonolithDeps wires the full monolith dependency graph.
func InitializeMonolithDeps(ctx context.Context, cfg *config.Config) (*MonolithDeps, error) {
	wire.Build(
		CoreSet,
		ProvideS3Client,
		ProvideLambdaClient,
		RepositorySet,
		StoreRepositorySet,
		ServiceSet,
		StoreServiceSet,
		HandlerSet,
		StoreHandlerSet,
		MiddlewareSet,
		StoreMiddlewareSet,
		wire.Struct(new(MonolithDeps), "*"),
	)
	return nil, nil
}

// ============================================================================
// BACKFILL DEPENDENCY CONTAINER
// ============================================================================

// BackfillDeps holds the minimal dependencies needed by the backfill Lambda
// (T19) to iterate over all products and upsert their embeddings.
type BackfillDeps struct {
	Config         *config.Config
	Pool           *pgxpool.Pool
	EmbedderClient *embedder.Client
}

// InitializeBackfillDeps creates the backfill Lambda dependencies.
func InitializeBackfillDeps(ctx context.Context, cfg *config.Config) (*BackfillDeps, error) {
	wire.Build(
		ProvidePostgresPool,
		ProvideEmbedderClient,
		wire.Struct(new(BackfillDeps), "*"),
	)
	return nil, nil
}

// ============================================================================
// METRICS CONSUMER DEPENDENCY CONTAINER
// ============================================================================

// MetricsConsumerDeps holds dependencies for the metrics consumer Lambda.
type MetricsConsumerDeps struct {
	Handler *metricsworker.Handler
}

// InitializeMetricsConsumerDeps creates the metrics consumer Lambda dependencies.
func InitializeMetricsConsumerDeps(ctx context.Context, cfg *config.Config) (*MetricsConsumerDeps, error) {
	wire.Build(
		ProvidePostgresPool,
		MetricsConsumerSet,
		wire.Struct(new(MetricsConsumerDeps), "*"),
	)
	return nil, nil
}
