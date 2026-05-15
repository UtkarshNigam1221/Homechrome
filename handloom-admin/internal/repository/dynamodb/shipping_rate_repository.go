package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// shippingRateBatchSize is the DynamoDB BatchWriteItem per-request limit.
const shippingRateBatchSize = 25

// shippingRateMaxRetries caps retries for items returned in UnprocessedItems.
const shippingRateMaxRetries = 5

// ShippingRateRepository persists rate matrix rows in the shipping table.
type ShippingRateRepository struct {
	client *Client
}

// NewShippingRateRepository constructs a repository.
func NewShippingRateRepository(client *Client) *ShippingRateRepository {
	return &ShippingRateRepository{client: client}
}

// Get retrieves a single rate row for a (zone, weight slab) pair.
func (r *ShippingRateRepository) Get(ctx context.Context, zone string, weightSlabGrams int) (*domain.ShippingRate, error) {
	pk := fmt.Sprintf("RATE#%s#%d", zone, weightSlabGrams)
	out, err := r.client.db.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.client.shippingTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get shipping rate")
	}
	if out.Item == nil {
		return nil, errors.NotFound("Shipping rate not found")
	}
	var rate domain.ShippingRate
	if err := attributevalue.UnmarshalMap(out.Item, &rate); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal shipping rate")
	}
	return &rate, nil
}

// Upsert writes a single rate row.
func (r *ShippingRateRepository) Upsert(ctx context.Context, rate *domain.ShippingRate) error {
	rate.SetKeys()
	av, err := attributevalue.MarshalMap(rate)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal shipping rate")
	}
	_, err = r.client.db.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.client.shippingTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to upsert shipping rate")
	}
	return nil
}

// BatchUpsert writes rate rows in chunks of 25 (DynamoDB BatchWriteItem limit),
// retrying UnprocessedItems with exponential backoff so throttled writes don't drop.
func (r *ShippingRateRepository) BatchUpsert(ctx context.Context, rates []*domain.ShippingRate) error {
	for i := 0; i < len(rates); i += shippingRateBatchSize {
		end := i + shippingRateBatchSize
		if end > len(rates) {
			end = len(rates)
		}
		writes := make([]types.WriteRequest, 0, end-i)
		for _, rate := range rates[i:end] {
			rate.SetKeys()
			av, err := attributevalue.MarshalMap(rate)
			if err != nil {
				return errors.Wrap(err, "Failed to marshal shipping rate")
			}
			writes = append(writes, types.WriteRequest{PutRequest: &types.PutRequest{Item: av}})
		}
		if err := r.batchWriteWithRetry(ctx, writes); err != nil {
			return err
		}
	}
	return nil
}

// batchWriteWithRetry writes a chunk and retries unprocessed items with linear backoff.
func (r *ShippingRateRepository) batchWriteWithRetry(ctx context.Context, items []types.WriteRequest) error {
	pending := items
	for attempt := 1; attempt <= shippingRateMaxRetries; attempt++ {
		out, err := r.client.db.BatchWriteItem(ctx, &awsdynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{r.client.shippingTable: pending},
		})
		if err != nil {
			return errors.Wrap(err, "Failed to batch upsert shipping rates")
		}
		unprocessed := out.UnprocessedItems[r.client.shippingTable]
		if len(unprocessed) == 0 {
			return nil
		}
		pending = unprocessed
		if attempt < shippingRateMaxRetries {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}
	return errors.Internal(fmt.Sprintf("batch upsert: %d shipping rates still unprocessed after %d attempts", len(pending), shippingRateMaxRetries))
}

// ListAll scans every SHIPPING_RATE row. Intended for admin tooling, not hot paths.
func (r *ShippingRateRepository) ListAll(ctx context.Context) ([]*domain.ShippingRate, error) {
	out, err := r.client.db.Scan(ctx, &awsdynamodb.ScanInput{
		TableName:        aws.String(r.client.shippingTable),
		FilterExpression: aws.String("entity_type = :et"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":et": &types.AttributeValueMemberS{Value: domain.EntityTypeShipping},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to scan shipping rates")
	}
	rates := make([]*domain.ShippingRate, 0, len(out.Items))
	for _, item := range out.Items {
		var rate domain.ShippingRate
		if err := attributevalue.UnmarshalMap(item, &rate); err != nil {
			continue
		}
		rates = append(rates, &rate)
	}
	return rates, nil
}

// Compile-time interface assertion.
var _ domain.ShippingRateRepository = (*ShippingRateRepository)(nil)
