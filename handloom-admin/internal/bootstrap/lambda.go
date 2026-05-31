// Package bootstrap centralises Lambda main-function setup so every
// cmd/lambda/* entry point shares the same init steps.
package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/handloom/admin/internal/config"
	pkgmetrics "github.com/handloom/admin/pkg/metrics"
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

	// Initialise the SQS-backed metrics publisher when METRICS_QUEUE_URL is set
	// (CDK MetricsStack + api.go inject this for every service Lambda; consumer
	// Lambda has it unset because it only reads SQS).
	if qURL := os.Getenv("METRICS_QUEUE_URL"); qURL != "" {
		initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		awsCfg, err := awsconfig.LoadDefaultConfig(initCtx)
		cancel()
		if err != nil {
			slog.Error("metrics: failed to load AWS config; staying on noop publisher", "error", err)
		} else {
			awsCfg.APIOptions = append(awsCfg.APIOptions, awsmiddleware.With(cfg.Telemetry.ServiceName))
			pkgmetrics.SetDefault(pkgmetrics.NewSQSPublisher(sqs.NewFromConfig(awsCfg), qURL))
			slog.Info("metrics: SQS publisher initialised", "queue_url", qURL)
		}
	}

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
