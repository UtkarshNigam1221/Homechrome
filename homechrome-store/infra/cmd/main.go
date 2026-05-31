package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/homechrome/store/infra/stacks"
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

	commonTags := &map[string]*string{
		"Environment": jsii.String(environment),
		"Project":     jsii.String("homechrome-store"),
		"ManagedBy":   jsii.String("cdk"),
	}

	// Logs stack — owns ServerLogGroup. Created first so the storefront stack
	// can reference it via props.LogsStack.
	logsStack := stacks.NewLogsStack(app, "HomechromeStoreLogsStack-"+environment, &stacks.LogsStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Homechrome Store - Shared CloudWatch log groups (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment: environment,
	})

	// Telemetry — layer ARN and SSM param names are passed at deploy time via env vars.
	// The layer is the community-published OpenTelemetry Collector (account 184161586896).
	// Compute the ARN using stacks.OtelCollectorLayerArn in handloom-admin/infra/stacks/telemetry.go,
	// then pass it here at deploy time:
	//   OTEL_COLLECTOR_LAYER_ARN=arn:aws:lambda:ap-south-1:184161586896:layer:opentelemetry-collector-arm64-0_22_0:1 npm run cdk:deploy:dev
	// Leave unset to skip telemetry (no-op path in StorefrontStack).
	collectorLayerArn := os.Getenv("OTEL_COLLECTOR_LAYER_ARN")
	nodeAutoInstrLayerArn := os.Getenv("NODE_AUTO_INSTR_LAYER_ARN")
	grafanaEndpointSSMParam := os.Getenv("GRAFANA_ENDPOINT_SSM_PARAM")
	if grafanaEndpointSSMParam == "" {
		grafanaEndpointSSMParam = "/handloom/" + environment + "/grafana-otlp-endpoint"
	}
	grafanaAuthSSMParam := os.Getenv("GRAFANA_AUTH_SSM_PARAM")
	if grafanaAuthSSMParam == "" {
		grafanaAuthSSMParam = "/handloom/" + environment + "/grafana-otlp-auth"
	}

	stacks.NewStorefrontStack(app, "HomechromeStoreStack-"+environment, &stacks.StorefrontStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Homechrome Store - Next.js SSR hosting (" + environment + ")"),
			Tags:        commonTags,
		},
		Environment:             environment,
		DomainNames:             cfg.DomainNames,
		CertArn:                 cfg.CertArn,
		BackendApiUrl:           cfg.BackendApiUrl,
		LogsStack:               logsStack,
		CollectorLayerArn:       collectorLayerArn,
		NodeAutoInstrLayerArn:   nodeAutoInstrLayerArn,
		GrafanaEndpointSSMParam: grafanaEndpointSSMParam,
		GrafanaAuthSSMParam:     grafanaAuthSSMParam,
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
