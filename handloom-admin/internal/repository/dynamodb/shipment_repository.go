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

// ShipmentRepository implements domain.ShipmentRepository
type ShipmentRepository struct {
	client *Client
}

// NewShipmentRepository creates a new ShipmentRepository
func NewShipmentRepository(client *Client) *ShipmentRepository {
	return &ShipmentRepository{client: client}
}

// Create creates a new shipment record
func (r *ShipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
	now := time.Now()
	shipment.CreatedAt = now
	shipment.UpdatedAt = now
	shipment.SetKeys()

	av, err := attributevalue.MarshalMap(shipment)
	if err != nil {
		return errors.Internal("Failed to marshal shipment")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Shipment already exists")
		}
		return errors.Wrap(err, "Failed to create shipment")
	}

	return nil
}

// GetByOrderID retrieves the latest shipment for an order
func (r *ShipmentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":       &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			":skPrefix": &types.AttributeValueMemberS{Value: "SHIPMENT#"},
		},
		Limit:            aws.Int32(1),
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query shipment by order ID")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("Shipment not found")
	}

	var shipment domain.Shipment
	if err := attributevalue.UnmarshalMap(result.Items[0], &shipment); err != nil {
		return nil, errors.Internal("Failed to unmarshal shipment")
	}

	return &shipment, nil
}

// UpdateStatus updates the status of a shipment with additional dynamic fields
func (r *ShipmentRepository) UpdateStatus(ctx context.Context, orderID, shipmentID string, status domain.ShipmentStatus, updates map[string]interface{}) error {
	du, err := buildDynamicUpdate(string(status), updates)
	if err != nil {
		return err
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			"SK": &types.AttributeValueMemberS{Value: "SHIPMENT#" + shipmentID},
		},
		UpdateExpression:          aws.String(du.Expression),
		ExpressionAttributeNames:  du.AttrNames,
		ExpressionAttributeValues: du.AttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Shipment not found")
		}
		return errors.Wrap(err, "Failed to update shipment status")
	}

	return nil
}

// Ensure interface compliance
var _ domain.ShipmentRepository = (*ShipmentRepository)(nil)
