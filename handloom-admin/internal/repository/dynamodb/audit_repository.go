package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// AuditRepository implements audit log operations
type AuditRepository struct {
	client *Client
}

// NewAuditRepository creates a new AuditRepository
func NewAuditRepository(client *Client) *AuditRepository {
	return &AuditRepository{client: client}
}

// Create creates a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	now := time.Now()
	log.CreatedAt = now
	log.SetKeys()

	av, err := attributevalue.MarshalMap(log)
	if err != nil {
		return errors.Internal("Failed to marshal audit log")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.auditTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create audit log")
	}

	return nil
}

// GetByID retrieves an audit log by ID
func (r *AuditRepository) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	// Since we need both PK and SK, we need to query
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.auditTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "AUDIT#" + id},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get audit log")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("Audit log not found")
	}

	var log domain.AuditLog
	if err := attributevalue.UnmarshalMap(result.Items[0], &log); err != nil {
		return nil, errors.Internal("Failed to unmarshal audit log")
	}

	return &log, nil
}

// List retrieves audit logs with filters
func (r *AuditRepository) List(ctx context.Context, req domain.ListAuditLogsRequest) (*domain.ListAuditLogsResponse, error) {
	filterBuilder := expression.Name("entity_type").Equal(expression.Value("AUDIT_LOG"))

	if req.Action != nil {
		filterBuilder = filterBuilder.And(expression.Name("action").Equal(expression.Value(*req.Action)))
	}

	if req.EntityType != nil {
		filterBuilder = filterBuilder.And(expression.Name("target_entity_type").Equal(expression.Value(*req.EntityType)))
	}

	if req.EntityID != nil {
		filterBuilder = filterBuilder.And(expression.Name("target_entity_id").Equal(expression.Value(*req.EntityID)))
	}

	if req.UserID != nil {
		filterBuilder = filterBuilder.And(expression.Name("user_id").Equal(expression.Value(*req.UserID)))
	}

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	result, err := r.client.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.auditTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list audit logs")
	}

	var logs []*domain.AuditLog
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &logs); err != nil {
		return nil, errors.Internal("Failed to unmarshal audit logs")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(logs, req.PaginationRequest)

	return &domain.ListAuditLogsResponse{
		Logs:       paged,
		Pagination: pg,
	}, nil
}

// GetByEntity retrieves audit logs for a specific entity
func (r *AuditRepository) GetByEntity(ctx context.Context, entityType string, entityID string, pagination domain.PaginationRequest) (*domain.ListAuditLogsResponse, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.auditTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: entityType + "#" + entityID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query audit logs by entity")
	}

	var logs []*domain.AuditLog
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &logs); err != nil {
		return nil, errors.Internal("Failed to unmarshal audit logs")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(logs, pagination)

	return &domain.ListAuditLogsResponse{
		Logs:       paged,
		Pagination: pg,
	}, nil
}

// GetByUser retrieves audit logs for a specific user
func (r *AuditRepository) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListAuditLogsResponse, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.auditTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER#" + userID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query audit logs by user")
	}

	var logs []*domain.AuditLog
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &logs); err != nil {
		return nil, errors.Internal("Failed to unmarshal audit logs")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(logs, pagination)

	return &domain.ListAuditLogsResponse{
		Logs:       paged,
		Pagination: pg,
	}, nil
}
