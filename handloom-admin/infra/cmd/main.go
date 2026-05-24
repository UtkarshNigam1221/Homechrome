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
		ImageResizer:   storageStack.ImageResizer,
		BaseDomain:     cfg.BaseDomain,
		DomainName:     cfg.DomainName,
		FrontendOrigin: cfg.FrontendOrigin,
		CertArn:        cfg.CertArn,
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
