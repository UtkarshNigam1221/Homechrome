package dynamodb

import (
	"context"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// ManifestRepository persists pickup manifests on the orders table under
// PK="MANIFEST#<id>", listed via GSI1 partition "MANIFEST".
type ManifestRepository struct {
	client *Client
}

// NewManifestRepository constructs a repository.
func NewManifestRepository(client *Client) *ManifestRepository {
	return &ManifestRepository{client: client}
}

// Create persists a manifest record.
func (r *ManifestRepository) Create(ctx context.Context, m *domain.Manifest) error {
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	m.SetKeys()
	av, err := attributevalue.MarshalMap(m)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal manifest")
	}
	_, err = r.client.db.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.client.ordersTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create manifest")
	}
	return nil
}

// List returns manifests in reverse chronological order via GSI1.
func (r *ManifestRepository) List(ctx context.Context, limit int) ([]*domain.Manifest, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	out, err := r.client.db.Query(ctx, &awsdynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "MANIFEST"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(int32(limit)), //nolint:gosec // bounded above
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list manifests")
	}
	out2 := make([]*domain.Manifest, 0, len(out.Items))
	for _, item := range out.Items {
		var m domain.Manifest
		if err := attributevalue.UnmarshalMap(item, &m); err != nil {
			continue
		}
		out2 = append(out2, &m)
	}
	return out2, nil
}

// Compile-time interface assertion.
var _ domain.ManifestRepository = (*ManifestRepository)(nil)
