// Package dynamodb provides DynamoDB repository implementations
package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// RefundRepository implements domain.RefundRepository using DynamoDB.
// Refunds live in the orders table beside payments.
type RefundRepository struct {
	client *Client
}

// NewRefundRepository creates a new RefundRepository.
func NewRefundRepository(client *Client) *RefundRepository {
	return &RefundRepository{client: client}
}

// Create persists a refund. Called before the provider is, so a refund can
// never leave the building without a local record to reconcile it against.
func (r *RefundRepository) Create(ctx context.Context, refund *domain.Refund) error {
	refund.SetKeys()

	av, err := attributevalue.MarshalMap(refund)
	if err != nil {
		return errors.Internal("Failed to marshal refund")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Refund already exists")
		}
		return errors.Wrap(err, "Failed to create refund")
	}

	return nil
}

// GetByID retrieves a refund by ID.
func (r *RefundRepository) GetByID(ctx context.Context, id string) (*domain.Refund, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get refund")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Refund not found")
	}

	var refund domain.Refund
	if err := attributevalue.UnmarshalMap(result.Item, &refund); err != nil {
		return nil, errors.Internal("Failed to unmarshal refund")
	}

	return &refund, nil
}

// ListByOrder returns an order's refunds, oldest first. GSI1's ORDER# partition holds
// payments too, so the sort-key prefix does the narrowing.
func (r *RefundRepository) ListByOrder(ctx context.Context, orderID string) ([]*domain.Refund, error) {
	return QueryAll[domain.Refund](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:     &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			exprPrefix: &types.AttributeValueMemberS{Value: "REFUND#"},
		},
	}, "Failed to list refunds for order")
}

// GetByProviderRefundID finds the refund a webhook is about. Webhooks carry only
// PhonePe's id, and an order can have several refunds.
func (r *RefundRepository) GetByProviderRefundID(ctx context.Context, providerRefundID string) (*domain.Refund, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "REFUND_PROVIDER#" + providerRefundID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to look up refund by provider id")
	}
	if len(result.Items) == 0 {
		return nil, errors.NotFound("Refund not found")
	}

	var refund domain.Refund
	if err := attributevalue.UnmarshalMap(result.Items[0], &refund); err != nil {
		return nil, errors.Internal("Failed to unmarshal refund")
	}

	return &refund, nil
}

// SetProviderRefundID records PhonePe's id, which also puts the refund on GSI2
// where the webhook can find it.
func (r *RefundRepository) SetProviderRefundID(ctx context.Context, id, providerRefundID string) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("SET provider_refund_id = :pid, GSI2PK = :gsi2pk, GSI2SK = :gsi2sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pid":     &types.AttributeValueMemberS{Value: providerRefundID},
			":gsi2pk":  &types.AttributeValueMemberS{Value: "REFUND_PROVIDER#" + providerRefundID},
			":gsi2sk":  &types.AttributeValueMemberS{Value: skMetadata},
			":pending": &types.AttributeValueMemberS{Value: string(domain.RefundStatusPending)},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND #status = :pending"),
		ExpressionAttributeNames: map[string]string{
			nameStatus: attrStatus,
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			// Already settled — a status re-check beat the initiation response
			// home. The id it recorded is the same one.
			return nil
		}
		return errors.Wrap(err, "Failed to record provider refund id")
	}
	return nil
}

// Settle moves a refund to a terminal state, only from PENDING. That condition is the
// gate: of two concurrent deliveries one wins, and only it runs the effects.
func (r *RefundRepository) Settle(ctx context.Context, id string, status domain.RefundStatus, completedAt time.Time, errorCode, detailedErrorCode string) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String(
			"SET #status = :status, completed_at = :completedAt, error_code = :errorCode, detailed_error_code = :detailedErrorCode"),
		ConditionExpression: aws.String("attribute_exists(PK) AND #status = :pending"),
		ExpressionAttributeNames: map[string]string{
			nameStatus: attrStatus,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			valStatus:            &types.AttributeValueMemberS{Value: string(status)},
			":completedAt":       &types.AttributeValueMemberS{Value: completedAt.Format(time.RFC3339)},
			":errorCode":         &types.AttributeValueMemberS{Value: errorCode},
			":detailedErrorCode": &types.AttributeValueMemberS{Value: detailedErrorCode},
			":pending":           &types.AttributeValueMemberS{Value: string(domain.RefundStatusPending)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeConflict, "Refund is no longer pending")
		}
		return errors.Wrap(err, "Failed to settle refund")
	}
	return nil
}

var _ domain.RefundRepository = (*RefundRepository)(nil)
