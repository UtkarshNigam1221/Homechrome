// Package dynamodb provides DynamoDB repository implementations
package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	appconfig "github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/pkg/metrics/awsmiddleware"
)

// Client wraps DynamoDB client with table names
type Client struct {
	db                 *dynamodb.Client
	coreTable          string
	ordersTable        string
	sessionsTable      string
	auditTable         string
	notificationsTable string
}

// NewClient creates a new DynamoDB client
func NewClient(ctx context.Context, cfg *appconfig.Config) (*Client, error) {
	var awsCfg aws.Config
	var err error

	if cfg.IsLocal() {
		// Local DynamoDB configuration
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(cfg.AWS.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				"local",
				"local",
				"",
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	} else {
		// Production AWS configuration
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(cfg.AWS.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)
	svcName := os.Getenv("OTEL_SERVICE_NAME")
	if svcName == "" {
		svcName = "handloom-lambda"
	}
	awsCfg.APIOptions = append(awsCfg.APIOptions, awsmiddleware.With(svcName))

	var client *dynamodb.Client
	if cfg.IsLocal() {
		client = dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	} else {
		client = dynamodb.NewFromConfig(awsCfg)
	}

	return &Client{
		db:                 client,
		coreTable:          cfg.DynamoDB.CoreTable,
		ordersTable:        cfg.DynamoDB.OrdersTable,
		sessionsTable:      cfg.DynamoDB.SessionsTable,
		auditTable:         cfg.DynamoDB.AuditTable,
		notificationsTable: cfg.DynamoDB.NotificationsTable,
	}, nil
}

// DB returns the underlying DynamoDB client
func (c *Client) DB() *dynamodb.Client {
	return c.db
}

// CoreTable returns the core table name
func (c *Client) CoreTable() string {
	return c.coreTable
}

// OrdersTable returns the orders table name
func (c *Client) OrdersTable() string {
	return c.ordersTable
}

// AuditTable returns the audit table name
func (c *Client) AuditTable() string {
	return c.auditTable
}

// SessionsTable returns the sessions table name
func (c *Client) SessionsTable() string {
	return c.sessionsTable
}

// NotificationsTable returns the notifications table name
func (c *Client) NotificationsTable() string {
	return c.notificationsTable
}

// isConditionalCheckFailed checks if an error is a DynamoDB conditional check failure
func isConditionalCheckFailed(err error) bool {
	var ccfe *types.ConditionalCheckFailedException
	return errors.As(err, &ccfe)
}

// isTransactionCanceled checks if an error is a DynamoDB transaction canceled exception
func isTransactionCanceled(err error) bool {
	var tce *types.TransactionCanceledException
	return errors.As(err, &tce)
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sr, tr := s[i+j], substr[j]
			if sr != tr && toLowerByte(sr) != toLowerByte(tr) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLowerByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
