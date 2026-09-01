package dynamodb

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// PaymentRepository implements domain.PaymentRepository
type PaymentRepository struct {
	client *Client
}

// NewPaymentRepository creates a new PaymentRepository
func NewPaymentRepository(client *Client) *PaymentRepository {
	return &PaymentRepository{client: client}
}

// Create creates a new payment record
func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	now := time.Now()
	payment.CreatedAt = now
	payment.UpdatedAt = now
	payment.SetKeys()

	av, err := attributevalue.MarshalMap(payment)
	if err != nil {
		return errors.Internal("Failed to marshal payment")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Payment already exists")
		}
		return errors.Wrap(err, "Failed to create payment")
	}

	return nil
}

// GetByID retrieves a payment by ID
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PAYMENT#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get payment")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Payment not found")
	}

	var payment domain.Payment
	if err := attributevalue.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, errors.Internal("Failed to unmarshal payment")
	}

	return &payment, nil
}

// GetByOrderID retrieves the most recent payment for an order using GSI1
func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName: aws.String(r.client.ordersTable),
		IndexName: aws.String("GSI1"),
		// Narrow on the sort key, not a filter: this partition also holds refunds, and a
		// filter runs after Limit — REFUND# sorts first, so limit 1 found no payment.
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:     &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			exprPrefix: &types.AttributeValueMemberS{Value: "PAYMENT#"},
		},
		ScanIndexForward: aws.Bool(false), // newest first
		Limit:            aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query payment by order ID")
	}

	var payments []domain.Payment
	for _, item := range result.Items {
		var p domain.Payment
		if err := attributevalue.UnmarshalMap(item, &p); err != nil {
			continue
		}
		if p.EntityType == "PAYMENT" {
			payments = append(payments, p)
		}
	}

	if len(payments) == 0 {
		return nil, errors.NotFound("Payment not found")
	}

	return &payments[0], nil
}

// GetByMerchantTxnID retrieves a payment by merchant transaction ID using GSI2
func (r *PaymentRepository) GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*domain.Payment, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "MERCHANT_TXN#" + merchantTxnID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query payment by merchant txn ID")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("Payment not found")
	}

	var payment domain.Payment
	if err := attributevalue.UnmarshalMap(result.Items[0], &payment); err != nil {
		return nil, errors.Internal("Failed to unmarshal payment")
	}

	return &payment, nil
}

// UpdateStatus updates the payment status and additional fields using a dynamic UpdateItem expression
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus, updates map[string]interface{}) error {
	du, err := buildDynamicUpdate(string(status), updates)
	if err != nil {
		return err
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PAYMENT#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression:          aws.String(du.Expression),
		ExpressionAttributeNames:  du.AttrNames,
		ExpressionAttributeValues: du.AttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Payment not found")
		}
		return errors.Wrap(err, "Failed to update payment status")
	}

	return nil
}

// Ensure interface compliance
var _ domain.PaymentRepository = (*PaymentRepository)(nil)

// AddRefundAmount implements domain.PaymentRepository.
func (r *PaymentRepository) AddRefundAmount(ctx context.Context, id string, amount int64) (int64, error) {
	result, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PAYMENT#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("ADD refund_amount :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":amount": &types.AttributeValueMemberN{Value: strconv.FormatInt(amount, 10)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
		ReturnValues:        types.ReturnValueUpdatedNew,
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return 0, errors.NotFound("Payment not found")
		}
		return 0, errors.Wrap(err, "Failed to record refund amount")
	}

	attr, ok := result.Attributes["refund_amount"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, errors.Internal("Failed to read refunded total")
	}

	total, err := strconv.ParseInt(attr.Value, 10, 64)
	if err != nil {
		return 0, errors.Internal("Failed to parse refunded total")
	}

	return total, nil
}
