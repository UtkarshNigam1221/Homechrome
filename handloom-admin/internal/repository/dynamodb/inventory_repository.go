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
	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// InventoryRepository implements domain.InventoryRepository
type InventoryRepository struct {
	client *Client
}

// NewInventoryRepository creates a new InventoryRepository
func NewInventoryRepository(client *Client) *InventoryRepository {
	return &InventoryRepository{client: client}
}

// Create creates a new inventory record
func (r *InventoryRepository) Create(ctx context.Context, inventory *domain.Inventory) error {
	now := time.Now()
	inventory.CreatedAt = now
	inventory.UpdatedAt = now
	inventory.SetKeys()

	av, err := attributevalue.MarshalMap(inventory)
	if err != nil {
		return errors.Internal("Failed to marshal inventory")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Inventory record already exists")
		}
		return errors.Wrap(err, "Failed to create inventory")
	}

	return nil
}

// GetByProductID retrieves inventory by product ID
func (r *InventoryRepository) GetByProductID(ctx context.Context, productID string) (*domain.Inventory, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "INVENTORY#" + productID},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get inventory")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Inventory not found")
	}

	var inventory domain.Inventory
	if err := attributevalue.UnmarshalMap(result.Item, &inventory); err != nil {
		return nil, errors.Internal("Failed to unmarshal inventory")
	}

	return &inventory, nil
}

// Update updates an existing inventory record
func (r *InventoryRepository) Update(ctx context.Context, inventory *domain.Inventory) error {
	inventory.UpdatedAt = time.Now()
	inventory.SetKeys()

	av, err := attributevalue.MarshalMap(inventory)
	if err != nil {
		return errors.Internal("Failed to marshal inventory")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Inventory not found")
		}
		return errors.Wrap(err, "Failed to update inventory")
	}

	return nil
}

// AddStock adds stock to inventory
func (r *InventoryRepository) AddStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	return r.modifyStock(ctx, productID, quantity, domain.InventoryTransactionTypeAdd, reason, "", userID)
}

// RemoveStock removes stock from inventory
func (r *InventoryRepository) RemoveStock(ctx context.Context, productID string, quantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	return r.modifyStock(ctx, productID, -quantity, domain.InventoryTransactionTypeRemove, reason, "", userID)
}

// ReserveStock reserves stock for an order
func (r *InventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.modifyReserved(ctx, productID, quantity, domain.InventoryTransactionTypeReserve, "ORDER", orderID)
}

// ReleaseStock releases reserved stock
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	return r.modifyReserved(ctx, productID, -quantity, domain.InventoryTransactionTypeRelease, "ORDER", orderID)
}

// AdjustStock adjusts stock to a specific quantity
func (r *InventoryRepository) AdjustStock(ctx context.Context, productID string, newQuantity int, reason string, userID string) (*domain.InventoryTransaction, error) {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	delta := newQuantity - inventory.Quantity
	return r.modifyStock(ctx, productID, delta, domain.InventoryTransactionTypeAdjust, reason, "", userID)
}

// modifyStock modifies stock quantity
func (r *InventoryRepository) modifyStock(ctx context.Context, productID string, delta int, txnType domain.InventoryTransactionType, reason string, referenceID string, userID string) (*domain.InventoryTransaction, error) {
	// Get current inventory
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	previousQty := inventory.Quantity
	newQty := inventory.Quantity + delta

	if newQty < 0 {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Insufficient stock")
	}

	// Update inventory
	now := time.Now()
	updateExpr := "SET quantity = :qty, available_qty = :avail, updated_at = :now"
	exprValues := map[string]types.AttributeValue{
		":qty":   &types.AttributeValueMemberN{Value: intToString(newQty)},
		":avail": &types.AttributeValueMemberN{Value: intToString(newQty - inventory.ReservedQty)},
		":now":   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	if txnType == domain.InventoryTransactionTypeAdd {
		updateExpr += ", last_restock_at = :restock"
		exprValues[":restock"] = &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)}
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "INVENTORY#" + productID},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to update inventory")
	}

	// Create transaction record
	txn := &domain.InventoryTransaction{
		ID:            "inv_txn_" + uuid.New().String()[:8],
		ProductID:     productID,
		Type:          txnType,
		Quantity:      abs(delta),
		PreviousQty:   previousQty,
		NewQty:        newQty,
		Reason:        reason,
		ReferenceType: "USER",
		ReferenceID:   referenceID,
		CreatedAt:     now,
		CreatedBy:     userID,
	}
	txn.SetKeys()

	if err := r.createTransaction(ctx, txn); err != nil {
		// Log but don't fail
		return txn, nil
	}

	return txn, nil
}

