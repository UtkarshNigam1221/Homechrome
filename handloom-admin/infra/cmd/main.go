package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	"github.com/handloom/admin/infra/stacks"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// Get deployment mode: "single" deploys one env, "all" deploys both
	deployMode := getDeployMode(app)

	// Get AWS environment configuration
	env := getAWSEnv()

	if deployMode == "all" {
		// Deploy both dev and prod environments
		createEnvironmentStacks(app, "dev", env)
		createEnvironmentStacks(app, "prod", env)
	} else {
		// Deploy single environment (default: dev)
		environment := getEnvironment(app)
		createEnvironmentStacks(app, environment, env)
	}

	app.Synth(nil)
}

// createEnvironmentStacks creates all stacks for a given environment
func createEnvironmentStacks(app awscdk.App, environment string, env *awscdk.Environment) {
	postgresDSN := getPostgresDSN(app)

	// Database stack
	databaseStack := stacks.NewDatabaseStack(app, "HandloomDatabaseStack-"+environment, &stacks.DatabaseStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Database resources (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("handloom-admin"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment: environment,
		PostgresDSN: postgresDSN,
	})

	// Storage stack
	storageStack := stacks.NewStorageStack(app, "HandloomStorageStack-"+environment, &stacks.StorageStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Storage resources (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("handloom-admin"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment: environment,
	})

	// DISABLED: Event stack (SNS + SQS + 4 worker Lambdas + EventBridge rule)
	// Uncomment below to re-enable event-driven workers.
	// eventStack := stacks.NewEventStack(app, "HandloomEventStack-"+environment, &stacks.EventStackProps{
	// 	StackProps: awscdk.StackProps{
	// 		Env:         env,
	// 		Description: jsii.String("Handloom Admin - Event-driven async infrastructure (" + environment + ")"),
	// 		Tags: &map[string]*string{
	// 			"Environment": jsii.String(environment),
	// 			"Project":     jsii.String("handloom-admin"),
	// 			"ManagedBy":   jsii.String("cdk"),
	// 		},
	// 	},
	// 	Environment:   environment,
	// 	DatabaseStack: databaseStack,
	// })

	// Compute custom domain config from CDK context
	certArn := getCertArn(app)
	baseDomain := getBaseDomain(app)
	var domainName, frontendOrigin string
	if certArn != "" {
		switch environment {
		case "prod":
			domainName = "api." + baseDomain
			frontendOrigin = "https://admin." + baseDomain
		default:
			domainName = "dev-api." + baseDomain
			frontendOrigin = "https://dev-admin." + baseDomain
		}
	}

	// API stack (depends on database, storage, and events)
	stacks.NewAPIStack(app, "HandloomAPIStack-"+environment, &stacks.APIStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - API and Lambda functions (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("handloom-admin"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment:    environment,
		DatabaseStack:  databaseStack,
		StorageStack:   storageStack,
		EventStack:     nil, // DISABLED: pass eventStack here when re-enabling
		BaseDomain:     baseDomain,
		DomainName:     domainName,
		FrontendOrigin: frontendOrigin,
		CertArn:        certArn,
	})
}

func getDeployMode(app constructs.Construct) string {
	// Try to get from CDK context
	if mode := app.Node().TryGetContext(jsii.String("deployMode")); mode != nil {
		return mode.(string)
	}

	// Try to get from environment variable
	if mode := os.Getenv("CDK_DEPLOY_MODE"); mode != "" {
		return mode
	}

	// Default to single environment deployment
	return "single"
}

func getEnvironment(app constructs.Construct) string {
	// Try to get from CDK context
	if env := app.Node().TryGetContext(jsii.String("environment")); env != nil {
		return env.(string)
	}

	// Try to get from environment variable
	if env := os.Getenv("CDK_ENVIRONMENT"); env != "" {
		return env
	}

	// Default to dev
	return "dev"
}

func getCertArn(app constructs.Construct) string {
	if arn := app.Node().TryGetContext(jsii.String("certArn")); arn != nil {
		return arn.(string)
	}
	if arn := os.Getenv("ACM_CERT_ARN"); arn != "" {
		return arn
	}
	return ""
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

func getBaseDomain(app constructs.Construct) string {
	if d := app.Node().TryGetContext(jsii.String("baseDomain")); d != nil {
		return d.(string)
	}
	if d := os.Getenv("BASE_DOMAIN"); d != "" {
		return d
	}
	return "homechrome.in"
}

func getAWSEnv() *awscdk.Environment {
	// Try to get from environment variables
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
