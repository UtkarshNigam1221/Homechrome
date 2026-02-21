package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// DatabaseStackProps holds properties for the database stack
type DatabaseStackProps struct {
	awscdk.StackProps
	Environment string
}

// DatabaseStack contains the DynamoDB tables
type DatabaseStack struct {
	awscdk.Stack
	CoreTable      awsdynamodb.Table
	OrdersTable    awsdynamodb.Table
	SessionsTable  awsdynamodb.Table
	AuditTable     awsdynamodb.Table
	AnalyticsTable awsdynamodb.Table
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
		PointInTimeRecovery:  jsii.Bool(false), // Disable PITR in dev to save costs (not free)
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
		PointInTimeRecovery:  jsii.Bool(false),
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
		PointInTimeRecovery:  jsii.Bool(false), // Disable PITR in dev to save costs
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
		PointInTimeRecovery: jsii.Bool(false), // Disable PITR to save costs
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

	// Analytics Table - analytics data
	analyticsTable := awsdynamodb.NewTable(stack, jsii.String("AnalyticsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-analytics-" + props.Environment),
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
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	// GSI1 - General purpose index for analytics
	analyticsTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
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

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("CoreTableName"), &awscdk.CfnOutputProps{
		Value:       coreTable.TableName(),
		Description: jsii.String("Core DynamoDB table name"),
		ExportName:  jsii.String("handloom-core-table-" + props.Environment),
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

	return &DatabaseStack{
		Stack:          stack,
		CoreTable:      coreTable,
		OrdersTable:    ordersTable,
		SessionsTable:  sessionsTable,
		AuditTable:     auditTable,
		AnalyticsTable: analyticsTable,
	}
}
