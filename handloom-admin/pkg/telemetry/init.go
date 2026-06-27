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
