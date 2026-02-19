package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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

	return nil
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
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
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "ORDER_NUMBER"},
			":sk": &types.AttributeValueMemberS{Value: orderNumber},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query order by number")
	}

	if len(result.Items) == 0 {
		return nil, errors.NotFound("Order not found")
	}

	var order domain.Order
	if err := attributevalue.UnmarshalMap(result.Items[0], &order); err != nil {
		return nil, errors.Internal("Failed to unmarshal order")
	}

	return &order, nil
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
	filterBuilder := expression.Name("entity_type").Equal(expression.Value("ORDER"))

	if req.Status != nil {
		filterBuilder = filterBuilder.And(expression.Name("status").Equal(expression.Value(*req.Status)))
	}

	if req.PaymentStatus != nil {
		filterBuilder = filterBuilder.And(expression.Name("payment_status").Equal(expression.Value(*req.PaymentStatus)))
	}

	if req.CustomerID != nil {
		filterBuilder = filterBuilder.And(expression.Name("customer_id").Equal(expression.Value(*req.CustomerID)))
	}

	if req.Search != "" {
		searchFilter := expression.Name("order_number").Contains(req.Search).
			Or(expression.Name("customer_name").Contains(req.Search))
		filterBuilder = filterBuilder.And(searchFilter)
	}

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	result, err := r.client.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.ordersTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
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
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
		},
		ScanIndexForward: aws.Bool(false), // Newest first
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query orders by customer")
	}

	var orders []*domain.Order
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &orders); err != nil {
		return nil, errors.Internal("Failed to unmarshal orders")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
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
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, updated_at = :now, updated_by = :by"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(status)},
			":now":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
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
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET notes = list_append(if_not_exists(notes, :empty), :note), updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":note":  &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: noteAV}}},
			":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			":now":   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
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
func (r *OrderRepository) UpdateTracking(ctx context.Context, id string, trackingNumber string, carrier string) error {
	now := time.Now()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "ORDER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET tracking_number = :tracking, shipping_carrier = :carrier, updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tracking": &types.AttributeValueMemberS{Value: trackingNumber},
			":carrier":  &types.AttributeValueMemberS{Value: carrier},
			":now":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
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

	return nil
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
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
			":pk": &types.AttributeValueMemberS{Value: "CUSTOMER_EMAIL"},
			":sk": &types.AttributeValueMemberS{Value: email},
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
	filterBuilder := expression.Name("entity_type").Equal(expression.Value("CUSTOMER"))

	if req.Search != "" {
		searchFilter := expression.Name("email").Contains(req.Search).
			Or(expression.Name("first_name").Contains(req.Search)).
			Or(expression.Name("last_name").Contains(req.Search)).
			Or(expression.Name("phone").Contains(req.Search))
		filterBuilder = filterBuilder.And(searchFilter)
	}

	expr, err := expression.NewBuilder().WithFilter(filterBuilder).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	result, err := r.client.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.ordersTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list customers")
	}

	var customers []*domain.Customer
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &customers); err != nil {
		return nil, errors.Internal("Failed to unmarshal customers")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
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
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
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

// helper function
func intToString(n int) string {
	return fmt.Sprintf("%d", n)
}

// Ensure interface compliance
var _ domain.OrderRepository = (*OrderRepository)(nil)
var _ domain.CustomerRepository = (*CustomerRepository)(nil)
