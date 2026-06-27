package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	"github.com/handloom/admin/infra/stacks"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	env := getAWSEnv()
	environment := getEnvironment(app)

	cfg, ok := envConfigs[environment]
	if !ok {
		panic(fmt.Sprintf("unknown environment: %s (valid: dev, prod)", environment))
	}
	if err := cfg.validate(environment); err != nil {
		panic(err)
	}

	postgresDSN := getPostgresDSN(app)

	commonTags := &map[string]*string{
		"Environment": jsii.String(environment),
		"Project":     jsii.String("handloom-admin"),
		"ManagedBy":   jsii.String("cdk"),
	}

	// Logs stack — owns ApiLogGroup + WorkerLogGroup. Created first so every
	// downstream stack can reference the same log groups via props.LogsStack.
	logsStack := stacks.NewLogsStack(app, "HandloomLogsStack-"+environment, &stacks.LogsStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Shared CloudWatch log groups (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment: environment,
	})

	// Database stack
	databaseStack := stacks.NewDatabaseStack(app, "HandloomDatabaseStack-"+environment, &stacks.DatabaseStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Database resources (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment: environment,
		PostgresDSN: postgresDSN,
		LogsStack:   logsStack,
	})

	// Storage stack
	storageStack := stacks.NewStorageStack(app, "HandloomStorageStack-"+environment, &stacks.StorageStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Storage resources (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment: environment,
	})

	// Resolve the community-published OTel Collector layer ARN for ap-south-1 / arm64.
	// No separate TelemetryStack needed — the layer is maintained by the OpenTelemetry
	// community and imported directly. See stacks.OtelCollectorLayerArn for version notes.
	collectorArn := stacks.OtelCollectorLayerArn("ap-south-1", "arm64")

	// Metrics stack first so its queue handle can be wired into both the
	// embedder Lambda and the per-service Lambdas behind APIStack for
	// SendMessage + METRICS_QUEUE_URL env injection.
	metricsStack := stacks.NewMetricsStack(app, "HandloomMetricsStack-"+environment, &stacks.MetricsStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Metrics SQS + consumer Lambda (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment:             environment,
		DatabaseStack:           databaseStack,
		LogsStack:               logsStack,
		CollectorLayerArn:       collectorArn,
		GrafanaAuthSSMParam:     "/handloom/" + environment + "/grafana-otlp-auth",
		GrafanaEndpointSSMParam: "/handloom/" + environment + "/grafana-otlp-endpoint",
	})

	// Embedder Lambda stack — image built inline from cmd/embedder/Dockerfile
	// via CDK's Code_FromAssetImage. CDK pushes the resulting image to its
	// bootstrap ECR repo automatically. No custom-managed ECR repo. The
	// download-embedder-assets Make target must have populated
	// cmd/embedder/assets/ before this stack synthesizes.
	embStack := stacks.NewEmbedderStack(app, "HandloomEmbedderStack-"+environment, &stacks.EmbedderStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Embedder Lambda for hybrid semantic search (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment:    environment,
		StoreFrontHost: cfg.StoreFrontHost,
		DatabaseStack:  databaseStack,
		LogsStack:      logsStack,
		MetricsStack:   metricsStack,
	})

	// API stack (depends on database, storage, metrics, and telemetry)
	stacks.NewAPIStack(app, "HandloomAPIStack-"+environment, &stacks.APIStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - API and Lambda functions (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment:             environment,
		DatabaseStack:           databaseStack,
		StorageStack:            storageStack,
		LogsStack:               logsStack,
		EmbedderStack:           embStack,           // Embedder for hybrid semantic search
		MetricsQueue:            metricsStack.Queue, // PG-backed metrics pipeline
		BaseDomain:              cfg.BaseDomain,
		DomainName:              cfg.DomainName,
		FrontendOrigin:          cfg.FrontendOrigin,
		CertArn:                 cfg.CertArn,
		PhonePeBaseURL:          cfg.PhonePeBaseURL,
		PhonePeCallbackURL:      cfg.PhonePeCallbackURL,
		PhonePeRedirectURL:      cfg.PhonePeRedirectURL,
		PhonePeClientVersion:    cfg.PhonePeClientVersion,
		MSG91BaseURL:            cfg.MSG91BaseURL,
		MSG91OTPTemplateID:      cfg.MSG91OTPTemplateID,
		ShiprocketBaseURL:       cfg.ShiprocketBaseURL,
		ShiprocketPickupPincode: cfg.ShiprocketPickupPincode,
		CollectorLayerArn:       collectorArn,
		GrafanaAuthSSMParam:     "/handloom/" + environment + "/grafana-otlp-auth",
		GrafanaEndpointSSMParam: "/handloom/" + environment + "/grafana-otlp-endpoint",
	})

	app.Synth(nil)
}

func getEnvironment(app constructs.Construct) string {
	if env := app.Node().TryGetContext(jsii.String("environment")); env != nil {
		return env.(string)
	}
	if env := os.Getenv("CDK_ENVIRONMENT"); env != "" {
		return env
	}
	return "dev"
}

func getPostgresDSN(app constructs.Construct) string {
	if dsn := app.Node().TryGetContext(jsii.String("postgresDsn")); dsn != nil {
		return dsn.(string)
	}
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return ""
}

func getAWSEnv() *awscdk.Environment {
	account := os.Getenv("CDK_DEFAULT_ACCOUNT")
	region := os.Getenv("CDK_DEFAULT_REGION")

	if account == "" {
		account = os.Getenv("AWS_ACCOUNT_ID")
	}
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = "ap-south-1"
	}

	if account == "" {
		return nil
	}

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}
