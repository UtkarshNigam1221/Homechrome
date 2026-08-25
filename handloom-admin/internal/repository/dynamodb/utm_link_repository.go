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

// utmLinkNotFound is the message every miss returns, so a 404 reads the same
// whether it came from a get, an update or a delete.
const utmLinkNotFound = "UTM link not found"

// UTMLinkRepository implements domain.UTMLinkRepository
type UTMLinkRepository struct {
	client *Client
}

// NewUTMLinkRepository creates a new UTMLinkRepository
func NewUTMLinkRepository(client *Client) *UTMLinkRepository {
	return &UTMLinkRepository{client: client}
}

// Create creates a new UTM link
func (r *UTMLinkRepository) Create(ctx context.Context, link *domain.UTMLink) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now
	// After CreatedAt, never before: GSI1SK is derived from it.
	link.SetKeys()

	item, err := attributevalue.MarshalMap(link)
	if err != nil {
		return errors.Internal(err)
	}

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
			return errors.Conflict("UTM link already exists")
		}
		return errors.Internal(err)
	}

	return nil
}

// GetByID retrieves a UTM link by ID
func (r *UTMLinkRepository) GetByID(ctx context.Context, id string) (*domain.UTMLink, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "UTM_LINK#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodeNotFound, utmLinkNotFound)
	}

	var link domain.UTMLink
	if err := attributevalue.UnmarshalMap(result.Item, &link); err != nil {
		return nil, errors.Internal(err)
	}

	return &link, nil
}

// Update updates an existing UTM link. Keys are left as they are — GSI1SK carries
// the original CreatedAt, so list ordering stays stable across edits.
func (r *UTMLinkRepository) Update(ctx context.Context, link *domain.UTMLink) error {
	link.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(link)
	if err != nil {
		return errors.Internal(err)
	}

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
			return errors.New(errors.ErrCodeNotFound, utmLinkNotFound)
		}
		return errors.Internal(err)
	}

	return nil
}

// Delete deletes a UTM link by ID
func (r *UTMLinkRepository) Delete(ctx context.Context, id string) error {
	condition := expression.AttributeExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "UTM_LINK#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.New(errors.ErrCodeNotFound, utmLinkNotFound)
		}
		return errors.Internal(err)
	}

	return nil
}

// List retrieves UTM links, newest first
func (r *UTMLinkRepository) List(ctx context.Context, req domain.ListUTMLinksRequest) (*domain.ListUTMLinksResponse, error) {
	req.Limit = DefaultLimit(req.Limit)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "UTM_LINK#ALL"},
		},
		ScanIndexForward: aws.Bool(false), // GSI1SK starts with CreatedAt, so descending = newest first
	}

	links, err := QueryAll[domain.UTMLink](ctx, r.client.db, input, "Failed to list UTM links")
	if err != nil {
		return nil, err
	}

	if req.Search != "" {
		var filtered []*domain.UTMLink
		for _, l := range links {
			if containsIgnoreCase(l.Name, req.Search) || containsIgnoreCase(l.UTMCampaign, req.Search) {
				filtered = append(filtered, l)
			}
		}
		links = filtered
	}

	// Paginated in memory because the search above is applied in Go: a cursor
	// would page the unfiltered index and hand back short, uneven pages.
	paged, pg := InMemoryPaginate(links, req.PaginationRequest)

	return &domain.ListUTMLinksResponse{
		Links:      paged,
		Pagination: pg,
	}, nil
}
