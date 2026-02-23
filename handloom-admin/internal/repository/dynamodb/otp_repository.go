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

// OTPRepository implements domain.OTPRepository using DynamoDB
type OTPRepository struct {
	client *Client
}

// NewOTPRepository creates a new OTPRepository
func NewOTPRepository(client *Client) *OTPRepository {
	return &OTPRepository{client: client}
}

// Store stores an OTP record with a 5-minute TTL
func (r *OTPRepository) Store(ctx context.Context, otp *domain.OTP) error {
	otp.SetKeys()
	otp.CreatedAt = time.Now()
	otp.TTL = time.Now().Add(5 * time.Minute).Unix()

	av, err := attributevalue.MarshalMap(otp)
	if err != nil {
		return errors.Internal("Failed to marshal OTP")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.sessionsTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to store OTP")
	}

	return nil
}

// Get retrieves an OTP record by phone number
func (r *OTPRepository) Get(ctx context.Context, phone string) (*domain.OTP, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "OTP#" + phone},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get OTP")
	}

	if result.Item == nil {
		return nil, errors.NotFound("OTP")
	}

	var otp domain.OTP
	if err := attributevalue.UnmarshalMap(result.Item, &otp); err != nil {
		return nil, errors.Internal("Failed to unmarshal OTP")
	}

	// Check TTL expiry
	if otp.TTL < time.Now().Unix() {
		return nil, errors.NotFound("OTP")
	}

	return &otp, nil
}

// IncrementAttempts atomically increments the attempts counter for an OTP
func (r *OTPRepository) IncrementAttempts(ctx context.Context, phone string) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "OTP#" + phone},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET attempts = attempts + :one"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to increment OTP attempts")
	}

	return nil
}

// Delete removes an OTP record by phone number
func (r *OTPRepository) Delete(ctx context.Context, phone string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.sessionsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "OTP#" + phone},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to delete OTP")
	}

	return nil
}

// Ensure interface compliance
var _ domain.OTPRepository = (*OTPRepository)(nil)
