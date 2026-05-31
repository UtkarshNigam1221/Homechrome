package dynamodb

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	testCoreTable          = "handloom-core-test"
	testOrdersTable        = "handloom-orders-test"
	testSessionsTable      = "handloom-sessions-test"
	testAuditTable         = "handloom-audit-test"
	testNotificationsTable = "handloom-notifications-test"
)

// testClient creates a DynamoDB client for testing
// Uses DynamoDB Local if DYNAMODB_LOCAL_ENDPOINT is set
func testClient(t *testing.T) *dynamodb.Client {
	endpoint := os.Getenv("DYNAMODB_LOCAL_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return client
}

// setupTestTable creates test tables in DynamoDB Local
func setupTestTable(t *testing.T, client *dynamodb.Client, tableName string) {
	ctx := context.Background()

	// Delete table if it exists
	_, _ = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})

	// Create table
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := client.CreateTable(ctx, input)
	if err != nil {
		t.Logf("Warning: Could not create table %s: %v", tableName, err)
	}
}

// cleanupTestTable deletes the test table
func cleanupTestTable(t *testing.T, client *dynamodb.Client, tableName string) {
	ctx := context.Background()
	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		t.Logf("Warning: Could not delete table %s: %v", tableName, err)
	}
}

// testWrappedClient creates a Client wrapper for testing with test table names
func testWrappedClient(t *testing.T) (*Client, *dynamodb.Client) {
	raw := testClient(t)
	wrapped := &Client{
		db:                 raw,
		coreTable:          testCoreTable,
		ordersTable:        testOrdersTable,
		sessionsTable:      testSessionsTable,
		auditTable:         testAuditTable,
		notificationsTable: testNotificationsTable,
	}
	return wrapped, raw
}

// skipIfNoLocal skips the test if DynamoDB Local is not available
func skipIfNoLocal(t *testing.T, client *dynamodb.Client) {
	ctx := context.Background()
	_, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		t.Skip("DynamoDB Local not available, skipping integration test")
	}
}
