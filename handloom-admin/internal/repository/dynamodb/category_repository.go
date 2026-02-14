package dynamodb

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CategoryRepository implements domain.CategoryRepository
type CategoryRepository struct {
	client *Client
}

// NewCategoryRepository creates a new CategoryRepository
func NewCategoryRepository(client *Client) *CategoryRepository {
	return &CategoryRepository{client: client}
}

// Create creates a new category
func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	category.SetKeys()
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(category)
	if err != nil {
		return errors.Internal(err)
	}

	// Condition to ensure unique ID
	condition := expression.AttributeNotExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(r.client.coreTable),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.Conflict("Category already exists")
		}
		return errors.Internal(err)
	}

	return nil
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CATEGORY#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodeCategoryNotFound, "Category not found")
	}

	var category domain.Category
	if err := attributevalue.UnmarshalMap(result.Item, &category); err != nil {
		return nil, errors.Internal(err)
	}

	return &category, nil
}

// Update updates an existing category
func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	category.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(category)
	if err != nil {
		return errors.Internal(err)
	}

	// Condition to ensure category exists
	condition := expression.AttributeExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(r.client.coreTable),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.New(errors.ErrCodeCategoryNotFound, "Category not found")
		}
		return errors.Internal(err)
	}

	return nil
}

// Delete deletes a category by ID
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	condition := expression.AttributeExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CATEGORY#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.New(errors.ErrCodeCategoryNotFound, "Category not found")
		}
		return errors.Internal(err)
	}

	return nil
}

// List retrieves categories with in-memory search, sort, and pagination.
// Fetches all categories from GSI1, then applies filters/sort/paginate in memory.
func (r *CategoryRepository) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	// Fetch all categories from GSI1
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("CATEGORY#ALL"))
	expr, err := expression.NewBuilder().WithKeyCondition(keyExpr).Build()
	if err != nil {
		return nil, errors.Internal(err)
	}

	var all []*domain.Category
	var lastKey map[string]types.AttributeValue

	for {
		queryInput := &dynamodb.QueryInput{
			TableName:                 aws.String(r.client.coreTable),
			IndexName:                 aws.String("GSI1"),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			ExclusiveStartKey:         lastKey,
		}

		result, err := r.client.db.Query(ctx, queryInput)
		if err != nil {
			return nil, errors.Internal(err)
		}

		var batch []*domain.Category
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &batch); err != nil {
			return nil, errors.Internal(err)
		}
		all = append(all, batch...)
		lastKey = result.LastEvaluatedKey
		if lastKey == nil {
			break
		}
	}

	// Filter by status
	if req.Status != nil {
		filtered := make([]*domain.Category, 0, len(all))
		for _, c := range all {
			if c.Status == *req.Status {
				filtered = append(filtered, c)
			}
		}
		all = filtered
	}

	// Filter by search (case-insensitive name contains)
	if req.Search != "" {
		search := strings.ToLower(req.Search)
		filtered := make([]*domain.Category, 0, len(all))
		for _, c := range all {
			if strings.Contains(strings.ToLower(c.Name), search) {
				filtered = append(filtered, c)
			}
		}
		all = filtered
	}

	// Sort (default: created_at desc)
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	// Paginate
	paged, pg := InMemoryPaginate(all, req.PaginationRequest)

	return &domain.ListCategoriesResponse{
		Categories: paged,
		Pagination: pg,
	}, nil
}

// IncrementProductCount increments the product count and updates the timestamp
func (r *CategoryRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	update := expression.Add(expression.Name("product_count"), expression.Value(delta)).
		Set(expression.Name("updated_at"), expression.Value(time.Now().UTC()))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CATEGORY#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

// Ensure interface compliance
var _ domain.CategoryRepository = (*CategoryRepository)(nil)
