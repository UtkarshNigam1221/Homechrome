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
	var eventStack *stacks.EventStack // DISABLED: assign from NewEventStack(...) when re-enabling.

	// CronStack is constructed before APIStack so APIStack can use
	// cronStack.RateRefreshFn.GrantInvoke() and FunctionName() — avoids the
	// cross-stack circular dep that existed when APIStack exposed
	// GatewayEnvVars to CronStack.
	cronEnv := buildCronEnv(environment, databaseStack, eventStack)
	cronStack := stacks.NewCronStack(app, "HandloomCronStack-"+environment, &stacks.CronStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin - Scheduled cron Lambdas (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("handloom-admin"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment:   environment,
		OrdersTable:   databaseStack.OrdersTable,
		ShippingTable: databaseStack.ShippingTable,
		CoreTable:     databaseStack.CoreTable,
		EventTopic:    nil, // wire when event stack is enabled
		EnvVars:       cronEnv,
	})

	// API stack (depends on database, storage, events, and cron rate-refresh fn)
	apiStack := stacks.NewAPIStack(app, "HandloomAPIStack-"+environment, &stacks.APIStackProps{
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
		EventStack:     eventStack, // DISABLED: nil unless EventStack is re-enabled above
		BaseDomain:     cfg.BaseDomain,
		DomainName:     cfg.DomainName,
		FrontendOrigin: cfg.FrontendOrigin,
		CertArn:        cfg.CertArn,
		RateRefreshFn:  cronStack.RateRefreshFn,
	})
	_ = apiStack

	app.Synth(nil)
}

// buildCronEnv constructs the slim environment variable map handed to the cron
// Lambdas. Cron Lambdas don't share most of APIStack's commonEnv (no JWT
// secrets, no MSG91, no PhonePe, no ALLOWED_ORIGINS/COOKIE_DOMAIN), so
// building this independently breaks the previous APIStack <-> CronStack
// circular dependency.
func buildCronEnv(environment string, db *stacks.DatabaseStack, eventStack *stacks.EventStack) *map[string]*string {
	cronEnv := map[string]*string{
		"APP_ENV":                 jsii.String(environment),
		"DYNAMODB_CORE_TABLE":     db.CoreTable.TableName(),
		"DYNAMODB_ORDERS_TABLE":   db.OrdersTable.TableName(),
		"DYNAMODB_SHIPPING_TABLE": db.ShippingTable.TableName(),
		"POSTGRES_DSN":            db.PostgresDSN,
	}

	// Propagate Delhivery + shipping behavior knobs from the deploy shell
	// environment. Empty values fall through to the gateway's DevClient.
	cronEnvKeys := []string{
		"DELHIVERY_API_TOKEN",
		"DELHIVERY_BASE_URL",
		"DELHIVERY_CLIENT_NAME",
		"DELHIVERY_WEBHOOK_TOKEN",
		"DELHIVERY_PICKUP_LOCATION",
		"NDR_REATTEMPT_LIMIT",
		"RETURN_WINDOW_DAYS",
	}
	for _, key := range cronEnvKeys {
		if v := os.Getenv(key); v != "" {
			cronEnv[key] = jsii.String(v)
		}
	}

	if eventStack != nil {
		cronEnv["SNS_TOPIC_ARN"] = eventStack.TopicARN
		cronEnv["EVENT_PUBLISHING_ENABLED"] = jsii.String("true")
	}

	return &cronEnv
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
