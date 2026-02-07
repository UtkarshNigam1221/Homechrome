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

// List retrieves users with pagination and filters
func (r *UserRepository) List(ctx context.Context, req domain.ListUsersRequest) (*domain.ListUsersResponse, error) {
	// Build filter expression
	filterBuilder := expression.Name("entity_type").Equal(expression.Value("USER"))

	if req.Role != nil {
		filterBuilder = filterBuilder.And(expression.Name("role").Equal(expression.Value(*req.Role)))
	}

	if req.Status != nil {
		filterBuilder = filterBuilder.And(expression.Name("status").Equal(expression.Value(*req.Status)))
	}

	if req.Search != "" {
		// Search in email, first_name, last_name
		searchFilter := expression.Name("email").Contains(req.Search).
			Or(expression.Name("first_name").Contains(req.Search)).
			Or(expression.Name("last_name").Contains(req.Search))
		filterBuilder = filterBuilder.And(searchFilter)
	}

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	// Scan for users (in production, consider using a GSI for better performance)
	input := &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.coreTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := r.client.db.Scan(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list users")
	}

	var users []*domain.User
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &users); err != nil {
		return nil, errors.Internal("Failed to unmarshal users")
	}

	// Manual pagination
	totalCount := int64(len(users))
	perPage := req.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * perPage
	end := start + perPage
	if start >= int(totalCount) {
		users = []*domain.User{}
	} else if end > int(totalCount) {
		users = users[start:]
	} else {
		users = users[start:end]
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	return &domain.ListUsersResponse{
		Users: users,
		Pagination: domain.PaginationResponse{
			CurrentPage: page,
			PerPage:     perPage,
			TotalCount:  totalCount,
			TotalPages:  totalPages,
		},
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
