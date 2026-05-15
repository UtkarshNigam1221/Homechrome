package dynamodb

import (
	"context"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// ReturnRepository persists customer return requests, colocated with their order.
type ReturnRepository struct {
	client *Client
}

// NewReturnRepository constructs a repository.
func NewReturnRepository(client *Client) *ReturnRepository {
	return &ReturnRepository{client: client}
}

// Create writes a new return request keyed under ORDER#<order_id> / RETURN#<id>.
func (r *ReturnRepository) Create(ctx context.Context, rr *domain.ReturnRequest) error {
	now := time.Now().UTC()
	rr.CreatedAt = now
	rr.UpdatedAt = now
	rr.SetKeys()
	av, err := attributevalue.MarshalMap(rr)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal return request")
	}
	_, err = r.client.db.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Return request already exists")
		}
		return errors.Wrap(err, "Failed to create return request")
	}
	return nil
}

// GetByID retrieves a return request by (order_id, return_id).
func (r *ReturnRepository) GetByID(ctx context.Context, orderID, returnID string) (*domain.ReturnRequest, error) {
	out, err := r.client.db.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			"SK": &types.AttributeValueMemberS{Value: "RETURN#" + returnID},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get return request")
	}
	if out.Item == nil {
		return nil, errors.NotFound("Return request not found")
	}
	var rr domain.ReturnRequest
	if err := attributevalue.UnmarshalMap(out.Item, &rr); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal return request")
	}
	return &rr, nil
}

// UpdateStatus transitions a return to a new status and applies arbitrary attribute updates.
func (r *ReturnRepository) UpdateStatus(ctx context.Context, orderID, returnID string, status domain.ReturnStatus, updates map[string]interface{}) error {
	du, err := buildDynamicUpdate(string(status), updates)
	if err != nil {
		return err
	}

	_, err = r.client.db.UpdateItem(ctx, &awsdynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			"SK": &types.AttributeValueMemberS{Value: "RETURN#" + returnID},
		},
		UpdateExpression:          aws.String(du.Expression),
		ExpressionAttributeNames:  du.AttrNames,
		ExpressionAttributeValues: du.AttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Return request not found")
		}
		return errors.Wrap(err, "Failed to update return status")
	}
	return nil
}

// ListByOrder returns every return colocated under a given order partition.
func (r *ReturnRepository) ListByOrder(ctx context.Context, orderID string) ([]*domain.ReturnRequest, error) {
	out, err := r.client.db.Query(ctx, &awsdynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			exprSK: &types.AttributeValueMemberS{Value: "RETURN#"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list returns")
	}
	returns := make([]*domain.ReturnRequest, 0, len(out.Items))
	for _, item := range out.Items {
		var rr domain.ReturnRequest
		if err := attributevalue.UnmarshalMap(item, &rr); err != nil {
			continue
		}
		returns = append(returns, &rr)
	}
	return returns, nil
}

// ListByStatus queries returns globally by status via the entity-status-index GSI.
// Uses base64-encoded ExclusiveStartKey cursors (see pagination.go).
func (r *ReturnRepository) ListByStatus(ctx context.Context, status domain.ReturnStatus, limit int, cursor string) ([]*domain.ReturnRequest, string, error) {
	if limit < 0 {
		limit = 0
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	input := &awsdynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("entity-status-index"),
		KeyConditionExpression: aws.String("entity_type = :et AND #s = :s"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":et": &types.AttributeValueMemberS{Value: domain.EntityTypeReturnRequest},
			":s":  &types.AttributeValueMemberS{Value: string(status)},
		},
		Limit: aws.Int32(int32(limit)), //nolint:gosec // bounded above
	}
	if cursor != "" {
		startKey, err := DecodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		input.ExclusiveStartKey = startKey
	}
	out, err := r.client.db.Query(ctx, input)
	if err != nil {
		return nil, "", errors.Wrap(err, "Failed to list returns by status")
	}
	returns := make([]*domain.ReturnRequest, 0, len(out.Items))
	for _, item := range out.Items {
		var rr domain.ReturnRequest
		if err := attributevalue.UnmarshalMap(item, &rr); err != nil {
			continue
		}
		returns = append(returns, &rr)
	}
	return returns, EncodeCursor(out.LastEvaluatedKey), nil
}

// Compile-time interface assertion.
var _ domain.ReturnRepository = (*ReturnRepository)(nil)
