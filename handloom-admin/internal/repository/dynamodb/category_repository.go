package dynamodb

import (
	"context"
	stderrors "errors"
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

// List retrieves categories with cursor-based pagination
func (r *CategoryRepository) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	limit := DefaultLimit(req.Limit)

	exclusiveStartKey, err := DecodeCursor(req.Cursor)
	if err != nil {
		return nil, err
	}

	// Query GSI1 using CATEGORY#ALL partition key
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("CATEGORY#ALL"))
	builder := expression.NewBuilder().WithKeyCondition(keyExpr)

	hasFilter := req.Status != nil
	if hasFilter {
		builder = builder.WithFilter(expression.Name("status").Equal(expression.Value(string(*req.Status))))
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, errors.Internal(err)
	}

	// Over-fetch when filters are active to reduce round-trips
	fetchLimit := int32(limit)
	if hasFilter {
		fetchLimit = int32(limit * 3)
		if fetchLimit > 300 {
			fetchLimit = 300
		}
	}

	var collected []*domain.Category
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
			return nil, errors.Internal(err)
		}

		var batch []*domain.Category
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &batch); err != nil {
			return nil, errors.Internal(err)
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

	return &domain.ListCategoriesResponse{
		Categories: collected,
		Pagination: BuildPaginationResponse(limit, lastEvaluatedKey),
	}, nil
}

// IncrementProductCount increments the product count
func (r *CategoryRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	update := expression.Add(expression.Name("product_count"), expression.Value(delta))
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
