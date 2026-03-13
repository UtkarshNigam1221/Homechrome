package dynamodb

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CartRepository implements domain.CartRepository
type CartRepository struct {
	client *Client
}

// NewCartRepository creates a new CartRepository
func NewCartRepository(client *Client) *CartRepository {
	return &CartRepository{client: client}
}

// GetCart retrieves a cart with all its items by querying the partition key.
// Returns cart header (SK=METADATA) + items (SK begins_with ITEM#).
// If no header is found, returns an empty cart.
func (r *CartRepository) GetCart(ctx context.Context, cartPK string) (*domain.CartWithItems, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: cartPK},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query cart")
	}

	cartWithItems := &domain.CartWithItems{
		Items: []domain.CartItem{},
	}

	for _, item := range result.Items {
		// Extract SK to determine if this is the header or an item
		skAttr, ok := item["SK"]
		if !ok {
			continue
		}
		sk, ok := skAttr.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}

		if sk.Value == "METADATA" {
			var cart domain.Cart
			if err := attributevalue.UnmarshalMap(item, &cart); err != nil {
				return nil, errors.Internal("Failed to unmarshal cart header")
			}
			cartWithItems.Cart = &cart
		} else if strings.HasPrefix(sk.Value, "ITEM#") {
			var cartItem domain.CartItem
			if err := attributevalue.UnmarshalMap(item, &cartItem); err != nil {
				return nil, errors.Internal("Failed to unmarshal cart item")
			}
			cartWithItems.Items = append(cartWithItems.Items, cartItem)
		}
	}

	// If no header found, return an empty cart
	if cartWithItems.Cart == nil {
		cartWithItems.Cart = &domain.Cart{
			PK:         cartPK,
			SK:         "METADATA",
			EntityType: "CART",
			Currency:   "INR",
			UpdatedAt:  time.Now(),
		}
	}

	return cartWithItems, nil
}

// PutCartItem writes a cart item to the orders table
func (r *CartRepository) PutCartItem(ctx context.Context, item *domain.CartItem) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return errors.Internal("Failed to marshal cart item")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.ordersTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to put cart item")
	}

	return nil
}

// UpdateCartItem updates the quantity and total price of a cart item
func (r *CartRepository) UpdateCartItem(ctx context.Context, cartPK, productID string, quantity int, totalPrice int64) error {
	ttl := time.Now().Add(30 * 24 * time.Hour).Unix()

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: cartPK},
			"SK": &types.AttributeValueMemberS{Value: "ITEM#" + productID},
		},
		UpdateExpression: aws.String("SET quantity = :qty, total_price = :tp, #ttl = :ttl"),
		ExpressionAttributeNames: map[string]string{
			"#ttl": "ttl",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":qty": &types.AttributeValueMemberN{Value: intToString(quantity)},
			":tp":  &types.AttributeValueMemberN{Value: intToString(int(totalPrice))},
			":ttl": &types.AttributeValueMemberN{Value: intToString(int(ttl))},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Cart item not found")
		}
		return errors.Wrap(err, "Failed to update cart item")
	}

	return nil
}

// DeleteCartItem removes a single item from the cart
func (r *CartRepository) DeleteCartItem(ctx context.Context, cartPK, productID string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: cartPK},
			"SK": &types.AttributeValueMemberS{Value: "ITEM#" + productID},
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to delete cart item")
	}

	return nil
}

// UpdateCartHeader writes/replaces the cart header (full put)
func (r *CartRepository) UpdateCartHeader(ctx context.Context, cart *domain.Cart) error {
	av, err := attributevalue.MarshalMap(cart)
	if err != nil {
		return errors.Internal("Failed to marshal cart header")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.ordersTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to update cart header")
	}

	return nil
}

// ClearCart removes all items and the header for a cart.
// It queries all records for the cart PK and batch-deletes them.
func (r *CartRepository) ClearCart(ctx context.Context, cartPK string) error {
	// Query all items for this cart
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: cartPK},
		},
		ProjectionExpression: aws.String("PK, SK"),
	})
	if err != nil {
		return errors.Wrap(err, "Failed to query cart for clearing")
	}

	if len(result.Items) == 0 {
		return nil
	}

	// Extract PK/SK keys for batch deletion
	keys := make([]map[string]types.AttributeValue, 0, len(result.Items))
	for _, item := range result.Items {
		keys = append(keys, map[string]types.AttributeValue{
			"PK": item["PK"],
			"SK": item["SK"],
		})
	}

	if err := batchDeleteKeys(ctx, r.client.db, r.client.ordersTable, keys); err != nil {
		return errors.Wrap(err, "Failed to batch delete cart items")
	}

	return nil
}

// Ensure interface compliance
var _ domain.CartRepository = (*CartRepository)(nil)
