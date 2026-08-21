// Package config provides application configuration
package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/handloom/admin/pkg/metrics/awsmiddleware"
)

// defaultJWTSecret is the fallback JWT secret used for local development.
const defaultJWTSecret = "your-super-secret-key-change-in-production"

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	AWS       AWSConfig
	DynamoDB  DynamoDBConfig
	Postgres  PostgresConfig
	JWT       JWTConfig
	App       AppConfig
	Store     StoreConfig
	Embedder  EmbedderConfig
	Telemetry TelemetryConfig
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	DSN string // Connection string: env POSTGRES_DSN
}

// StoreConfig holds B2C storefront configuration
type StoreConfig struct {
	// PhonePe Standard Checkout
	PhonePeClientID        string
	PhonePeClientSecret    string
	PhonePeClientVersion   string
	PhonePeBaseURL         string
	PhonePeAuthBaseURL     string
	PhonePeCallbackURL     string
	PhonePeRedirectURL     string
	PhonePeWebhookUsername string
	PhonePeWebhookPassword string

	// MSG91
	MSG91AuthKey       string
	MSG91OTPTemplateID string
	MSG91BaseURL       string

	// Customer Auth
	CustomerJWTSecret       string
	CustomerAccessTokenTTL  time.Duration
	CustomerRefreshTokenTTL time.Duration

	// E2E OTP short-circuit. TestPhones is an exact-match E.164 allowlist whose
	// numbers skip the SMS gateway and are issued TestOTP instead of a random
	// code, so an automated suite can log in without sending real SMS. Never a
	// prefix or pattern: an exact match cannot widen by accident.
	//
	// Validate() refuses to start when these are set in production. The prod CDK
	// stack also never sets the variables at all, so console or SSM drift cannot
	// switch this on without a code deploy.
	TestPhones []string
	TestOTP    string
}

// EmbedderConfig holds embedder Lambda configuration (used by catalog + backfill Lambdas).
type EmbedderConfig struct {
	FunctionName string // EMBEDDER_FN_NAME — e.g. handloom-embedder-dev
	AuthKeyParam string // EMBEDDER_AUTH_KEY_PARAM — SSM SecureString path
	TimeoutMs    int    // EMBEDDER_TIMEOUT_MS — default 10000
	ModelVersion string // EMBEDDING_MODEL_VERSION — default l3cube-indic-sbert-nli-v1
}

// TelemetryConfig holds OpenTelemetry-related runtime settings.
type TelemetryConfig struct {
	ServiceName    string // OTEL_SERVICE_NAME — set per-Lambda by CDK
	ServiceVersion string // OTEL_SERVICE_VERSION — git short SHA at build time
	Environment    string // matches App.Environment
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// AWSConfig holds AWS configuration
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // For local development (empty in production)
	S3Bucket        string
	CDNUrl          string
	ImageResizerFn  string // Lambda function name (e.g. homechrome-image-resizer-dev)
}

