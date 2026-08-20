package dynamodb

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// OrderRepository implements domain.OrderRepository
type OrderRepository struct {
	client *Client
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(client *Client) *OrderRepository {
	return &OrderRepository{client: client}
}

// Create creates a new order
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now
	order.SetKeys()

	av, err := attributevalue.MarshalMap(order)
	if err != nil {
		return errors.Internal("Failed to marshal order")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Order already exists")
		}
		return errors.Wrap(err, "Failed to create order")
	}

	// Write order number index for lookup
	if order.OrderNumber != "" {
		idx := &domain.OrderNumberIndex{OrderID: order.ID}
		idx.SetKeys(order.OrderNumber)
		idxAV, err := attributevalue.MarshalMap(idx)
		if err != nil {
			return errors.Internal("Failed to marshal order number index")
		}
		_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName:           aws.String(r.client.ordersTable),
			Item:                idxAV,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		})
		if err != nil && !isConditionalCheckFailed(err) {
			return errors.Wrap(err, "Failed to write order number index")
		}
	}

	return nil
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get order")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Order not found")
	}

	var order domain.Order
	if err := attributevalue.UnmarshalMap(result.Item, &order); err != nil {
		return nil, errors.Internal("Failed to unmarshal order")
	}

	return &order, nil
}

// GetByOrderNumber retrieves an order by order number
func (r *OrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	// Lookup order ID from index item
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER_NUMBER#" + orderNumber},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to lookup order number")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Order not found")
	}

	var index domain.OrderNumberIndex
	if err := attributevalue.UnmarshalMap(result.Item, &index); err != nil {
		return nil, errors.Internal("Failed to unmarshal order number index")
	}

	return r.GetByID(ctx, index.OrderID)
}

// Update updates an existing order
func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	order.UpdatedAt = time.Now()
	order.SetKeys()

	av, err := attributevalue.MarshalMap(order)
	if err != nil {
		return errors.Internal("Failed to marshal order")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Order not found")
		}
		return errors.Wrap(err, "Failed to update order")
	}

	return nil
}

// List retrieves orders with filters
func (r *OrderRepository) List(ctx context.Context, req domain.ListOrdersRequest) (*domain.ListOrdersResponse, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "ORDER#ALL"},
		},
		ScanIndexForward: aws.Bool(false), // newest first
	}

	// Build filter expressions — always filter by entity_type to exclude non-order records
	filters := []string{"entity_type = :et"}
	input.ExpressionAttributeValues[":et"] = &types.AttributeValueMemberS{Value: "ORDER"}
	exprAttrNames := map[string]string{}

	if req.Status != nil {
		filters = append(filters, "#status = :status")
		input.ExpressionAttributeValues[valStatus] = &types.AttributeValueMemberS{Value: string(*req.Status)}
		exprAttrNames[nameStatus] = attrStatus
	}

	if req.PaymentStatus != nil {
		filters = append(filters, "payment_status = :paymentStatus")
		input.ExpressionAttributeValues[":paymentStatus"] = &types.AttributeValueMemberS{Value: string(*req.PaymentStatus)}
	}

	if req.CustomerID != nil {
		filters = append(filters, "customer_id = :custID")
		input.ExpressionAttributeValues[":custID"] = &types.AttributeValueMemberS{Value: *req.CustomerID}
	}

	if req.Search != "" {
		filters = append(filters, "(contains(order_number, :search) OR contains(customer_name, :search))")
		input.ExpressionAttributeValues[":search"] = &types.AttributeValueMemberS{Value: req.Search}
	}

	input.FilterExpression = aws.String(strings.Join(filters, " AND "))
	if len(exprAttrNames) > 0 {
		input.ExpressionAttributeNames = exprAttrNames
	}

	orders, pg, err := QueryPage[domain.Order](ctx, r.client.db, input, req.PaginationRequest, "Failed to list orders")
	if err != nil {
		return nil, err
	}

	return &domain.ListOrdersResponse{
		Orders:     orders,
		Pagination: pg,
	}, nil
}

