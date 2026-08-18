package dynamodb

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CustomerTokenStore implements domain.CustomerTokenStore using DynamoDB
type CustomerTokenStore struct {
	client *Client
}

// NewCustomerTokenStore creates a new CustomerTokenStore
func NewCustomerTokenStore(client *Client) *CustomerTokenStore {
	return &CustomerTokenStore{client: client}
}

// StoreToken stores a customer refresh token hash with TTL
func (s *CustomerTokenStore) StoreToken(ctx context.Context, customerID, tokenHash string, ttl int64) error {
	now := time.Now()

	_, err := s.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Item: map[string]types.AttributeValue{
			"PK":          &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
			"SK":          &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
			"customer_id": &types.AttributeValueMemberS{Value: customerID},
			"token_hash":  &types.AttributeValueMemberS{Value: tokenHash},
			"entity_type": &types.AttributeValueMemberS{Value: "CUSTOMER_REFRESH_TOKEN"},
			attrTTL:       &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
			"created_at":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to store customer token")
	}

	return nil
}

// ValidateToken checks if a customer refresh token exists and is not expired
func (s *CustomerTokenStore) ValidateToken(ctx context.Context, customerID, tokenHash string) (bool, error) {
	result, err := s.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
		},
	})
	if err != nil {
		return false, errors.Wrap(err, "Failed to validate customer token")
	}

	if result.Item == nil {
		return false, nil
	}

	// Check TTL
	if ttlAttr, ok := result.Item[attrTTL].(*types.AttributeValueMemberN); ok {
		ttl, _ := strconv.ParseInt(ttlAttr.Value, 10, 64)
		if ttl < time.Now().Unix() {
			return false, nil
		}
	}

	return true, nil
}

// ClaimRotation implements domain.CustomerTokenStore. The conditional update is
// what serializes concurrent refreshes: DynamoDB evaluates the condition and
// applies the write as one atomic operation per item, so the second refresh to
// arrive finds successor_hash already set and loses the claim.
func (s *CustomerTokenStore) ClaimRotation(ctx context.Context, customerID, tokenHash, successorHash string, graceTTL int64) (bool, error) {
	_, err := s.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
		},
		UpdateExpression:    aws.String("SET #successor = :successor, #ttl = :ttl"),
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_not_exists(#successor)"),
		ExpressionAttributeNames: map[string]string{
			"#successor": attrSuccessorHash,
			"#ttl":       attrTTL,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":successor": &types.AttributeValueMemberS{Value: successorHash},
			":ttl":       &types.AttributeValueMemberN{Value: strconv.FormatInt(graceTTL, 10)},
		},
		// Returning the offending item is what separates the two ways the
		// condition can fail: a row that is present was already rotated, a row
		// that is absent was revoked.
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	})
	if err != nil {
		var conditionFailed *types.ConditionalCheckFailedException
		if stderrors.As(err, &conditionFailed) {
			if conditionFailed.Item == nil {
				return false, errors.New(errors.ErrCodeInvalidToken, "Refresh token has been revoked")
			}
			return false, nil
		}
		return false, errors.Wrap(err, "Failed to claim customer token rotation")
	}

	return true, nil
}

// RevokeToken revokes a specific customer refresh token
func (s *CustomerTokenStore) RevokeToken(ctx context.Context, customerID, tokenHash string) error {
	_, err := s.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to revoke customer token")
	}

	return nil
}

// queryTokenRows pages through every refresh-token row for a customer,
// projecting PK, SK, and the ttl attribute.
func (s *CustomerTokenStore) queryTokenRows(ctx context.Context, customerID string) ([]map[string]types.AttributeValue, error) {
	var exclusiveStartKey map[string]types.AttributeValue
	var items []map[string]types.AttributeValue

	for {
		result, err := s.client.db.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(s.client.sessionsTable),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				exprPK: &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
				exprSK: &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#"},
			},
			ExpressionAttributeNames: map[string]string{"#ttl": attrTTL},
			ProjectionExpression:     aws.String("PK, SK, #ttl"),
			ExclusiveStartKey:        exclusiveStartKey,
		})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to query customer tokens")
		}

		items = append(items, result.Items...)

		if result.LastEvaluatedKey == nil {
			break
		}
		exclusiveStartKey = result.LastEvaluatedKey
	}

	return items, nil
}

// batchDeleteItems deletes each item by its PK/SK, ignoring any other
// projected attributes.
func (s *CustomerTokenStore) batchDeleteItems(ctx context.Context, items []map[string]types.AttributeValue) error {
	if len(items) == 0 {
		return nil
	}

	keys := make([]map[string]types.AttributeValue, 0, len(items))
	for _, item := range items {
		keys = append(keys, map[string]types.AttributeValue{
			"PK": item["PK"],
			"SK": item["SK"],
		})
	}

	if err := batchDeleteKeys(ctx, s.client.db, s.client.sessionsTable, keys); err != nil {
		return errors.Wrap(err, "Failed to batch-delete customer tokens")
	}

	return nil
}

// RevokeAllTokens revokes all refresh tokens for a customer
func (s *CustomerTokenStore) RevokeAllTokens(ctx context.Context, customerID string) error {
	items, err := s.queryTokenRows(ctx, customerID)
	if err != nil {
		return err
	}

	return s.batchDeleteItems(ctx, items)
}

// RevokeTokensExpiringBefore deletes every one of the customer's tokens whose
// TTL is at or before cutoff. See the domain.CustomerTokenStore doc comment
// for why a near-future cutoff only ever catches rotation's grace-window
// predecessors, never another device's live session.
func (s *CustomerTokenStore) RevokeTokensExpiringBefore(ctx context.Context, customerID string, cutoff int64) error {
	items, err := s.queryTokenRows(ctx, customerID)
	if err != nil {
		return err
	}

	var expiring []map[string]types.AttributeValue
	for _, item := range items {
		ttlAttr, ok := item[attrTTL].(*types.AttributeValueMemberN)
		if !ok {
			continue
		}
		ttl, err := strconv.ParseInt(ttlAttr.Value, 10, 64)
		if err != nil || ttl > cutoff {
			continue
		}
		expiring = append(expiring, item)
	}

	return s.batchDeleteItems(ctx, expiring)
}

// Ensure interface compliance
var _ domain.CustomerTokenStore = (*CustomerTokenStore)(nil)
