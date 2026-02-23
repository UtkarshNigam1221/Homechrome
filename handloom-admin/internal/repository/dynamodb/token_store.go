package dynamodb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// TokenStore implements domain.TokenStore using DynamoDB
type TokenStore struct {
	client *Client
}

// NewTokenStore creates a new TokenStore
func NewTokenStore(client *Client) *TokenStore {
	return &TokenStore{client: client}
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	UserID     string `dynamodbav:"user_id"`
	TokenHash  string `dynamodbav:"token_hash"`
	EntityType string `dynamodbav:"entity_type"`
	TTL        int64  `dynamodbav:"ttl"`
	CreatedAt  string `dynamodbav:"created_at"`
}

// PasswordResetToken represents a stored password reset token
type PasswordResetToken struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	UserID     string `dynamodbav:"user_id"`
	TokenHash  string `dynamodbav:"token_hash"`
	EntityType string `dynamodbav:"entity_type"`
	TTL        int64  `dynamodbav:"ttl"`
	CreatedAt  string `dynamodbav:"created_at"`
}

// hashToken creates a SHA256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// StoreRefreshToken stores a refresh token for a user
func (s *TokenStore) StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error {
	tokenHash := hashToken(token)
	now := time.Now()
	ttl := now.Add(expiry).Unix()

	item := RefreshToken{
		PK:         "USER#" + userID,
		SK:         "REFRESH_TOKEN#" + tokenHash,
		UserID:     userID,
		TokenHash:  tokenHash,
		EntityType: "REFRESH_TOKEN",
		TTL:        ttl,
		CreatedAt:  now.Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return errors.Internal("Failed to marshal token")
	}

	_, err = s.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to store refresh token")
	}

	return nil
}

// ValidateRefreshToken validates if a refresh token exists and is valid
func (s *TokenStore) ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error) {
	tokenHash := hashToken(token)

	result, err := s.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + userID},
			"SK": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
		},
	})
	if err != nil {
		return false, errors.Wrap(err, "Failed to validate refresh token")
	}

	if result.Item == nil {
		return false, nil
	}

	var storedToken RefreshToken
	if err := attributevalue.UnmarshalMap(result.Item, &storedToken); err != nil {
		return false, errors.Internal("Failed to unmarshal token")
	}

	// Check TTL
	if storedToken.TTL < time.Now().Unix() {
		return false, nil
	}

	return true, nil
}

// RevokeRefreshToken revokes a specific refresh token
func (s *TokenStore) RevokeRefreshToken(ctx context.Context, userID string, token string) error {
	tokenHash := hashToken(token)

	_, err := s.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + userID},
			"SK": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#" + tokenHash},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to revoke refresh token")
	}

	return nil
}

// RevokeAllUserTokens revokes all tokens for a user
func (s *TokenStore) RevokeAllUserTokens(ctx context.Context, userID string) error {
	var exclusiveStartKey map[string]types.AttributeValue
	var allKeys []map[string]types.AttributeValue

	// Paginate through all tokens for user
	for {
		result, err := s.client.db.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(s.client.sessionsTable),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "USER#" + userID},
				":sk": &types.AttributeValueMemberS{Value: "REFRESH_TOKEN#"},
			},
			ProjectionExpression: aws.String("PK, SK"),
			ExclusiveStartKey:    exclusiveStartKey,
		})
		if err != nil {
			return errors.Wrap(err, "Failed to query user tokens")
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
		return errors.Wrap(err, "Failed to batch-delete user tokens")
	}

	return nil
}

// StorePasswordResetToken stores a password reset token
func (s *TokenStore) StorePasswordResetToken(ctx context.Context, userID string, token string, expiry time.Duration) error {
	tokenHash := hashToken(token)
	now := time.Now()
	ttl := now.Add(expiry).Unix()

	item := PasswordResetToken{
		PK:         "PASSWORD_RESET#" + tokenHash,
		SK:         "METADATA",
		UserID:     userID,
		TokenHash:  tokenHash,
		EntityType: "PASSWORD_RESET_TOKEN",
		TTL:        ttl,
		CreatedAt:  now.Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return errors.Internal("Failed to marshal token")
	}

	_, err = s.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to store password reset token")
	}

	return nil
}

// ValidatePasswordResetToken validates a password reset token and returns the user ID
func (s *TokenStore) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	tokenHash := hashToken(token)

	result, err := s.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PASSWORD_RESET#" + tokenHash},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to validate password reset token")
	}

	if result.Item == nil {
		return "", errors.New(errors.ErrCodeInvalidToken, "Invalid or expired reset token")
	}

	var storedToken PasswordResetToken
	if err := attributevalue.UnmarshalMap(result.Item, &storedToken); err != nil {
		return "", errors.Internal("Failed to unmarshal token")
	}

	// Check TTL
	if storedToken.TTL < time.Now().Unix() {
		return "", errors.New(errors.ErrCodeInvalidToken, "Reset token has expired")
	}

	return storedToken.UserID, nil
}

// RevokePasswordResetToken revokes a password reset token
func (s *TokenStore) RevokePasswordResetToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)

	_, err := s.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PASSWORD_RESET#" + tokenHash},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to revoke password reset token")
	}

	return nil
}

// Ensure interface compliance
var _ domain.TokenStore = (*TokenStore)(nil)
