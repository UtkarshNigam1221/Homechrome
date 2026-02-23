package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// DatabaseStackProps holds properties for the database stack
type DatabaseStackProps struct {
	awscdk.StackProps
	Environment string
}

// DatabaseStack contains the DynamoDB tables and PostgreSQL catalog database
type DatabaseStack struct {
	awscdk.Stack
	CoreTable          awsdynamodb.Table
	OrdersTable        awsdynamodb.Table
	SessionsTable      awsdynamodb.Table
	AuditTable         awsdynamodb.Table
	AnalyticsTable     awsdynamodb.Table
	NotificationsTable awsdynamodb.Table
	EventsTable        awsdynamodb.Table
	// PostgreSQL catalog database
	CatalogVPC      awsec2.Vpc
	CatalogDB       awsrds.DatabaseInstance
	CatalogDBSecret awsrds.DatabaseSecret
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

	// Events Table - raw frontend tracking events with 30-day TTL
	// Simple PK/SK table, no GSIs needed
	eventsTable := awsdynamodb.NewTable(stack, jsii.String("EventsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-events-" + props.Environment),
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

	// --- PostgreSQL for Catalog Data ---

	// VPC with public subnets only (no NAT Gateway = cheapest)
	catalogVPC := awsec2.NewVpc(stack, jsii.String("CatalogVPC"), &awsec2.VpcProps{
		VpcName:    jsii.String(fmt.Sprintf("handloom-catalog-vpc-%s", props.Environment)),
		MaxAzs:     jsii.Number(2),
		NatGateways: jsii.Number(0),
		SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
			{
				Name:       jsii.String("Public"),
				SubnetType: awsec2.SubnetType_PUBLIC,
				CidrMask:   jsii.Number(24),
			},
		},
	})

	// Security group — allow PostgreSQL from anywhere (Lambdas have dynamic IPs)
	catalogSG := awsec2.NewSecurityGroup(stack, jsii.String("CatalogDBSG"), &awsec2.SecurityGroupProps{
		Vpc:              catalogVPC,
		SecurityGroupName: jsii.String(fmt.Sprintf("handloom-catalog-db-%s", props.Environment)),
		Description:       jsii.String("Allow PostgreSQL access for Lambda functions"),
		AllowAllOutbound:  jsii.Bool(true),
	})
	catalogSG.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(5432)),
		jsii.String("PostgreSQL from Lambda / dev machines"),
		jsii.Bool(false),
	)

	// RDS credentials in Secrets Manager
	catalogDBSecret := awsrds.NewDatabaseSecret(stack, jsii.String("CatalogDBSecret"), &awsrds.DatabaseSecretProps{
		SecretName: jsii.String(fmt.Sprintf("handloom/%s/catalog-db", props.Environment)),
		Username:   jsii.String("handloom"),
	})

	// RDS PostgreSQL db.t4g.micro
	catalogDB := awsrds.NewDatabaseInstance(stack, jsii.String("CatalogDB"), &awsrds.DatabaseInstanceProps{
		InstanceIdentifier: jsii.String(fmt.Sprintf("handloom-catalog-%s", props.Environment)),
		Engine: awsrds.DatabaseInstanceEngine_Postgres(&awsrds.PostgresInstanceEngineProps{
			Version: awsrds.PostgresEngineVersion_VER_16(),
		}),
		InstanceType:              awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE4_GRAVITON, awsec2.InstanceSize_MICRO),
		Vpc:                       catalogVPC,
		VpcSubnets:                &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
		SecurityGroups:            &[]awsec2.ISecurityGroup{catalogSG},
		Credentials:               awsrds.Credentials_FromSecret(catalogDBSecret, nil),
		DatabaseName:              jsii.String("handloom"),
		AllocatedStorage:          jsii.Number(20),
		StorageType:               awsrds.StorageType_GP3,
		MultiAz:                   jsii.Bool(false),
		PubliclyAccessible:        jsii.Bool(true),
		RemovalPolicy:             removalPolicy,
		DeletionProtection:        jsii.Bool(isProd),
		BackupRetention:           awscdk.Duration_Days(jsii.Number(1)),
		MonitoringInterval:        awscdk.Duration_Seconds(jsii.Number(0)),
		EnablePerformanceInsights: jsii.Bool(false),
		StorageEncrypted:          jsii.Bool(true),
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

	awscdk.NewCfnOutput(stack, jsii.String("NotificationsTableName"), &awscdk.CfnOutputProps{
		Value:       notificationsTable.TableName(),
		Description: jsii.String("Notifications DynamoDB table name"),
		ExportName:  jsii.String("handloom-notifications-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("EventsTableName"), &awscdk.CfnOutputProps{
		Value:       eventsTable.TableName(),
		Description: jsii.String("Events DynamoDB table name"),
		ExportName:  jsii.String("handloom-events-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("CatalogDBEndpoint"), &awscdk.CfnOutputProps{
		Value:       catalogDB.DbInstanceEndpointAddress(),
		Description: jsii.String("Catalog PostgreSQL endpoint"),
		ExportName:  jsii.String(fmt.Sprintf("handloom-catalog-db-endpoint-%s", props.Environment)),
	})

	awscdk.NewCfnOutput(stack, jsii.String("CatalogDBSecretARN"), &awscdk.CfnOutputProps{
		Value:       catalogDBSecret.SecretArn(),
		Description: jsii.String("Catalog DB credentials secret ARN"),
		ExportName:  jsii.String(fmt.Sprintf("handloom-catalog-db-secret-%s", props.Environment)),
	})

	return &DatabaseStack{
		Stack:              stack,
		CoreTable:          coreTable,
		OrdersTable:        ordersTable,
		SessionsTable:      sessionsTable,
		AuditTable:         auditTable,
		AnalyticsTable:     analyticsTable,
		NotificationsTable: notificationsTable,
		EventsTable:        eventsTable,
		CatalogVPC:         catalogVPC,
		CatalogDB:          catalogDB,
		CatalogDBSecret:    catalogDBSecret,
	}
}
