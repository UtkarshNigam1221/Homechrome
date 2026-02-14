//go:build wireinject
// +build wireinject

// Package wire provides dependency injection using Google Wire
package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/pkg/logger"
)

// ============================================================================
// SERVICE-SPECIFIC DEPENDENCY CONTAINERS
// ============================================================================

// AuthDeps holds dependencies for the Auth Lambda
type AuthDeps struct {
	Config  *config.Config
	Logger  *logger.Logger
	Handler *handler.AuthHandler
}

// UserDeps holds dependencies for the User Lambda
type UserDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.UserHandler
	AuthMiddleware *middleware.Auth
}

// CatalogDeps holds dependencies for the Catalog Lambda
type CatalogDeps struct {
	Config          *config.Config
	Logger          *logger.Logger
	CategoryHandler *handler.CategoryHandler
	ProductHandler  *handler.ProductHandler
	AuthMiddleware  *middleware.Auth
}

// OrderDeps holds dependencies for the Order Lambda
type OrderDeps struct {
	Config          *config.Config
	Logger          *logger.Logger
	OrderHandler    *handler.OrderHandler
	CustomerHandler *handler.CustomerHandler
	AuthMiddleware  *middleware.Auth
}

// PricingDeps holds dependencies for the Pricing Lambda
type PricingDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.PricingHandler
	AuthMiddleware *middleware.Auth
}

// InventoryDeps holds dependencies for the Inventory Lambda
type InventoryDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.InventoryHandler
	AuthMiddleware *middleware.Auth
}

// AnalyticsDeps holds dependencies for the Analytics Lambda
type AnalyticsDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.AnalyticsHandler
	AuthMiddleware *middleware.Auth
}

// NotificationDeps holds dependencies for the Notification Lambda
type NotificationDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.NotificationHandler
	AuthMiddleware *middleware.Auth
}

// CouponDeps holds dependencies for the Coupon Lambda
type CouponDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.CouponHandler
	AuthMiddleware *middleware.Auth
}

// ArtisanDeps holds dependencies for the Artisan Lambda
type ArtisanDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.ArtisanHandler
	AuthMiddleware *middleware.Auth
}

// BulkDeps holds dependencies for the Bulk Lambda
type BulkDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.BulkHandler
	AuthMiddleware *middleware.Auth
}

// AssetDeps holds dependencies for the Asset Lambda
type AssetDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.AssetHandler
	AuthMiddleware *middleware.Auth
}

// ReportDeps holds dependencies for the Report Lambda
type ReportDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.ReportHandler
	AuthMiddleware *middleware.Auth
}

// AuditDeps holds dependencies for the Audit Lambda
type AuditDeps struct {
	Config         *config.Config
	Logger         *logger.Logger
	Handler        *handler.AuditHandler
	AuthMiddleware *middleware.Auth
}

// ============================================================================
// INJECTOR FUNCTIONS
// ============================================================================

// InitializeAuthDeps creates Auth Lambda dependencies
func InitializeAuthDeps(ctx context.Context, cfg *config.Config) (*AuthDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuthService,
		ProvideAuthHandler,
		wire.Struct(new(AuthDeps), "*"),
	)
	return nil, nil
}

// InitializeUserDeps creates User Lambda dependencies
func InitializeUserDeps(ctx context.Context, cfg *config.Config) (*UserDeps, error) {
	wire.Build(
		CoreSet,
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
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideCategoryRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideS3Client,
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
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideAuthService,
		ProvideOrderService,
		ProvideCustomerService,
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
		ProvideProductRepository,
		ProvideAuthService,
		ProvideInventoryService,
		ProvideInventoryHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(InventoryDeps), "*"),
	)
	return nil, nil
}

// InitializeAnalyticsDeps creates Analytics Lambda dependencies
func InitializeAnalyticsDeps(ctx context.Context, cfg *config.Config) (*AnalyticsDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAnalyticsRepository,
		ProvideAuthService,
		ProvideAnalyticsService,
		ProvideAnalyticsHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(AnalyticsDeps), "*"),
	)
	return nil, nil
}

// InitializeNotificationDeps creates Notification Lambda dependencies
func InitializeNotificationDeps(ctx context.Context, cfg *config.Config) (*NotificationDeps, error) {
	wire.Build(
		CoreSet,
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

// InitializeArtisanDeps creates Artisan Lambda dependencies
func InitializeArtisanDeps(ctx context.Context, cfg *config.Config) (*ArtisanDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideArtisanRepository,
		ProvideAuthService,
		ProvideArtisanService,
		ProvideArtisanHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(ArtisanDeps), "*"),
	)
	return nil, nil
}

// InitializeBulkDeps creates Bulk Lambda dependencies
func InitializeBulkDeps(ctx context.Context, cfg *config.Config) (*BulkDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideBulkOperationRepository,
		ProvideProductRepository,
		ProvideCategoryRepository,
		ProvideInventoryRepository,
		ProvideS3Client,
		ProvideAuthService,
		ProvideAssetService,
		ProvideProductService,
		ProvideInventoryService,
		ProvideBulkService,
		ProvideBulkHandler,
		ProvideAuthMiddleware,
		wire.Struct(new(BulkDeps), "*"),
	)
	return nil, nil
}

// InitializeAssetDeps creates Asset Lambda dependencies
func InitializeAssetDeps(ctx context.Context, cfg *config.Config) (*AssetDeps, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideS3Client,
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
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideReportRepository,
		ProvideOrderRepository,
		ProvideProductRepository,
		ProvideCategoryRepository,
		ProvideInventoryRepository,
		ProvideCustomerRepository,
		ProvidePricingRuleRepository,
		ProvidePriceQuoteRepository,
		ProvideAnalyticsRepository,
		ProvideS3Client,
		ProvideAuthService,
		ProvideAssetService,
		ProvideProductService,
		ProvideInventoryService,
		ProvideCategoryService,
		ProvideCustomerService,
		ProvidePricingService,
		ProvideOrderService,
		ProvideAnalyticsService,
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
// LEGACY APP STRUCT (for backwards compatibility)
// ============================================================================

// App holds all application dependencies (deprecated, use service-specific deps)
type App struct {
	Config         *config.Config
	Logger         *logger.Logger
	DynamoDBClient *dynamodb.Client
	AuthMiddleware *middleware.Auth
}

// InitializeApp creates the application with all dependencies (deprecated)
func InitializeApp(ctx context.Context, cfg *config.Config) (*App, error) {
	wire.Build(
		CoreSet,
		ProvideUserRepository,
		ProvideTokenStore,
		ProvideAuthService,
		ProvideAuthMiddleware,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