// GetByCustomer retrieves orders by customer ID
func (r *OrderRepository) GetByCustomer(ctx context.Context, customerID string, pagination domain.PaginationRequest) (*domain.ListOrdersResponse, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
		},
		ScanIndexForward: aws.Bool(false),
	}

	orders, pg, err := QueryPage[domain.Order](ctx, r.client.db, input, pagination, "Failed to query orders by customer")
	if err != nil {
		return nil, err
	}

	return &domain.ListOrdersResponse{
		Orders:     orders,
		Pagination: pg,
	}, nil
}

// ApplyRefundSettlement writes only what a settled refund owns: the lines' refunded
// quantities and the payment status. Targeted rather than a whole-item write, which
// would revert a status, tracking or note change made since the order was read.
//
// No optimistic lock needed. The caller recomputes both values from every completed
// refund, so two settlements racing here both write the same answer.
func (r *OrderRepository) ApplyRefundSettlement(ctx context.Context, id string, items []domain.OrderItem, paymentStatus domain.PaymentStatus) error {
	marshaledItems, err := attributevalue.Marshal(items)
	if err != nil {
		return errors.Internal("Failed to marshal order items")
	}

	// A partial settlement must never overwrite a full one: two settlements racing can
	// land in the opposite order to the totals they derived from.
	condition := "attribute_exists(PK)"
	values := map[string]types.AttributeValue{
		":items": marshaledItems,
		":ps":    &types.AttributeValueMemberS{Value: string(paymentStatus)},
		exprNow:  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}
	if paymentStatus != domain.PaymentStatusRefunded {
		condition += " AND payment_status <> :refunded"
		values[":refunded"] = &types.AttributeValueMemberS{Value: string(domain.PaymentStatusRefunded)}
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		// items is a DynamoDB reserved word, like status, so it has to be aliased.
		UpdateExpression:          aws.String("SET #items = :items, payment_status = :ps, updated_at = :now"),
		ExpressionAttributeNames:  map[string]string{"#items": "items"},
		ExpressionAttributeValues: values,
		ConditionExpression:       aws.String(condition),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			// Usually a settlement that already marked the order fully refunded, which is
			// not worth failing a webhook over. It also covers a missing order, so say so
			// rather than let that pass silently.
			slog.WarnContext(ctx, "Refund settlement was refused by its condition",
				"order_id", id, "payment_status", paymentStatus)
			return nil
		}
		return errors.Wrap(err, "Failed to apply refund settlement")
	}

	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus, updatedBy string) error {
	now := time.Now()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("SET #status = :status, updated_at = :now, updated_by = :by"),
		ExpressionAttributeNames: map[string]string{
			nameStatus: attrStatus,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			valStatus: &types.AttributeValueMemberS{Value: string(status)},
			exprNow:   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":by":     &types.AttributeValueMemberS{Value: updatedBy},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Order not found")
		}
		return errors.Wrap(err, "Failed to update order status")
	}

	return nil
}

// AddNote adds a note to an order
func (r *OrderRepository) AddNote(ctx context.Context, id string, note domain.OrderNote) error {
	now := time.Now()

	noteAV, err := attributevalue.MarshalMap(note)
	if err != nil {
		return errors.Internal("Failed to marshal note")
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		// Must be internal_notes — that is what Order.InternalNotes unmarshals
		// from. Writing to a bare `notes` attribute persisted the note where
		// nothing ever read it.
		UpdateExpression: aws.String("SET internal_notes = list_append(if_not_exists(internal_notes, :empty), :note), updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":note":  &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: noteAV}}},
			":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			exprNow:  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Order not found")
		}
		return errors.Wrap(err, "Failed to add note")
	}

	return nil
}

// UpdateTracking updates tracking information
func (r *OrderRepository) UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string, trackingURL string) error {
	now := time.Now()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("SET tracking_number = :tracking, shipping_carrier = :carrier, tracking_url = :url, updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tracking": &types.AttributeValueMemberS{Value: trackingNumber},
			":carrier":  &types.AttributeValueMemberS{Value: carrier},
			":url":      &types.AttributeValueMemberS{Value: trackingURL},
			exprNow:     &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Order not found")
		}
		return errors.Wrap(err, "Failed to update tracking")
	}

	return nil
}

var _ domain.OrderRepository = (*OrderRepository)(nil)
