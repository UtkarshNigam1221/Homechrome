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
	// CERT_ARN override lets a fresh account inject its own ACM cert without a
	// code change (the baked ARNs belong to one account and pass validate() but
	// fail at deploy on any other account).
	if v := os.Getenv("CERT_ARN"); v != "" {
		cfg.CertArn = v
	}

	postgresDSN := getPostgresDSN(app)

	commonTags := &map[string]*string{
		"Environment": jsii.String(environment),
		"Project":     jsii.String("handloom-admin"),
		"ManagedBy":   jsii.String("cdk"),
	}

	// Two independent app targets selected by `-c app=` (or CDK_APP_TARGET):
	//   app=embedder → ONLY the standalone EmbedderStack (own log group; metrics
	//     queue + Postgres referenced by name/string). Zero cross-stack refs, so
	//     it synthesizes + deploys with no other stack present and no wasted
	//     ImageResizer/active-lambda asset builds.
	//   default (backend) → Logs/Database/Storage/Metrics/API. APIStack imports
	//     the embedder BY NAME, so the embedder must already be deployed on a
	//     fresh env (embedder-first ordering; no CDK construct edge enforces it).
	switch appTarget(app) {
	case "embedder":
		buildEmbedderApp(app, env, environment, cfg, postgresDSN, commonTags)
	default:
		buildBackendApp(app, env, environment, cfg, postgresDSN, commonTags)
	}

	app.Synth(nil)
}

// buildEmbedderApp instantiates only the standalone embedder stack.
func buildEmbedderApp(app awscdk.App, env *awscdk.Environment, environment string, cfg EnvConfig, postgresDSN string, tags *map[string]*string) {
	// Image built inline from cmd/embedder/Dockerfile via Code_FromAssetImage;
	// cmd/embedder/assets/ must be populated (make prepare-embedder-assets) first.
	stacks.NewEmbedderStack(app, "HandloomEmbedderStack-"+environment, &stacks.EmbedderStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Embedder Lambda for hybrid semantic search (" + environment + ")"),
			Tags:        tags,
		},
		Environment:         environment,
		FnName:              cfg.EmbedderFnName,
		StoreFrontHost:      cfg.StoreFrontHost,
		PostgresDSN:         postgresDSN,
		MetricsQueueName:    cfg.MetricsQueueName,
		GrafanaEndpoint:     cfg.GrafanaEndpoint,
		GrafanaAuthSSMParam: "/handloom/" + environment + "/grafana-otlp-auth",
	})
}

// buildBackendApp instantiates the backend stacks (no embedder; API imports it by name).
func buildBackendApp(app awscdk.App, env *awscdk.Environment, environment string, cfg EnvConfig, postgresDSN string, tags *map[string]*string) {
	// Logs stack — owns ApiLogGroup + WorkerLogGroup.
	logsStack := stacks.NewLogsStack(app, "HandloomLogsStack-"+environment, &stacks.LogsStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Shared CloudWatch log groups (" + environment + ")"),
			Tags:        tags,
		},
		Environment: environment,
	})

	databaseStack := stacks.NewDatabaseStack(app, "HandloomDatabaseStack-"+environment, &stacks.DatabaseStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Database resources (" + environment + ")"),
			Tags:        tags,
		},
		Environment: environment,
		PostgresDSN: postgresDSN,
		LogsStack:   logsStack,
	})

	storageStack := stacks.NewStorageStack(app, "HandloomStorageStack-"+environment, &stacks.StorageStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Storage resources (" + environment + ")"),
			Tags:        tags,
		},
		Environment: environment,
	})

	// Community-published OTel Collector layer ARN for ap-south-1 / arm64.
	collectorArn := stacks.OtelCollectorLayerArn("ap-south-1", "arm64")

	metricsStack := stacks.NewMetricsStack(app, "HandloomMetricsStack-"+environment, &stacks.MetricsStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Metrics SQS + consumer Lambda (" + environment + ")"),
			Tags:        tags,
		},
		Environment:         environment,
		QueueName:           cfg.MetricsQueueName,
		DatabaseStack:       databaseStack,
		LogsStack:           logsStack,
		CollectorLayerArn:   collectorArn,
		GrafanaAuthSSMParam: "/handloom/" + environment + "/grafana-otlp-auth",
		GrafanaEndpoint:     cfg.GrafanaEndpoint,
	})

	stacks.NewAPIStack(app, "HandloomAPIStack-"+environment, &stacks.APIStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - API and Lambda functions (" + environment + ")"),
			Tags:        tags,
		},
		Environment:             environment,
		DatabaseStack:           databaseStack,
		StorageStack:            storageStack,
		LogsStack:               logsStack,
		EmbedderFnName:          cfg.EmbedderFnName, // imported by name — no cross-stack ref to the embedder app
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
		GrafanaEndpoint:         cfg.GrafanaEndpoint,
	})
}

// appTarget selects which CDK app to build: "embedder" or "backend" (default).
func appTarget(app constructs.Construct) string {
	if t := app.Node().TryGetContext(jsii.String("app")); t != nil {
		return t.(string)
	}
	if t := os.Getenv("CDK_APP_TARGET"); t != "" {
		return t
	}
	return "backend"
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
