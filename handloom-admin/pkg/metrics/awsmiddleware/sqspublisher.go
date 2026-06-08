package awsmiddleware

import (
	"context"
	"log/slog"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	pkgmetrics "github.com/handloom/admin/pkg/metrics"
)

// InitSQSMetricsPublisher installs the SQS-backed metrics publisher as the
// process-global publisher when METRICS_QUEUE_URL is set (CDK injects it for
// every publishing Lambda). It is a no-op when the var is unset — the metrics
// consumer Lambda and local dev stay on the global Noop publisher. serviceName
// labels the aws_sdk_call{} metrics emitted by the SQS client.
func InitSQSMetricsPublisher(ctx context.Context, serviceName string) {
	qURL := os.Getenv("METRICS_QUEUE_URL")
	if qURL == "" {
		slog.Info("metrics: METRICS_QUEUE_URL unset; using noop publisher")
		return
	}
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	awsCfg, err := awsconfig.LoadDefaultConfig(initCtx)
	if err != nil {
		slog.Error("metrics: failed to load AWS config; staying on noop publisher", "error", err)
		return
	}
	awsCfg.APIOptions = append(awsCfg.APIOptions, With(serviceName))
	pkgmetrics.SetDefault(pkgmetrics.NewSQSPublisher(sqs.NewFromConfig(awsCfg), qURL))
	// queue_url is operator-supplied env config (METRICS_QUEUE_URL), not user input.
	slog.Info("metrics: SQS publisher initialized", "queue_url", qURL) //nolint:gosec // G706: operator env config, not user input
}
