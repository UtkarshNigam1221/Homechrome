package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Shutdown is a closure callers must invoke before process exit to flush
// and stop all telemetry providers.
type Shutdown func(context.Context)

// MustInit boots the tracer provider from env config and returns a shutdown
// closure. Panics on construction failure — telemetry init must be
// deterministic. Honors OTEL_SDK_DISABLED=true as an emergency kill switch.
func MustInit(serviceName, serviceVersion, environment string) Shutdown {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Info("telemetry disabled via OTEL_SDK_DISABLED")
		return func(context.Context) {}
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	cfg := NewConfigFromApp(
		serviceName, serviceVersion, environment,
		"otlp-grpc", endpoint, 1.0, true /*insecure*/, true, /*traceCorrelation*/
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		panic("telemetry: tracer provider: " + err.Error())
	}

	// All attrs are operator-supplied build/env config, not user input.
	slog.Info("telemetry initialized", //nolint:gosec // G706: operator env/build config, not user input
		"version", serviceVersion,
		"environment", environment,
		"endpoint", endpoint,
	)

	return func(ctx context.Context) {
		_ = tp.Shutdown(ctx)
	}
}

// MustInitDirect boots a tracer provider that exports OTLP directly to a given
// endpoint with explicit auth headers — used by the container-image embedder
// Lambda, which cannot use the OTel Collector layer the zip lambdas rely on.
// It returns the provider (so the caller can ForceFlush per-invocation) plus a
// Shutdown closure for process exit. Honors OTEL_SDK_DISABLED=true.
func MustInitDirect(serviceName, serviceVersion, environment, endpoint string, headers map[string]string) (*TracerProvider, Shutdown) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Info("telemetry disabled via OTEL_SDK_DISABLED")
		return nil, func(context.Context) {}
	}

	// Grafana Cloud OTLP gateway: HTTP/protobuf over TLS, always-sample.
	cfg := NewConfigFromApp(
		serviceName, serviceVersion, environment,
		"otlp-http", endpoint, 1.0, false /*insecure*/, true, /*traceCorrelation*/
	)
	cfg.Tracing.Headers = headers // NewConfigFromApp seeds an empty map; set auth here

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		panic("telemetry: tracer provider: " + err.Error())
	}

	slog.Info("telemetry initialized (direct export)", //nolint:gosec // G706: operator env/build config, not user input
		"version", serviceVersion,
		"environment", environment,
		"endpoint", endpoint,
	)

	return tp, func(ctx context.Context) {
		_ = tp.Shutdown(ctx)
	}
}
