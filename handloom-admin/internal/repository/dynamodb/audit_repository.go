package dynamodb

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
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
	// Default date range: last 7 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	if req.StartDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return nil, errors.BadRequest("Invalid start_date format, expected YYYY-MM-DD")
		}
		startDate = parsed
	}
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, errors.BadRequest("Invalid end_date format, expected YYYY-MM-DD")
		}
		endDate = parsed
	}

	var allLogs []*domain.AuditLog

	// Query each day partition
	for d := endDate; !d.Before(startDate); d = d.AddDate(0, 0, -1) {
		dateStr := d.Format("2006-01-02")
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.client.auditTable),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "AUDIT#" + dateStr},
			},
			ScanIndexForward: aws.Bool(false),
		}

		// Build filter expressions
		var filters []string
		exprAttrNames := map[string]string{}

		if req.Action != nil {
			filters = append(filters, "#action = :action")
			input.ExpressionAttributeValues[":action"] = &types.AttributeValueMemberS{Value: string(*req.Action)}
			exprAttrNames["#action"] = "action"
		}
		if req.EntityType != nil {
			filters = append(filters, "entity_type_audit = :entityType")
			input.ExpressionAttributeValues[":entityType"] = &types.AttributeValueMemberS{Value: *req.EntityType}
		}
		if req.EntityID != nil {
			filters = append(filters, "entity_id = :entityID")
			input.ExpressionAttributeValues[":entityID"] = &types.AttributeValueMemberS{Value: *req.EntityID}
		}
		if req.UserID != nil {
			filters = append(filters, "user_id = :userID")
			input.ExpressionAttributeValues[":userID"] = &types.AttributeValueMemberS{Value: *req.UserID}
		}

		if len(filters) > 0 {
			input.FilterExpression = aws.String(strings.Join(filters, " AND "))
		}
		if len(exprAttrNames) > 0 {
			input.ExpressionAttributeNames = exprAttrNames
		}

		result, err := r.client.db.Query(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to query audit logs for date "+dateStr)
		}

		var logs []*domain.AuditLog
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &logs); err != nil {
			return nil, errors.Internal("Failed to unmarshal audit logs")
		}

		allLogs = append(allLogs, logs...)
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(allLogs, req.PaginationRequest)

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
