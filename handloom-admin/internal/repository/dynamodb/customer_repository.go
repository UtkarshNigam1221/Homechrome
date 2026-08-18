// Package dynamodb provides DynamoDB repository implementations
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

	// Write the email pointer. Unlike the phone index this refuses a duplicate:
	// two customers sharing an address made GetByEmail return whichever the
	// index happened to yield.
	if customer.Email != "" {
		if err := r.putEmailIndex(ctx, customer.Email, customer.ID); err != nil {
			return err
		}
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
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER_EMAIL#" + email},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to look up customer email")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Customer not found")
	}

	var idx domain.CustomerEmailIndex
	if err := attributevalue.UnmarshalMap(result.Item, &idx); err != nil {
		return nil, errors.Internal("Failed to unmarshal email index")
	}

	return r.GetByID(ctx, idx.CustomerID)
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

	// The pointer is a separate item, so an address change has to move it.
	// Claim the new one first: if it belongs to somebody else the update is
	// refused before the customer record has been touched.
	existing, err := r.GetByID(ctx, customer.ID)
	if err != nil {
		return err
	}
	if existing.Email != customer.Email {
		if customer.Email != "" {
			if claimErr := r.putEmailIndex(ctx, customer.Email, customer.ID); claimErr != nil {
				return claimErr
			}
		}
		if existing.Email != "" {
			if releaseErr := r.deleteEmailIndex(ctx, existing.Email); releaseErr != nil {
				return releaseErr
			}
		}
	}

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

// putEmailIndex claims an address for a customer, refusing one already taken.
func (r *CustomerRepository) putEmailIndex(ctx context.Context, email, customerID string) error {
	idx := &domain.CustomerEmailIndex{CustomerID: customerID}
	idx.SetKeys(email)

	av, err := attributevalue.MarshalMap(idx)
	if err != nil {
		return errors.Internal("Failed to marshal email index")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK) OR customer_id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: customerID},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Email already in use")
		}
		return errors.Wrap(err, "Failed to write email index")
	}
	return nil
}

// deleteEmailIndex releases an address the customer no longer uses.
func (r *CustomerRepository) deleteEmailIndex(ctx context.Context, email string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER_EMAIL#" + email},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to remove email index")
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

	customers, pg, err := QueryPage[domain.Customer](ctx, r.client.db, input, req.Pagination, "Failed to list customers")
	if err != nil {
		return nil, err
	}

	paged := customers

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

var _ domain.CustomerRepository = (*CustomerRepository)(nil)
