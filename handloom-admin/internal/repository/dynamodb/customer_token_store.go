package dynamodb

import (
	"context"
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
			"ttl":         &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
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
	if ttlAttr, ok := result.Item["ttl"].(*types.AttributeValueMemberN); ok {
		ttl, _ := strconv.ParseInt(ttlAttr.Value, 10, 64)
		if ttl < time.Now().Unix() {
			return false, nil
		}
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

// RevokeAllTokens revokes all refresh tokens for a customer
func (s *CustomerTokenStore) RevokeAllTokens(ctx context.Context, customerID string) error {
	var exclusiveStartKey map[string]types.AttributeValue
	var allKeys []map[string]types.AttributeValue

	// Paginate through all tokens for customer
	for {
		result, err := s.client.db.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(s.client.sessionsTable),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "CUST_TOKEN#" + customerID},
				":sk": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#"},
			},
			ProjectionExpression: aws.String("PK, SK"),
			ExclusiveStartKey:    exclusiveStartKey,
		})
		if err != nil {
			return errors.Wrap(err, "Failed to query customer tokens")
		}

		for _, item := range result.Items {
			allKeys = append(allKeys, map[string]types.AttributeValue{
				"PK": item["PK"],
				"SK": item["SK"],
			})
		}

		if result.LastEvaluatedKey == nil {
			break
		}
		exclusiveStartKey = result.LastEvaluatedKey
	}

	if len(allKeys) == 0 {
		return nil
	}

	if err := batchDeleteKeys(ctx, s.client.db, s.client.sessionsTable, allKeys); err != nil {
		return errors.Wrap(err, "Failed to batch-delete customer tokens")
	}

	return nil
}

// Ensure interface compliance
var _ domain.CustomerTokenStore = (*CustomerTokenStore)(nil)
