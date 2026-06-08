// Package bootstrap centralises Lambda main-function setup so every
// cmd/lambda/* entry point shares the same init steps.
package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/pkg/metrics/awsmiddleware"
	"github.com/handloom/admin/pkg/slogx"
	"github.com/handloom/admin/pkg/telemetry"
)

// LambdaContext bundles artifacts shared by every Lambda main.
type LambdaContext struct {
	Cfg      *config.Config
	Shutdown func()
}

// InitLambda performs the common pre-handler setup. Callers must defer the
// returned Shutdown before any blocking handler call. The serviceName
// argument overrides anything in env (so an operator forgetting
// OTEL_SERVICE_NAME does not silently brand every signal as
// "handloom-unknown").
func InitLambda(serviceName string) *LambdaContext {
	cfg := config.Load()

	if cfg.Telemetry.ServiceName == "" || cfg.Telemetry.ServiceName == "handloom-unknown" {
		cfg.Telemetry.ServiceName = serviceName
	}

	// Set SERVICE_NAME so slogx.Setup picks it up as a top-level attribute.
	_ = os.Setenv("SERVICE_NAME", cfg.Telemetry.ServiceName)
	slogx.Setup(cfg.App.Debug)

	telShutdown := telemetry.MustInit(
		cfg.Telemetry.ServiceName,
		cfg.Telemetry.ServiceVersion,
		cfg.Telemetry.Environment,
	)

	// Initialise the SQS-backed metrics publisher (no-op when METRICS_QUEUE_URL
	// is unset — e.g. the consumer Lambda, local dev). Shared with the embedder
	// Lambda via awsmiddleware so the wiring lives in one place.
	awsmiddleware.InitSQSMetricsPublisher(context.Background(), cfg.Telemetry.ServiceName)

	// service attribute already attached globally by slogx.Setup; no need to repeat.
	slog.Info("lambda bootstrap done")

	return &LambdaContext{
		Cfg: cfg,
		Shutdown: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			telShutdown(ctx)
		},
	}
}
