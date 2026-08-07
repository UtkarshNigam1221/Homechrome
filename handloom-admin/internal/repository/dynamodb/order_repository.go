package dynamodb

import (
	"context"
	"strconv"
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
		input.ExpressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: string(*req.Status)}
		exprAttrNames["#status"] = attrStatus
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

	result, err := r.client.db.Query(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list orders")
	}

	var orders []*domain.Order
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &orders); err != nil {
		return nil, errors.Internal("Failed to unmarshal orders")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(orders, req.PaginationRequest)

	return &domain.ListOrdersResponse{
		Orders:     paged,
		Pagination: pg,
	}, nil
}

// GetByCustomer retrieves orders by customer ID
func (r *OrderRepository) GetByCustomer(ctx context.Context, customerID string, pagination domain.PaginationRequest) (*domain.ListOrdersResponse, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query orders by customer")
	}

	var orders []*domain.Order
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &orders); err != nil {
		return nil, errors.Internal("Failed to unmarshal orders")
	}

	paged, pg := InMemoryPaginate(orders, pagination)

	return &domain.ListOrdersResponse{
		Orders:     paged,
		Pagination: pg,
	}, nil
}

// UpdateStatus updates order status
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
			"#status": attrStatus,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(status)},
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

// CustomerRepository implements domain.CustomerRepository
type CustomerRepository struct {
	client *Client
}

// NewCustomerRepository creates a new CustomerRepository
func NewCustomerRepository(client *Client) *CustomerRepository {
	return &CustomerRepository{client: client}
}

// Create creates a new customer
func (r *CustomerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now
	customer.SetKeys()

	av, err := attributevalue.MarshalMap(customer)
	if err != nil {
		return errors.Internal("Failed to marshal customer")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Customer already exists")
		}
		return errors.Wrap(err, "Failed to create customer")
	}

	// Write phone index for lookup
	if customer.Phone != "" {
		idx := &domain.CustomerPhoneIndex{CustomerID: customer.ID}
		idx.SetKeys(customer.Phone)
		idxAV, err := attributevalue.MarshalMap(idx)
		if err != nil {
			return errors.Internal("Failed to marshal phone index")
		}
		_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName:           aws.String(r.client.ordersTable),
			Item:                idxAV,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		})
		if err != nil && !isConditionalCheckFailed(err) {
			return errors.Wrap(err, "Failed to write phone index")
		}
	}

	return nil
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get customer")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Customer not found")
	}

	var customer domain.Customer
	if err := attributevalue.UnmarshalMap(result.Item, &customer); err != nil {
		return nil, errors.Internal("Failed to unmarshal customer")
	}

	return &customer, nil
}

// GetByEmail retrieves a customer by email
func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "CUSTOMER_EMAIL"},
			exprSK: &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query customer by email")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("Customer not found")
	}

	var customer domain.Customer
	if err := attributevalue.UnmarshalMap(result.Items[0], &customer); err != nil {
		return nil, errors.Internal("Failed to unmarshal customer")
	}

	return &customer, nil
}

// GetByPhone retrieves a customer by phone number using the phone index
func (r *CustomerRepository) GetByPhone(ctx context.Context, phone string) (*domain.Customer, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER_PHONE#" + phone},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to lookup customer by phone")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Customer not found")
	}
	var index domain.CustomerPhoneIndex
	if err := attributevalue.UnmarshalMap(result.Item, &index); err != nil {
		return nil, errors.Internal("Failed to unmarshal phone index")
	}
	return r.GetByID(ctx, index.CustomerID)
}

// Update updates an existing customer
func (r *CustomerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	customer.UpdatedAt = time.Now()
	customer.SetKeys()

	av, err := attributevalue.MarshalMap(customer)
	if err != nil {
		return errors.Internal("Failed to marshal customer")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Customer not found")
		}
		return errors.Wrap(err, "Failed to update customer")
	}

	return nil
}

// List retrieves customers with filters
func (r *CustomerRepository) List(ctx context.Context, req domain.ListCustomersRequest) (*domain.ListCustomersResponse, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "CUSTOMER#ALL"},
		},
		ScanIndexForward: aws.Bool(false),
	}

	if req.Search != "" {
		input.FilterExpression = aws.String("contains(email, :search) OR contains(first_name, :search) OR contains(last_name, :search) OR contains(phone, :search)")
		input.ExpressionAttributeValues[":search"] = &types.AttributeValueMemberS{Value: req.Search}
	}

	result, err := r.client.db.Query(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list customers")
	}

	var customers []*domain.Customer
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &customers); err != nil {
		return nil, errors.Internal("Failed to unmarshal customers")
	}

	paged, pg := InMemoryPaginate(customers, req.Pagination)

	return &domain.ListCustomersResponse{
		Customers:  paged,
		Pagination: pg,
	}, nil
}

// Delete deletes a customer by ID
func (r *CustomerRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Customer not found")
		}
		return errors.Wrap(err, "Failed to delete customer")
	}
	return nil
}

// Search searches customers by query
func (r *CustomerRepository) Search(ctx context.Context, query string, pagination domain.PaginationRequest) (*domain.ListCustomersResponse, error) {
	req := domain.ListCustomersRequest{
		Search:     query,
		Pagination: pagination,
	}
	return r.List(ctx, req)
}

// RecordPurchase atomically bumps the customer's OrderCount by 1 and adds
// amountPaise to TotalSpent, returning the new count. DynamoDB ADD initializes
// an attribute to 0 when it does not yet exist, so the very first call always
// returns 1. Using ReturnValues=UPDATED_NEW means callers can gate
// first-purchase logic on newCount==1 without a separate read, closing the
// concurrent-payment race. Both counters move in the same UpdateItem so they
// cannot diverge.
func (r *CustomerRepository) RecordPurchase(ctx context.Context, customerID string, amountPaise int64) (int64, error) {
	out, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("ADD order_count :one, total_spent :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one":    &types.AttributeValueMemberN{Value: "1"},
			":amount": &types.AttributeValueMemberN{Value: strconv.FormatInt(amountPaise, 10)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, errors.Wrap(err, "Failed to record customer purchase")
	}
	raw, ok := out.Attributes["order_count"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, errors.Internal("order_count missing from UpdateItem response")
	}
	n, err := strconv.ParseInt(raw.Value, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse order_count")
	}
	return n, nil
}

// Ensure interface compliance
var _ domain.OrderRepository = (*OrderRepository)(nil)
var _ domain.CustomerRepository = (*CustomerRepository)(nil)
