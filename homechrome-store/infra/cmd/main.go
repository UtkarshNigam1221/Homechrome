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
	if err := cfg.validate(environment); err != nil {
		panic(err)
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

	// Telemetry config is baked per-env in config.go (no deploy-time env vars).
	// Empty CollectorLayerArn → StorefrontStack skips OTel (no-op path).
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
		CollectorLayerArn:       cfg.CollectorLayerArn,
		NodeAutoInstrLayerArn:   cfg.NodeAutoInstrLayerArn,
		GrafanaEndpointSSMParam: cfg.GrafanaEndpointSSMParam,
		GrafanaAuthSSMParam:     cfg.GrafanaAuthSSMParam,
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
