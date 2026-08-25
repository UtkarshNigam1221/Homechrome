package dynamodb

import (
	"context"
	"sort"
	"strings"
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

// couponCodeIndex points a code at the coupon that currently holds it. A separate item
// rather than a GSI because only a conditional put can make a code unique, and only a
// real item is strongly consistent — a GSI is neither.
type couponCodeIndex struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	EntityType string `dynamodbav:"entity_type"`
	CouponID   string `dynamodbav:"coupon_id"`
}

func marshalCouponCodeIndex(code, couponID string) (map[string]types.AttributeValue, error) {
	idx := couponCodeIndex{
		PK:         "CODE#" + strings.ToUpper(code),
		SK:         skMetadata,
		EntityType: "COUPON_CODE_INDEX",
		CouponID:   couponID,
	}
	av, err := attributevalue.MarshalMap(idx)
	if err != nil {
		return nil, errors.Internal("Failed to marshal coupon code index")
	}
	return av, nil
}

// Create writes the coupon and its code pointer in one transaction. Written separately,
// a failure on the second leaves a coupon GetByCode cannot find.
func (r *CouponRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	now := time.Now()
	coupon.CreatedAt = now
	coupon.UpdatedAt = now
	coupon.Code = strings.ToUpper(coupon.Code)
	coupon.SetKeys()

	av, err := attributevalue.MarshalMap(coupon)
	if err != nil {
		return errors.Internal("Failed to marshal coupon")
	}

	codeAV, err := marshalCouponCodeIndex(coupon.Code, coupon.ID)
	if err != nil {
		return err
	}

	if _, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{
				TableName:           aws.String(r.client.couponsTable),
				Item:                av,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &types.Put{
				TableName:           aws.String(r.client.couponsTable),
				Item:                codeAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
		},
	}); err != nil {
		if isTransactionCanceled(err) {
			// Either the id is taken or the code is. Both mean pick another code.
			return errors.New(errors.ErrCodeAlreadyExists, "Coupon code already exists")
		}
		return errors.Wrap(err, "Failed to create coupon")
	}

	return nil
}

// GetByID retrieves a coupon by ID
func (r *CouponRepository) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.couponsTable),
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

// GetByCode resolves a code through the pointer, then reads the coupon. Two O(1) reads,
// both strongly consistent, and no partition holds more than one code.
func (r *CouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CODE#" + strings.ToUpper(strings.TrimSpace(code))},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to look up coupon code")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Coupon")
	}

	var idx couponCodeIndex
	if err := attributevalue.UnmarshalMap(result.Item, &idx); err != nil {
		return nil, errors.Internal("Failed to unmarshal coupon code index")
	}

	return r.GetByID(ctx, idx.CouponID)
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
		TableName:           aws.String(r.client.couponsTable),
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
		TableName: aws.String(r.client.couponsTable),
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

// List reads the public listing partition. Personal codes carry a different GSI1PK, so
// they are absent by construction rather than by filter.
//
// Reads the partition whole because the filters and sort below are applied in Go — a
// DynamoDB cursor would only order the page it returned, not the whole list. Correct
// while public coupons number in the dozens; past a few hundred, status belongs in the
// partition key.
func (r *CouponRepository) List(ctx context.Context, req domain.ListCouponsRequest) (*domain.ListCouponsResponse, error) {
	all, err := QueryAll[domain.Coupon](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "COUPON#ALL"},
		},
	}, "Failed to list coupons")
	if err != nil {
		return nil, err
	}

	filtered := all[:0]
	for _, c := range all {
		if req.Status != nil && c.Status != *req.Status {
			continue
		}
		if req.Type != nil && c.Type != *req.Type {
			continue
		}
		if req.IsActive != nil && (c.Status == domain.CouponStatusActive) != *req.IsActive {
			continue
		}
		if req.Search != "" &&
			!containsIgnoreCase(c.Code, req.Search) &&
			!containsIgnoreCase(c.Name, req.Search) {
			continue
		}
		filtered = append(filtered, c)
	}

	// Newest first, matching every other admin list.
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	page, pagination := InMemoryPaginate(filtered, req.PaginationRequest)

	return &domain.ListCouponsResponse{
		Coupons:    page,
		Pagination: pagination,
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
		TableName: aws.String(r.client.couponsTable),
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
		TableName:              aws.String(r.client.couponsTable),
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

// IncrementUsage atomically claims one redemption.
// TODO(task-5): replace with the real conditional-update implementation.
func (r *CouponRepository) IncrementUsage(ctx context.Context, couponID string) (bool, error) {
	return false, nil
}

// GetCustomerUsage returns how many times this customer has redeemed this coupon.
// TODO(task-5): replace with the real implementation.
func (r *CouponRepository) GetCustomerUsage(ctx context.Context, customerID, couponID string) (int, error) {
	return 0, nil
}

// IncrementCustomerUsage bumps this customer's count for this coupon.
// TODO(task-5): replace with the real implementation.
func (r *CouponRepository) IncrementCustomerUsage(ctx context.Context, customerID, couponID string) error {
	return nil
}

// Ensure interface compliance
var _ domain.CouponRepository = (*CouponRepository)(nil)
