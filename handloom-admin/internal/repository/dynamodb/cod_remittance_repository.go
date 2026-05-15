package dynamodb

import (
	"context"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CODRemittanceRepository persists COD remittance payouts.
type CODRemittanceRepository struct {
	client *Client
}

// NewCODRemittanceRepository constructs a repository.
func NewCODRemittanceRepository(client *Client) *CODRemittanceRepository {
	return &CODRemittanceRepository{client: client}
}

// Get retrieves a remittance by its carrier reference.
func (r *CODRemittanceRepository) Get(ctx context.Context, remittanceRef string) (*domain.CODRemittance, error) {
	out, err := r.client.db.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.client.shippingTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REMIT#" + remittanceRef},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get remittance")
	}
	if out.Item == nil {
		return nil, errors.NotFound("Remittance not found")
	}
	var rem domain.CODRemittance
	if err := attributevalue.UnmarshalMap(out.Item, &rem); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal remittance")
	}
	return &rem, nil
}

// Upsert writes a remittance row (used both for first ingest and reconciliation updates).
func (r *CODRemittanceRepository) Upsert(ctx context.Context, rem *domain.CODRemittance) error {
	rem.SetKeys()
	av, err := attributevalue.MarshalMap(rem)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal remittance")
	}
	_, err = r.client.db.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.client.shippingTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to upsert remittance")
	}
	return nil
}

// ListByStatus queries remittances by reconciliation status via the
// entity-status-index GSI (PK=entity_type, SK=status).
func (r *CODRemittanceRepository) ListByStatus(ctx context.Context, status domain.CODRemittanceStatus, limit int) ([]*domain.CODRemittance, error) {
	if limit < 0 {
		limit = 0
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	out, err := r.client.db.Query(ctx, &awsdynamodb.QueryInput{
		TableName:              aws.String(r.client.shippingTable),
		IndexName:              aws.String("entity-status-index"),
		KeyConditionExpression: aws.String("entity_type = :et AND #s = :s"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":et": &types.AttributeValueMemberS{Value: domain.EntityTypeCODRemittance},
			":s":  &types.AttributeValueMemberS{Value: string(status)},
		},
		Limit: aws.Int32(int32(limit)), //nolint:gosec // bounded above
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query remittances")
	}
	rems := make([]*domain.CODRemittance, 0, len(out.Items))
	for _, item := range out.Items {
		var rem domain.CODRemittance
		if err := attributevalue.UnmarshalMap(item, &rem); err != nil {
			continue
		}
		rems = append(rems, &rem)
	}
	return rems, nil
}

// Compile-time interface assertion.
var _ domain.CODRemittanceRepository = (*CODRemittanceRepository)(nil)
