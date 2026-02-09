package dynamodb

import (
	"context"
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

// DesignRepository implements domain.DesignRepository
type DesignRepository struct {
	client *Client
}

// NewDesignRepository creates a new DesignRepository
func NewDesignRepository(client *Client) *DesignRepository {
	return &DesignRepository{client: client}
}

// Create creates a new design
func (r *DesignRepository) Create(ctx context.Context, design *domain.Design) error {
	now := time.Now()
	design.CreatedAt = now
	design.UpdatedAt = now
	design.SetKeys()

	av, err := attributevalue.MarshalMap(design)
	if err != nil {
		return errors.Internal("Failed to marshal design")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Design already exists")
		}
		return errors.Wrap(err, "Failed to create design")
	}

	return nil
}

// GetByID retrieves a design by ID
func (r *DesignRepository) GetByID(ctx context.Context, id string) (*domain.Design, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DESIGN#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get design")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Design not found")
	}

	var design domain.Design
	if err := attributevalue.UnmarshalMap(result.Item, &design); err != nil {
		return nil, errors.Internal("Failed to unmarshal design")
	}

	return &design, nil
}

// Update updates an existing design
func (r *DesignRepository) Update(ctx context.Context, design *domain.Design) error {
	design.UpdatedAt = time.Now()
	design.SetKeys()

	av, err := attributevalue.MarshalMap(design)
	if err != nil {
		return errors.Internal("Failed to marshal design")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Design not found")
		}
		return errors.Wrap(err, "Failed to update design")
	}

	return nil
}

// Delete deletes a design by ID
func (r *DesignRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DESIGN#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Design not found")
		}
		return errors.Wrap(err, "Failed to delete design")
	}

	return nil
}

// List retrieves designs with filters
func (r *DesignRepository) List(ctx context.Context, req domain.ListDesignsRequest) (*domain.ListDesignsResponse, error) {
	filterBuilder := expression.Name("entity_type").Equal(expression.Value("DESIGN"))

	if req.CategoryID != nil {
		filterBuilder = filterBuilder.And(expression.Name("category_id").Equal(expression.Value(*req.CategoryID)))
	}

	if req.Status != nil {
		filterBuilder = filterBuilder.And(expression.Name("status").Equal(expression.Value(*req.Status)))
	}

	if req.Search != "" {
		searchLower := strings.ToLower(req.Search)
		searchFilter := expression.Name("name").Contains(searchLower).
			Or(expression.Name("description").Contains(searchLower))
		filterBuilder = filterBuilder.And(searchFilter)
	}

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	result, err := r.client.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.coreTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list designs")
	}

	var designs []*domain.Design
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &designs); err != nil {
		return nil, errors.Internal("Failed to unmarshal designs")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(designs, req.PaginationRequest)

	return &domain.ListDesignsResponse{
		Designs:    paged,
		Pagination: pg,
	}, nil
}

// IncrementProductCount increments the product count
func (r *DesignRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DESIGN#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET product_count = product_count + :delta, updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":delta": &types.AttributeValueMemberN{Value: intToString(delta)},
			":now":   &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Design not found")
		}
		return errors.Wrap(err, "Failed to increment product count")
	}

	return nil
}

// Ensure interface compliance
var _ domain.DesignRepository = (*DesignRepository)(nil)
