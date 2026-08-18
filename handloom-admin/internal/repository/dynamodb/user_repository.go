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

// UserRepository implements domain.UserRepository
type UserRepository struct {
	client *Client
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(client *Client) *UserRepository {
	return &UserRepository{client: client}
}

// Create creates a new user with atomic email uniqueness guarantee.
// Uses TransactWriteItems to atomically put the user item and an email guard item.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.SetKeys()

	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return errors.Internal("Failed to marshal user")
	}

	// Email guard item ensures uniqueness across concurrent creates
	emailGuard := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: "USER_EMAIL#" + user.Email},
		"SK":          &types.AttributeValueMemberS{Value: "UNIQUENESS"},
		"user_id":     &types.AttributeValueMemberS{Value: user.ID},
		"entity_type": &types.AttributeValueMemberS{Value: "EMAIL_GUARD"},
	}

	_, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName:           aws.String(r.client.coreTable),
					Item:                av,
					ConditionExpression: aws.String("attribute_not_exists(PK)"),
				},
			},
			{
				Put: &types.Put{
					TableName:           aws.String(r.client.coreTable),
					Item:                emailGuard,
					ConditionExpression: aws.String("attribute_not_exists(PK)"),
				},
			},
		},
	})
	if err != nil {
		if isTransactionCanceled(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "User with this email already exists")
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
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
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
			exprPK: &types.AttributeValueMemberS{Value: "USER_EMAIL"},
			exprSK: &types.AttributeValueMemberS{Value: email},
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

// Delete deletes a user and their email guard item atomically.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	// Fetch user to get email for guard cleanup
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	_, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(r.client.coreTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: "USER#" + id},
						"SK": &types.AttributeValueMemberS{Value: skMetadata},
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(r.client.coreTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: "USER_EMAIL#" + user.Email},
						"SK": &types.AttributeValueMemberS{Value: "UNIQUENESS"},
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}

	return nil
}

// List retrieves users with in-memory filtering, sorting, and cursor-based pagination.
// Fetches all users from GSI1, applies Go-level filters, sorts, and paginates.
func (r *UserRepository) List(ctx context.Context, req domain.ListUsersRequest) (*domain.ListUsersResponse, error) {
	// Read every user: the sort below is applied in Go, so a cursor would only
	// order the page it returned rather than the whole list.
	allUsers, err := QueryAll[domain.User](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "USER_EMAIL"},
		},
	}, "Failed to list users")
	if err != nil {
		return nil, err
	}

	// Filter in memory
	filtered := allUsers[:0]
	for _, u := range allUsers {
		if req.Role != nil && u.Role != *req.Role {
			continue
		}
		if req.Status != nil && u.Status != *req.Status {
			continue
		}
		if req.Search != "" {
			s := req.Search
			if !containsIgnoreCase(u.Email, s) &&
				!containsIgnoreCase(u.FirstName, s) &&
				!containsIgnoreCase(u.LastName, s) {
				continue
			}
		}
		filtered = append(filtered, u)
	}

	// Sort
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	descending := req.SortDir != "asc"

	sort.Slice(filtered, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "email":
			less = strings.ToLower(filtered[i].Email) < strings.ToLower(filtered[j].Email)
		case "first_name":
			less = strings.ToLower(filtered[i].FirstName) < strings.ToLower(filtered[j].FirstName)
		case "last_name":
			less = strings.ToLower(filtered[i].LastName) < strings.ToLower(filtered[j].LastName)
		case "role":
			less = string(filtered[i].Role) < string(filtered[j].Role)
		case attrStatus:
			less = string(filtered[i].Status) < string(filtered[j].Status)
		default: // created_at
			less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
		}
		if descending {
			return !less
		}
		return less
	})

	// Paginated in memory because the sort above is applied in Go.
	page, pagination := InMemoryPaginate(filtered, req.PaginationRequest)

	return &domain.ListUsersResponse{
		Users:      page,
		Pagination: pagination,
	}, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	now := time.Now()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
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
