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

// CouponRepository implements domain.CouponRepository
type CouponRepository struct {
	client *Client
}

// NewCouponRepository creates a new CouponRepository
func NewCouponRepository(client *Client) *CouponRepository {
	return &CouponRepository{
		client: client,
	}
}

// Create creates a new coupon
func (r *CouponRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	now := time.Now()
	coupon.CreatedAt = now
	coupon.UpdatedAt = now
	coupon.SetKeys()

	av, err := attributevalue.MarshalMap(coupon)
	if err != nil {
		return errors.Internal("Failed to marshal coupon")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Coupon already exists")
		}
		return errors.Wrap(err, "Failed to create coupon")
	}

	return nil
}

// GetByID retrieves a coupon by ID
func (r *CouponRepository) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUPON#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get coupon")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Coupon")
	}

	var coupon domain.Coupon
	if err := attributevalue.UnmarshalMap(result.Item, &coupon); err != nil {
		return nil, errors.Internal("Failed to unmarshal coupon")
	}

	return &coupon, nil
}

// GetByCode retrieves a coupon by code
func (r *CouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	// TODO: Implement with GSI query on code
	return nil, nil
}

// Update updates a coupon
func (r *CouponRepository) Update(ctx context.Context, coupon *domain.Coupon) error {
	coupon.UpdatedAt = time.Now()
	coupon.SetKeys()

	av, err := attributevalue.MarshalMap(coupon)
	if err != nil {
		return errors.Internal("Failed to marshal coupon")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Coupon")
		}
		return errors.Wrap(err, "Failed to update coupon")
	}

	return nil
}

// Delete deletes a coupon
func (r *CouponRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUPON#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Coupon")
		}
		return errors.Wrap(err, "Failed to delete coupon")
	}

	return nil
}

// List lists coupons with filters
func (r *CouponRepository) List(ctx context.Context, req domain.ListCouponsRequest) (*domain.ListCouponsResponse, error) {
	// TODO: Implement with DynamoDB scan/query
	return &domain.ListCouponsResponse{
		Coupons:    []*domain.Coupon{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// RecordUsage records coupon usage
func (r *CouponRepository) RecordUsage(ctx context.Context, usage *domain.CouponUsage) error {
	usage.SetKeys()

	av, err := attributevalue.MarshalMap(usage)
	if err != nil {
		return errors.Internal("Failed to marshal coupon usage")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.coreTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to record coupon usage")
	}

	return nil
}

// GetUserUsageCount gets the number of times a user has used a coupon
func (r *CouponRepository) GetUserUsageCount(ctx context.Context, couponID, customerID string) (int, error) {
	// Query for usage records matching this coupon and customer
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		FilterExpression:       aws.String("customer_id = :customerID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:        &types.AttributeValueMemberS{Value: "COUPON#" + couponID},
			exprSK:        &types.AttributeValueMemberS{Value: "USAGE#"},
			":customerID": &types.AttributeValueMemberS{Value: customerID},
		},
		Select: types.SelectCount,
	})
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get user usage count")
	}

	return int(result.Count), nil
}

// Ensure interface compliance
var _ domain.CouponRepository = (*CouponRepository)(nil)
