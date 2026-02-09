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

// UserRepository implements domain.UserRepository
type UserRepository struct {
	client *Client
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(client *Client) *UserRepository {
	return &UserRepository{client: client}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.SetKeys()

	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return errors.Internal("Failed to marshal user")
	}

	// Check if email already exists
	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "User already exists")
		}
		return errors.Wrap(err, "Failed to create user")
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get user")
	}

	if result.Item == nil {
		return nil, errors.NotFound("User not found")
	}

	var user domain.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, errors.Internal("Failed to unmarshal user")
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER_EMAIL"},
			":sk": &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query user by email")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("User not found")
	}

	var user domain.User
	if err := attributevalue.UnmarshalMap(result.Items[0], &user); err != nil {
		return nil, errors.Internal("Failed to unmarshal user")
	}

	return &user, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	user.SetKeys()

	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return errors.Internal("Failed to marshal user")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("User not found")
		}
		return errors.Wrap(err, "Failed to update user")
	}

	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("User not found")
		}
		return errors.Wrap(err, "Failed to delete user")
	}

	return nil
}

// List retrieves users with cursor-based pagination using GSI1.
func (r *UserRepository) List(ctx context.Context, req domain.ListUsersRequest) (*domain.ListUsersResponse, error) {
	limit := DefaultLimit(req.Limit)

	exclusiveStartKey, err := DecodeCursor(req.Cursor)
	if err != nil {
		return nil, err
	}

	// Query GSI1 using USER_EMAIL partition key (all users share this GSI1PK)
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("USER_EMAIL"))
	builder := expression.NewBuilder().WithKeyCondition(keyExpr)

	// Build filter expression for role, status, search
	hasFilter := false
	var filterExpr expression.ConditionBuilder
	if req.Role != nil {
		filterExpr = expression.Name("role").Equal(expression.Value(string(*req.Role)))
		hasFilter = true
	}
	if req.Status != nil {
		f := expression.Name("status").Equal(expression.Value(string(*req.Status)))
		if hasFilter {
			filterExpr = filterExpr.And(f)
		} else {
			filterExpr = f
			hasFilter = true
		}
	}
	if req.Search != "" {
		f := expression.Name("email").Contains(req.Search).
			Or(expression.Name("first_name").Contains(req.Search)).
			Or(expression.Name("last_name").Contains(req.Search))
		if hasFilter {
			filterExpr = filterExpr.And(f)
		} else {
			filterExpr = f
			hasFilter = true
		}
	}

	if hasFilter {
		builder = builder.WithFilter(filterExpr)
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	// Over-fetch when filters are active to reduce round-trips
	fetchLimit := int32(limit)
	if hasFilter {
		fetchLimit = int32(limit * 3)
		if fetchLimit > 300 {
			fetchLimit = 300
		}
	}

	var collected []*domain.User
	var lastEvaluatedKey map[string]types.AttributeValue
	currentStartKey := exclusiveStartKey

	for {
		queryInput := &dynamodb.QueryInput{
			TableName:                 aws.String(r.client.coreTable),
			IndexName:                 aws.String("GSI1"),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			Limit:                     aws.Int32(fetchLimit),
			ExclusiveStartKey:         currentStartKey,
		}
		if expr.Filter() != nil {
			queryInput.FilterExpression = expr.Filter()
		}

		result, err := r.client.db.Query(ctx, queryInput)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list users")
		}

		var batch []*domain.User
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &batch); err != nil {
			return nil, errors.Internal("Failed to unmarshal users")
		}

		collected = append(collected, batch...)
		lastEvaluatedKey = result.LastEvaluatedKey

		if len(collected) >= limit || lastEvaluatedKey == nil {
			break
		}

		currentStartKey = lastEvaluatedKey
	}

	// Trim to exactly limit items if we over-collected
	if len(collected) > limit {
		collected = collected[:limit]
		// Build cursor from the last returned item's keys
		last := collected[limit-1]
		lastEvaluatedKey = map[string]types.AttributeValue{
			"PK":     &types.AttributeValueMemberS{Value: last.PK},
			"SK":     &types.AttributeValueMemberS{Value: last.SK},
			"GSI1PK": &types.AttributeValueMemberS{Value: last.GSI1PK},
			"GSI1SK": &types.AttributeValueMemberS{Value: last.GSI1SK},
		}
	}

	return &domain.ListUsersResponse{
		Users:      collected,
		Pagination: BuildPaginationResponse(limit, lastEvaluatedKey),
	}, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	now := time.Now()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET last_login_at = :t, updated_at = :u"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":t": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":u": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("User not found")
		}
		return errors.Wrap(err, "Failed to update last login")
	}

	return nil
}

// Ensure interface compliance
var _ domain.UserRepository = (*UserRepository)(nil)
