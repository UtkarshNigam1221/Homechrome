package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/triggers"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// DatabaseStackProps holds properties for the database stack
type DatabaseStackProps struct {
	awscdk.StackProps
	Environment string
	PostgresDSN string
	LogsStack   *LogsStack // shared log groups (migrator writes to ApiLogGroup)
}

// DatabaseStack contains the DynamoDB tables and the Postgres DSN for catalog data
type DatabaseStack struct {
	awscdk.Stack
	CoreTable          awsdynamodb.Table
	CouponsTable       awsdynamodb.Table
	OrdersTable        awsdynamodb.Table
	SessionsTable      awsdynamodb.Table
	AuditTable         awsdynamodb.Table
	NotificationsTable awsdynamodb.Table
	// External PostgreSQL (Neon) connection string
	PostgresDSN *string
}

// NewDatabaseStack creates a new database stack
// AWS DynamoDB Free Tier (Always Free):
// - 25 GB of storage
// - 25 provisioned Write Capacity Units (WCU)
// - 25 provisioned Read Capacity Units (RCU)
// OR with on-demand: 25 WRUs and 25 RRUs per month
func NewDatabaseStack(scope constructs.Construct, id string, props *DatabaseStackProps) *DatabaseStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Billing mode - PAY_PER_REQUEST (on-demand) is actually better for free tier
	// when traffic is unpredictable and low. For dev, this stays within free tier.
	billingMode := awsdynamodb.BillingMode_PAY_PER_REQUEST
	removalPolicy := awscdk.RemovalPolicy_DESTROY
	if isProd {
		removalPolicy = awscdk.RemovalPolicy_RETAIN
	}

	// Core Table - categories, designs, products, pricing rules, users, etc.
	// Free tier: 25GB storage across all tables
	coreTable := awsdynamodb.NewTable(stack, jsii.String("CoreTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-core-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // TTL for pricing/coupon expiry
	})

	// GSI1 - General purpose index
	coreTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index
	coreTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Coupons Table — coupons, the code pointer, per-customer redemption counters,
	// redemption history, and bulk batches.
	//
	// No TimeToLiveAttribute, deliberately. The core table sets one "for pricing/coupon
	// expiry"; redemption records are financial history and expired coupons stay visible
	// to the admin, so neither may ever be reaped. Omitting the attribute makes that
	// impossible rather than merely unintended.
	couponsTable := awsdynamodb.NewTable(stack, jsii.String("CouponsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-coupons-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
	})

	couponsTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Notifications Table - notifications with recipient-based GSI
	// Separated for independent scaling during notification spikes
	notificationsTable := awsdynamodb.NewTable(stack, jsii.String("NotificationsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-notifications-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	notificationsTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Sessions Table - OTP, admin refresh tokens, customer refresh tokens
	// Isolated from core table to separate TTL churn from catalog data
	sessionsTable := awsdynamodb.NewTable(stack, jsii.String("SessionsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-sessions-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	// Orders Table - orders and customers
	ordersTable := awsdynamodb.NewTable(stack, jsii.String("OrdersTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-orders-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // TTL for PriceQuote and Cart expiry
	})

	ordersTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index for orders (e.g., customer lookups, price quote expiry)
	ordersTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Audit Table - audit logs with 30-day TTL to limit storage
	auditTable := awsdynamodb.NewTable(stack, jsii.String("AuditTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-audit-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // Use TTL to auto-delete old records and save storage
	})

	auditTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index for audit
	auditTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// --- Schema Migrator Lambda ---
	// Runs embedded SQL migrations against external Postgres (Neon) during CDK deploy.
	postgresDSN := jsii.String(props.PostgresDSN)

	migratorFn := awslambda.NewFunction(stack, jsii.String("MigratorFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("handloom-migrator-%s", props.Environment)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String("../bin/lambda/migrator"), nil),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(128),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(60)),
		LogGroup:     props.LogsStack.ApiLogGroup,
		Environment: &map[string]*string{
			"POSTGRES_DSN": postgresDSN,
			"APP_ENV":      jsii.String(props.Environment),
		},
	})

	triggers.NewTrigger(stack, jsii.String("MigratorTrigger"), &triggers.TriggerProps{
		Handler:                migratorFn,
		ExecuteOnHandlerChange: jsii.Bool(true),
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("CoreTableName"), &awscdk.CfnOutputProps{
		Value:       coreTable.TableName(),
		Description: jsii.String("Core DynamoDB table name"),
		ExportName:  jsii.String("handloom-core-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("CouponsTableName"), &awscdk.CfnOutputProps{
		Value: couponsTable.TableName(),
	})

	awscdk.NewCfnOutput(stack, jsii.String("OrdersTableName"), &awscdk.CfnOutputProps{
		Value:       ordersTable.TableName(),
		Description: jsii.String("Orders DynamoDB table name"),
		ExportName:  jsii.String("handloom-orders-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("SessionsTableName"), &awscdk.CfnOutputProps{
		Value:       sessionsTable.TableName(),
		Description: jsii.String("Sessions DynamoDB table name"),
		ExportName:  jsii.String("handloom-sessions-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("NotificationsTableName"), &awscdk.CfnOutputProps{
		Value:       notificationsTable.TableName(),
		Description: jsii.String("Notifications DynamoDB table name"),
		ExportName:  jsii.String("handloom-notifications-table-" + props.Environment),
	})

	return &DatabaseStack{
		Stack:              stack,
		CoreTable:          coreTable,
		CouponsTable:       couponsTable,
		OrdersTable:        ordersTable,
		SessionsTable:      sessionsTable,
		AuditTable:         auditTable,
		NotificationsTable: notificationsTable,
		PostgresDSN:        postgresDSN,
	}
}
