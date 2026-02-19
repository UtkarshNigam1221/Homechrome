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

// ArtisanRepository implements domain.ArtisanRepository
type ArtisanRepository struct {
	client *Client
}

// NewArtisanRepository creates a new ArtisanRepository
func NewArtisanRepository(client *Client) *ArtisanRepository {
	return &ArtisanRepository{
		client: client,
	}
}

// Create creates a new artisan
func (r *ArtisanRepository) Create(ctx context.Context, artisan *domain.Artisan) error {
	now := time.Now()
	artisan.CreatedAt = now
	artisan.UpdatedAt = now
	artisan.SetKeys()

	av, err := attributevalue.MarshalMap(artisan)
	if err != nil {
		return errors.Internal("Failed to marshal artisan")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Artisan already exists")
		}
		return errors.Wrap(err, "Failed to create artisan")
	}

	return nil
}

// GetByID retrieves an artisan by ID
func (r *ArtisanRepository) GetByID(ctx context.Context, id string) (*domain.Artisan, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ARTISAN#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get artisan")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Artisan")
	}

	var artisan domain.Artisan
	if err := attributevalue.UnmarshalMap(result.Item, &artisan); err != nil {
		return nil, errors.Internal("Failed to unmarshal artisan")
	}

	return &artisan, nil
}

// Update updates an artisan
func (r *ArtisanRepository) Update(ctx context.Context, artisan *domain.Artisan) error {
	artisan.UpdatedAt = time.Now()
	artisan.SetKeys()

	av, err := attributevalue.MarshalMap(artisan)
	if err != nil {
		return errors.Internal("Failed to marshal artisan")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Artisan")
		}
		return errors.Wrap(err, "Failed to update artisan")
	}

	return nil
}

// Delete deletes an artisan
func (r *ArtisanRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ARTISAN#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Artisan")
		}
		return errors.Wrap(err, "Failed to delete artisan")
	}

	return nil
}

// List lists artisans with filters
func (r *ArtisanRepository) List(ctx context.Context, req domain.ListArtisansRequest) (*domain.ListArtisansResponse, error) {
	// TODO: Implement with DynamoDB scan/query
	return &domain.ListArtisansResponse{
		Artisans:   []*domain.Artisan{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// UpdateStats updates artisan statistics
func (r *ArtisanRepository) UpdateStats(ctx context.Context, id string, productCount int, totalSales int64, totalEarnings int64) error {
	// TODO: Implement with DynamoDB update expression
	return nil
}

// GetPayouts retrieves payouts for an artisan
func (r *ArtisanRepository) GetPayouts(ctx context.Context, artisanID string, pagination domain.PaginationRequest) (*domain.ListArtisanPayoutsResponse, error) {
	// TODO: Implement with GSI query
	return &domain.ListArtisanPayoutsResponse{
		Payouts:    []*domain.ArtisanPayout{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// CreatePayout creates a new payout
func (r *ArtisanRepository) CreatePayout(ctx context.Context, payout *domain.ArtisanPayout) error {
	// TODO: Implement
	return nil
}

// GetProducts retrieves products for an artisan
func (r *ArtisanRepository) GetProducts(ctx context.Context, artisanID string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	// TODO: Implement with GSI query on artisan_id
	return &domain.ListProductsResponse{
		Products:   []*domain.Product{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// Search searches artisans by query
func (r *ArtisanRepository) Search(ctx context.Context, query string, pagination domain.PaginationRequest) (*domain.ListArtisansResponse, error) {
	// TODO: Implement with scan and filter or OpenSearch
	return &domain.ListArtisansResponse{
		Artisans:   []*domain.Artisan{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// Ensure interface compliance
var _ domain.ArtisanRepository = (*ArtisanRepository)(nil)