// modifyReserved modifies reserved stock
func (r *InventoryRepository) modifyReserved(ctx context.Context, productID string, delta int, txnType domain.InventoryTransactionType, refType string, refID string) (*domain.InventoryTransaction, error) {
	// Get current inventory
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	previousReserved := inventory.ReservedQty
	newReserved := inventory.ReservedQty + delta

	if newReserved < 0 {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Cannot release more than reserved")
	}

	availableQty := inventory.Quantity - newReserved
	if availableQty < 0 {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Insufficient available stock")
	}

	// Update inventory
	now := time.Now()
	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "INVENTORY#" + productID},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET reserved_qty = :res, available_qty = :avail, updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":res":   &types.AttributeValueMemberN{Value: intToString(newReserved)},
			":avail": &types.AttributeValueMemberN{Value: intToString(availableQty)},
			":now":   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to update inventory")
	}

	// Create transaction record
	txn := &domain.InventoryTransaction{
		ID:            "inv_txn_" + uuid.New().String()[:8],
		ProductID:     productID,
		Type:          txnType,
		Quantity:      abs(delta),
		PreviousQty:   previousReserved,
		NewQty:        newReserved,
		Reason:        fmt.Sprintf("%s %s", refType, refID),
		ReferenceType: refType,
		ReferenceID:   refID,
		CreatedAt:     now,
	}
	txn.SetKeys()

	_ = r.createTransaction(ctx, txn)

	return txn, nil
}

// createTransaction creates an inventory transaction record
func (r *InventoryRepository) createTransaction(ctx context.Context, txn *domain.InventoryTransaction) error {
	av, err := attributevalue.MarshalMap(txn)
	if err != nil {
		return errors.Internal("Failed to marshal transaction")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.coreTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create transaction")
	}

	return nil
}

// GetTransactions retrieves inventory transactions
func (r *InventoryRepository) GetTransactions(ctx context.Context, productID string, pagination domain.PaginationRequest) (*domain.ListInventoryTransactionsResponse, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "INVENTORY#" + productID},
			":sk": &types.AttributeValueMemberS{Value: "TXN#"},
		},
		ScanIndexForward: aws.Bool(false), // Newest first
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get transactions")
	}

	var transactions []*domain.InventoryTransaction
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &transactions); err != nil {
		return nil, errors.Internal("Failed to unmarshal transactions")
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(transactions, pagination)

	return &domain.ListInventoryTransactionsResponse{
		Transactions: paged,
		Pagination:   pg,
	}, nil
}

// GetLowStockProducts retrieves products with low stock
func (r *InventoryRepository) GetLowStockProducts(ctx context.Context, pagination domain.PaginationRequest) (*domain.ListInventoryResponse, error) {
	filterExpr := expression.Name("entity_type").Equal(expression.Value("INVENTORY"))

	expr, err := expression.NewBuilder().WithFilter(filterExpr).Build()
	if err != nil {
		return nil, errors.Internal("Failed to build query expression")
	}

	result, err := r.client.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.client.coreTable),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list inventory")
	}

	var allInventories []*domain.Inventory
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &allInventories); err != nil {
		return nil, errors.Internal("Failed to unmarshal inventory")
	}

	// Filter for low stock
	var lowStockInventories []*domain.Inventory
	for _, inv := range allInventories {
		if inv.AvailableQty <= inv.LowStockThreshold {
			lowStockInventories = append(lowStockInventories, inv)
		}
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(lowStockInventories, pagination)

	return &domain.ListInventoryResponse{
		Inventories: paged,
		Pagination:  pg,
	}, nil
}

// DeleteByProductID deletes the inventory record and all its transactions for a product.
// Queries all items under PK = INVENTORY#<productID> and batch-deletes them.
func (r *InventoryRepository) DeleteByProductID(ctx context.Context, productID string) error {
	pk := "INVENTORY#" + productID

	// Query all items under this partition key (METADATA + TXN records)
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
		ProjectionExpression: aws.String("PK, SK"),
	})
	if err != nil {
		return errors.Wrap(err, "Failed to query inventory items for deletion")
	}

	if len(result.Items) == 0 {
		return nil // Nothing to delete
	}

	// Batch delete in chunks of 25 (DynamoDB BatchWriteItem limit)
	for i := 0; i < len(result.Items); i += 25 {
		end := i + 25
		if end > len(result.Items) {
			end = len(result.Items)
		}

		var writeRequests []types.WriteRequest
		for _, item := range result.Items[i:end] {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"PK": item["PK"],
						"SK": item["SK"],
					},
				},
			})
		}

		_, err := r.client.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				r.client.coreTable: writeRequests,
			},
		})
		if err != nil {
			return errors.Wrap(err, "Failed to batch delete inventory items")
		}
	}

	return nil
}

// abs returns absolute value of int
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Ensure interface compliance
var _ domain.InventoryRepository = (*InventoryRepository)(nil)
