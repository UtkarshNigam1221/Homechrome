// Package telemetry provides OpenTelemetry instrumentation for tracing and logging.
package telemetry

import (
	"os"
	"time"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	// ServiceName is the name of the service for tracing
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment (development, staging, production)
	Environment string

	// Tracing configuration
	Tracing TracingConfig

	// Logging configuration
	Logging LoggingConfig
}

// TracingConfig holds tracing-specific configuration.
type TracingConfig struct {
	// Enabled enables/disables tracing
	Enabled bool

	// Exporter type: "otlp-grpc", "otlp-http", "stdout", "none"
	Exporter string

	// Endpoint for OTLP exporter (e.g., "localhost:4317" for gRPC, "localhost:4318" for HTTP)
	Endpoint string

	// Insecure disables TLS for the exporter connection
	Insecure bool

	// SampleRate is the sampling rate (0.0 to 1.0, where 1.0 = 100%)
	SampleRate float64

	// BatchTimeout is the maximum time before a batch is exported
	BatchTimeout time.Duration

	// MaxExportBatchSize is the maximum number of spans in a batch
	MaxExportBatchSize int

	// Headers for OTLP exporter (e.g., for authentication)
	Headers map[string]string
}

// LoggingConfig holds logging-specific configuration.
type LoggingConfig struct {
	// TraceCorrelation enables adding trace/span IDs to log entries
	TraceCorrelation bool

	// Level is the minimum log level
	Level string
}

// NewConfigFromApp creates a telemetry Config from application configuration values.
// This allows the application to pass its own config instead of reading from env vars.
func NewConfigFromApp(serviceName, serviceVersion, environment, exporter, endpoint string, sampleRate float64, insecure, traceCorrelation bool) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    environment,
		Tracing: TracingConfig{
			Enabled:            true,
			Exporter:           exporter,
			Endpoint:           endpoint,
			Insecure:           insecure,
			SampleRate:         sampleRate,
			BatchTimeout:       5 * time.Second,
			MaxExportBatchSize: 512,
			Headers:            make(map[string]string),
		},
		Logging: LoggingConfig{
			TraceCorrelation: traceCorrelation,
			Level:            "info",
		},
	}
}

// DefaultConfig returns the default OpenTelemetry configuration.
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    getEnv("OTEL_SERVICE_NAME", "handloom-admin"),
		ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("APP_ENV", "development"),
		Tracing: TracingConfig{
			Enabled:            getEnvBool("OTEL_TRACING_ENABLED", true),
			Exporter:           getEnv("OTEL_EXPORTER_TYPE", "stdout"),
			Endpoint:           getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			Insecure:           getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
			SampleRate:         getEnvFloat("OTEL_TRACE_SAMPLE_RATE", 1.0),
			BatchTimeout:       getEnvDuration("OTEL_BATCH_TIMEOUT", 5*time.Second),
			MaxExportBatchSize: getEnvInt("OTEL_MAX_EXPORT_BATCH_SIZE", 512),
			Headers:            parseHeaders(getEnv("OTEL_EXPORTER_OTLP_HEADERS", "")),
		},
		Logging: LoggingConfig{
			TraceCorrelation: getEnvBool("OTEL_LOG_TRACE_CORRELATION", true),
			Level:            getEnv("LOG_LEVEL", "info"),
		},
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	_, err := parseIntValue(value, &result)
	if err != nil {
		return defaultValue
	}
	return result
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result float64
	_, err := parseFloatValue(value, &result)
	if err != nil {
		return defaultValue
	}
	return result
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}

func parseIntValue(s string, target *int) (bool, error) {
	var v int
	n, err := parseFmtInt(s, &v)
	if err != nil || n == 0 {
		return false, err
	}
	*target = v
	return true, nil
}

func parseFmtInt(s string, v *int) (int, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			*v = *v*10 + int(c-'0')
			n++
		} else {
			break
		}
	}
	return n, nil
}

func parseFloatValue(s string, target *float64) (bool, error) {
	var v float64
	var decimal bool
	var divisor float64 = 1
	for _, c := range s {
		if c >= '0' && c <= '9' {
			if decimal {
				divisor *= 10
				v = v + float64(c-'0')/divisor
			} else {
				v = v*10 + float64(c-'0')
			}
		} else if c == '.' && !decimal {
			decimal = true
		} else {
			break
		}
	}
	*target = v
	return true, nil
}

func parseHeaders(s string) map[string]string {
	headers := make(map[string]string)
	if s == "" {
		return headers
	}
	// Parse format: "key1=value1,key2=value2"
	var key, value string
	inKey := true
	for _, c := range s {
		switch c {
		case '=':
			inKey = false
		case ',':
			if key != "" {
				headers[key] = value
			}
			key = ""
			value = ""
			inKey = true
		default:
			if inKey {
				key += string(c)
			} else {
				value += string(c)
			}
		}
	}
	if key != "" {
		headers[key] = value
	}
	return headers
}
