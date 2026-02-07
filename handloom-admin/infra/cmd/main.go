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

	// API stack (depends on database and storage)
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
		Environment:   environment,
		DatabaseStack: databaseStack,
		StorageStack:  storageStack,
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

	// Return nil to use default credentials
	if account == "" || region == "" {
		return nil
	}

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}
