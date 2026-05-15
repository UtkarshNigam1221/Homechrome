package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// PincodeRepository persists pincode → zone cache rows.
type PincodeRepository struct {
	client *Client
}

// NewPincodeRepository constructs a repository.
func NewPincodeRepository(client *Client) *PincodeRepository {
	return &PincodeRepository{client: client}
}

// Get retrieves a cached pincode row. Returns NotFound on cache miss.
func (r *PincodeRepository) Get(ctx context.Context, pincode string) (*domain.PincodeZone, error) {
	out, err := r.client.db.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.client.shippingTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PIN#" + pincode},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get pincode")
	}
	if out.Item == nil {
		return nil, errors.NotFound("Pincode not found")
	}
	var pz domain.PincodeZone
	if err := attributevalue.UnmarshalMap(out.Item, &pz); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal pincode")
	}
	return &pz, nil
}

// Upsert writes (or refreshes) a single pincode row.
func (r *PincodeRepository) Upsert(ctx context.Context, pz *domain.PincodeZone) error {
	pz.SetKeys()
	av, err := attributevalue.MarshalMap(pz)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal pincode")
	}
	_, err = r.client.db.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.client.shippingTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to upsert pincode")
	}
	return nil
}

// Compile-time interface assertion.
var _ domain.PincodeRepository = (*PincodeRepository)(nil)
