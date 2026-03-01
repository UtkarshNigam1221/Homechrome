// Package config provides application configuration
package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	AWS      AWSConfig
	DynamoDB DynamoDBConfig
	Postgres PostgresConfig
	JWT      JWTConfig
	App      AppConfig
	Store    StoreConfig
	Event    EventConfig
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	DSN          string // Direct DSN (local dev): env POSTGRES_DSN
	SecretARN    string // Secrets Manager ARN (Lambda): env RDS_SECRET_ARN
	Endpoint     string // RDS endpoint (Lambda): env RDS_ENDPOINT
	Port         string // RDS port (Lambda): env RDS_PORT
	DatabaseName string // RDS database name (Lambda): env RDS_DATABASE
}

// StoreConfig holds B2C storefront configuration
type StoreConfig struct {
	// PhonePe
	PhonePeMerchantID  string
	PhonePeSaltKey     string
	PhonePeSaltIndex   string
	PhonePeBaseURL     string
	PhonePeCallbackURL string
	PhonePeRedirectURL string

	// Shiprocket
	ShiprocketEmail         string
	ShiprocketPassword      string
	ShiprocketBaseURL       string
	ShiprocketPickupPincode string

	// MSG91
	MSG91AuthKey       string
	MSG91OTPTemplateID string
	MSG91BaseURL       string

	// Customer Auth
	CustomerJWTSecret       string
	CustomerAccessTokenTTL  time.Duration
	CustomerRefreshTokenTTL time.Duration
}

// EventConfig holds event bus configuration
type EventConfig struct {
	SNSTopicARN string
	Enabled     bool
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
}

// DynamoDBConfig holds DynamoDB table names
type DynamoDBConfig struct {
	CoreTable          string // User, PricingRule, Coupon
	OrdersTable        string
	SessionsTable      string
	AuditTable         string
	AnalyticsTable     string // + Report
	NotificationsTable string // Notification
	EventsTable        string // Raw event store
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
	return &Config{
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
		},
		DynamoDB: DynamoDBConfig{
			CoreTable:          getEnv("DYNAMODB_CORE_TABLE", "handloom-core"),
			OrdersTable:        getEnv("DYNAMODB_ORDERS_TABLE", "handloom-orders"),
			SessionsTable:      getEnv("DYNAMODB_SESSIONS_TABLE", "handloom-sessions"),
			AuditTable:         getEnv("DYNAMODB_AUDIT_TABLE", "handloom-audit"),
			AnalyticsTable:     getEnv("DYNAMODB_ANALYTICS_TABLE", "handloom-analytics"),
			NotificationsTable: getEnv("DYNAMODB_NOTIFICATIONS_TABLE", "handloom-notifications"),
			EventsTable:        getEnv("DYNAMODB_EVENTS_TABLE", "handloom-events"),
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
			DSN:          getEnv("POSTGRES_DSN", ""),
			SecretARN:    getEnv("RDS_SECRET_ARN", ""),
			Endpoint:     getEnv("RDS_ENDPOINT", ""),
			Port:         getEnv("RDS_PORT", "5432"),
			DatabaseName: getEnv("RDS_DATABASE", "handloom"),
		},
		Event: EventConfig{
			SNSTopicARN: getEnv("SNS_TOPIC_ARN", ""),
			Enabled:     getBoolEnv("EVENT_PUBLISHING_ENABLED", false),
		},
		Store: StoreConfig{
			PhonePeMerchantID:  getEnv("PHONEPE_MERCHANT_ID", ""),
			PhonePeSaltKey:     getEnv("PHONEPE_SALT_KEY", ""),
			PhonePeSaltIndex:   getEnv("PHONEPE_SALT_INDEX", "1"),
			PhonePeBaseURL:     getEnv("PHONEPE_BASE_URL", "https://api-preprod.phonepe.com/apis/pg-sandbox"),
			PhonePeCallbackURL: getEnv("PHONEPE_CALLBACK_URL", ""),
			PhonePeRedirectURL: getEnv("PHONEPE_REDIRECT_URL", ""),

			ShiprocketEmail:         getEnv("SHIPROCKET_EMAIL", ""),
			ShiprocketPassword:      getEnv("SHIPROCKET_PASSWORD", ""),
			ShiprocketBaseURL:       getEnv("SHIPROCKET_BASE_URL", "https://apiv2.shiprocket.in/v1/external"),
			ShiprocketPickupPincode: getEnv("SHIPROCKET_PICKUP_PINCODE", "560001"),

			MSG91AuthKey:       getEnv("MSG91_AUTH_KEY", ""),
			MSG91OTPTemplateID: getEnv("MSG91_OTP_TEMPLATE_ID", ""),
			MSG91BaseURL:       getEnv("MSG91_BASE_URL", "https://control.msg91.com"),

			CustomerJWTSecret:       getEnv("CUSTOMER_JWT_SECRET", "customer-secret-change-in-production"),
			CustomerAccessTokenTTL:  getDurationEnv("CUSTOMER_ACCESS_TOKEN_TTL", 15*time.Minute),
			CustomerRefreshTokenTTL: getDurationEnv("CUSTOMER_REFRESH_TOKEN_TTL", 30*24*time.Hour),
		},
	}
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
		return "your-super-secret-key-change-in-production"
	}

	// Fetch from SSM Parameter Store
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		// Fall back to default if we can't load AWS config
		return "your-super-secret-key-change-in-production"
	}

	ssmClient := ssm.NewFromConfig(cfg)
	result, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &paramName,
		WithDecryption: boolPtr(true),
	})
	if err != nil {
		// Fall back to default if parameter fetch fails
		return "your-super-secret-key-change-in-production"
	}

	if result.Parameter != nil && result.Parameter.Value != nil {
		return *result.Parameter.Value
	}

	return "your-super-secret-key-change-in-production"
}

func boolPtr(b bool) *bool {
	return &b
}
