package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// MetricsStackProps holds properties for the metrics SQS + consumer Lambda stack.
type MetricsStackProps struct {
	awscdk.StackProps
	Environment string

	// DatabaseStack provides the Neon Postgres DSN that the consumer Lambda
	// uses to UPSERT aggregated metric counters.
	DatabaseStack *DatabaseStack

	// LogsStack provides the shared WorkerLogGroup so consumer log output
	// stays grouped with the other async workers.
	LogsStack *LogsStack

	// OTel collector layer + Grafana Cloud SSM endpoints. When all three are
	// set, the consumer Lambda ships traces + logs to Grafana Cloud via the
	// extension layer. Empty values degrade gracefully (no telemetry).
	CollectorLayerArn       string
	GrafanaAuthSSMParam     string
	GrafanaEndpointSSMParam string
}

// MetricsStack contains the SQS queue, DLQ, and consumer Lambda for the
// PostgreSQL-backed metrics pipeline.
type MetricsStack struct {
	awscdk.Stack
	Queue      awssqs.Queue
	DLQ        awssqs.Queue
	ConsumerFn awslambda.Function
}

// NewMetricsStack creates the metrics SQS queue + DLQ + consumer Lambda. Other
// Lambdas (api.go services) get the queue URL injected as METRICS_QUEUE_URL
// plus SendMessage permission so they can publish without needing a reference
// to this stack at runtime.
func NewMetricsStack(scope constructs.Construct, id string, props *MetricsStackProps) *MetricsStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Memory matches the worker pattern used in events.go for free-tier friendliness.
	memorySize := float64(128)
	if isProd {
		memorySize = 256
	}

	// Worker Lambdas share the WorkerLogGroup.
	var workerLogGroup awslogs.ILogGroup
	if props.LogsStack != nil {
		workerLogGroup = props.LogsStack.WorkerLogGroup
	}

	// --- DLQ ---
	// 14d retention gives operators time to spot and replay failures.
	dlq := awssqs.NewQueue(stack, jsii.String("MetricsEventsDLQ"), &awssqs.QueueProps{
		QueueName:       jsii.String(fmt.Sprintf("handloom-metrics-events-dlq-%s", props.Environment)),
		RetentionPeriod: awscdk.Duration_Days(jsii.Number(14)),
	})

	// --- Main queue ---
	// VisibilityTimeout ≥ Lambda Timeout (consumer below = 30s) so partial
	// retries don't pick up still-running work. Long-poll (20s) reduces
	// empty-receive costs while still letting EventSourceMapping batch.
	queue := awssqs.NewQueue(stack, jsii.String("MetricsEventsQueue"), &awssqs.QueueProps{
		QueueName:              jsii.String(fmt.Sprintf("handloom-metrics-events-%s", props.Environment)),
		RetentionPeriod:        awscdk.Duration_Days(jsii.Number(14)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(60)),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(3),
			Queue:           dlq,
		},
	})

	// --- Consumer Lambda ---
	// The consumer reads SQS batches (up to 100 messages, 30s window) and
	// UPSERTs into metric_counters via the wire-DI'd handler. It does NOT
	// emit metrics itself, so METRICS_QUEUE_URL is intentionally absent.
	env := map[string]*string{
		"APP_ENV":      jsii.String(props.Environment),
		"APP_DEBUG":    jsii.String(fmt.Sprintf("%t", !isProd)),
		"POSTGRES_DSN": props.DatabaseStack.PostgresDSN,
		"SERVICE_NAME": jsii.String("handloom-worker-metrics-consumer"),
	}

	consumer := awslambda.NewFunction(stack, jsii.String("MetricsConsumerFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("handloom-worker-metrics-consumer-%s", props.Environment)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String("../bin/lambda/worker-metrics-consumer"), &awss3assets.AssetOptions{}),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(memorySize),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Environment:  &env,
		LogGroup:     workerLogGroup,
		Tracing:      awslambda.Tracing_DISABLED,
	})

	// Attach OTel collector layer + env so consumer logs ship to Grafana Loki
	// (and traces to Tempo). Mirrors APIStack.applyTelemetry. Skipped when
	// CollectorLayerArn or Grafana SSM params are empty.
	if props.CollectorLayerArn != "" && props.GrafanaAuthSSMParam != "" && props.GrafanaEndpointSSMParam != "" {
		collectorLayer := awslambda.LayerVersion_FromLayerVersionArn(
			stack,
			jsii.String("MetricsConsumerCollectorLayer"),
			jsii.String(props.CollectorLayerArn),
		)
		consumer.AddLayers(collectorLayer)
		consumer.AddEnvironment(jsii.String("OTEL_SERVICE_NAME"),
			jsii.String("handloom-worker-metrics-consumer"), nil)
		consumer.AddEnvironment(jsii.String("OTEL_RESOURCE_ATTRIBUTES"),
			jsii.String("deployment.environment="+props.Environment+",service.namespace=handloom"), nil)
		consumer.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_PROTOCOL"),
			jsii.String("grpc"), nil)
		consumer.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_ENDPOINT"),
			jsii.String("localhost:4317"), nil)
		consumer.AddEnvironment(jsii.String("AWS_LAMBDA_EXEC_WRAPPER"),
			jsii.String("/opt/otel-handler"), nil)
		consumer.AddEnvironment(jsii.String("OPENTELEMETRY_COLLECTOR_CONFIG_URI"),
			jsii.String("/var/task/otel.yaml"), nil)
		consumer.AddEnvironment(jsii.String("GRAFANA_OTLP_ENDPOINT"),
			jsii.String("{{resolve:ssm:"+props.GrafanaEndpointSSMParam+"}}"), nil)
		consumer.AddEnvironment(jsii.String("GRAFANA_OTLP_AUTH"),
			jsii.String("{{resolve:ssm:"+props.GrafanaAuthSSMParam+"}}"), nil)
	}

	// SQS event source: batch up to 100 with a 30s window to amortize PG
	// round-trips. ReportBatchItemFailures lets the handler return
	// per-message failures without re-delivering the whole batch.
	consumer.AddEventSource(awslambdaeventsources.NewSqsEventSource(queue, &awslambdaeventsources.SqsEventSourceProps{
		BatchSize:               jsii.Number(100),
		MaxBatchingWindow:       awscdk.Duration_Seconds(jsii.Number(30)),
		ReportBatchItemFailures: jsii.Bool(true),
		Enabled:                 jsii.Bool(true),
	}))

	// CfnOutput so the queue URL is discoverable post-deploy.
	awscdk.NewCfnOutput(stack, jsii.String("MetricsQueueURL"), &awscdk.CfnOutputProps{
		Value:       queue.QueueUrl(),
		Description: jsii.String("URL of the metrics events SQS queue"),
		ExportName:  jsii.String(fmt.Sprintf("handloom-metrics-queue-url-%s", props.Environment)),
	})

	awscdk.NewCfnOutput(stack, jsii.String("MetricsDLQURL"), &awscdk.CfnOutputProps{
		Value:       dlq.QueueUrl(),
		Description: jsii.String("URL of the metrics events DLQ"),
		ExportName:  jsii.String(fmt.Sprintf("handloom-metrics-dlq-url-%s", props.Environment)),
	})

	return &MetricsStack{
		Stack:      stack,
		Queue:      queue,
		DLQ:        dlq,
		ConsumerFn: consumer,
	}
}