// DynamoDBConfig holds DynamoDB table names
type DynamoDBConfig struct {
	CoreTable          string // User, PricingRule, Coupon
	OrdersTable        string
	SessionsTable      string
	AuditTable         string
	NotificationsTable string // Notification
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Environment      string
	Debug            bool
	QuoteValidityHrs int
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "ap-south-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("AWS_ENDPOINT", ""), // Empty for production
			S3Bucket:        getEnvAny([]string{"S3_ASSETS_BUCKET", "AWS_S3_BUCKET"}, "handloom-assets"),
			CDNUrl:          getEnvAny([]string{"CDN_DOMAIN", "AWS_CDN_URL"}, ""),
			ImageResizerFn:  getEnvAny([]string{"IMAGE_RESIZER_FUNCTION_NAME"}, ""),
		},
		DynamoDB: DynamoDBConfig{
			CoreTable:          getEnv("DYNAMODB_CORE_TABLE", "handloom-core"),
			OrdersTable:        getEnv("DYNAMODB_ORDERS_TABLE", "handloom-orders"),
			SessionsTable:      getEnv("DYNAMODB_SESSIONS_TABLE", "handloom-sessions"),
			AuditTable:         getEnv("DYNAMODB_AUDIT_TABLE", "handloom-audit"),
			NotificationsTable: getEnv("DYNAMODB_NOTIFICATIONS_TABLE", "handloom-notifications"),
		},
		JWT: JWTConfig{
			SecretKey:            getJWTSecret(),
			AccessTokenDuration:  getDurationEnv("JWT_ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: getDurationEnv("JWT_REFRESH_TOKEN_DURATION", 7*24*time.Hour),
			Issuer:               getEnv("JWT_ISSUER", "handloom-admin"),
		},
		App: AppConfig{
			Environment:      getEnv("APP_ENV", "development"),
			Debug:            getBoolEnv("APP_DEBUG", true),
			QuoteValidityHrs: getIntEnv("QUOTE_VALIDITY_HRS", 24),
		},
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", ""),
		},
		Embedder: EmbedderConfig{
			FunctionName: getEnv("EMBEDDER_FN_NAME", ""),
			AuthKeyParam: getEnv("EMBEDDER_AUTH_KEY_PARAM", ""),
			TimeoutMs:    getIntEnv("EMBEDDER_TIMEOUT_MS", 10000),
			ModelVersion: getEnv("EMBEDDING_MODEL_VERSION", "l3cube-indic-sbert-nli-v1"),
		},
		Store: StoreConfig{
			PhonePeClientID:        getEnv("PHONEPE_CLIENT_ID", ""),
			PhonePeClientSecret:    getEnv("PHONEPE_CLIENT_SECRET", ""),
			PhonePeClientVersion:   getEnv("PHONEPE_CLIENT_VERSION", "1"),
			PhonePeBaseURL:         getEnv("PHONEPE_BASE_URL", ""),
			PhonePeAuthBaseURL:     getEnv("PHONEPE_AUTH_BASE_URL", ""),
			PhonePeCallbackURL:     getEnv("PHONEPE_CALLBACK_URL", ""),
			PhonePeRedirectURL:     getEnv("PHONEPE_REDIRECT_URL", ""),
			PhonePeWebhookUsername: getEnv("PHONEPE_WEBHOOK_USERNAME", ""),
			PhonePeWebhookPassword: getEnv("PHONEPE_WEBHOOK_PASSWORD", ""),

			MSG91AuthKey:       getEnv("MSG91_AUTH_KEY", ""),
			MSG91OTPTemplateID: getEnv("MSG91_OTP_TEMPLATE_ID", ""),
			MSG91BaseURL:       getEnv("MSG91_BASE_URL", "https://control.msg91.com"),

			CustomerJWTSecret:       getEnv("CUSTOMER_JWT_SECRET", "customer-secret-change-in-production"),
			CustomerAccessTokenTTL:  getDurationEnv("CUSTOMER_ACCESS_TOKEN_TTL", 15*time.Minute),
			CustomerRefreshTokenTTL: getDurationEnv("CUSTOMER_REFRESH_TOKEN_TTL", 30*24*time.Hour),

			TestPhones: splitAndTrim(getEnv("STORE_TEST_PHONES", "")),
			TestOTP:    getEnv("STORE_TEST_OTP", ""),
		},
	}
	cfg.Telemetry = TelemetryConfig{
		ServiceName:    getEnv("OTEL_SERVICE_NAME", "handloom-unknown"),
		ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "dev"),
		Environment:    cfg.App.Environment,
	}
	return cfg
}

// splitAndTrim turns a comma-separated env value into a slice, dropping empties
// so a trailing comma or a blank variable yields no entries rather than one
// empty entry that could match an empty phone.
func splitAndTrim(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Validate refuses to start on a configuration that is safe in dev and unsafe
// in production. Called from Load's callers at startup, so a misconfigured
// Lambda dies on cold start rather than serving traffic with a test bypass
// live. Loud and immediate beats subtle and later.
func (c *Config) Validate() error {
	if c.App.Environment != "prod" && c.App.Environment != "production" {
		return nil
	}
	if len(c.Store.TestPhones) > 0 {
		return fmt.Errorf("STORE_TEST_PHONES is set in %s: the OTP test bypass must never be enabled in production", c.App.Environment)
	}
	if c.Store.TestOTP != "" {
		return fmt.Errorf("STORE_TEST_OTP is set in %s: the OTP test bypass must never be enabled in production", c.App.Environment)
	}
	return nil
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsLocal returns true if running locally (with local DynamoDB)
func (c *Config) IsLocal() bool {
	return c.AWS.Endpoint != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAny tries multiple env var keys in order, returning the first non-empty value
func getEnvAny(keys []string, defaultValue string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getJWTSecret gets the JWT secret from SSM Parameter Store or environment variable
func getJWTSecret() string {
	// First check if we have a direct secret key
	if secret := os.Getenv("JWT_SECRET_KEY"); secret != "" {
		return secret
	}

	// Check if we have an SSM parameter name
	paramName := os.Getenv("JWT_SECRET_PARAM")
	if paramName == "" {
		// Default for local development
		return defaultJWTSecret
	}

	// Fetch from SSM Parameter Store
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		// Fall back to default if we can't load AWS config
		return defaultJWTSecret
	}

	awsmiddleware.Instrument(&cfg)

	ssmClient := ssm.NewFromConfig(cfg)
	result, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &paramName,
		WithDecryption: boolPtr(true),
	})
	if err != nil {
		// Fall back to default if parameter fetch fails
		return defaultJWTSecret
	}

	if result.Parameter != nil && result.Parameter.Value != nil {
		return *result.Parameter.Value
	}

	return defaultJWTSecret
}

func boolPtr(b bool) *bool {
	return &b
}
