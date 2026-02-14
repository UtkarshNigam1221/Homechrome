package dynamodb

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// ProductRepository implements domain.ProductRepository
type ProductRepository struct {
	client *Client
}

// NewProductRepository creates a new ProductRepository
func NewProductRepository(client *Client) *ProductRepository {
	return &ProductRepository{client: client}
}

// GetByID retrieves a product by ID
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodeProductNotFound, "Product not found")
	}

	var product domain.Product
	if err := attributevalue.UnmarshalMap(result.Item, &product); err != nil {
		return nil, errors.Internal(err)
	}

	return &product, nil
}

// GetBySKU retrieves a product by SKU using the SKU uniqueness item.
func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "SKU#" + sku},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodeProductNotFound, "Product not found")
	}

	// Extract product_id from the SKU index item, then fetch the full product
	var productID string
	if pidAttr, ok := result.Item["product_id"]; ok {
		if s, ok := pidAttr.(*types.AttributeValueMemberS); ok {
			productID = s.Value
		}
	}
	if productID == "" {
		return nil, errors.Internal("SKU index missing product_id")
	}

	return r.GetByID(ctx, productID)
}

// Delete deletes a product by ID
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeProductNotFound, "Product not found")
		}
		return errors.Internal(err)
	}

	return nil
}

// List retrieves products with filters using GSI queries (no Scan).
// When CategoryID is set, queries GSI1 (CATEGORY#<id>) for that category's products.
// When CategoryID is nil, queries GSI2 (PRODUCT#ALL) for all products.
func (r *ProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	limit := DefaultLimit(req.Limit)

	if req.CategoryID != nil {
		return r.listByCategory(ctx, *req.CategoryID, limit, req.Cursor, req)
	}
	return r.listAllProducts(ctx, limit, req.Cursor, req)
}

// GetByCategory retrieves products by category ID with cursor-based pagination.
func (r *ProductRepository) GetByCategory(ctx context.Context, categoryID string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	return r.listByCategory(ctx, categoryID, DefaultLimit(pagination.Limit), pagination.Cursor, domain.ListProductsRequest{})
}

// listByCategory queries products for a single category using GSI1 with cursor pagination.
func (r *ProductRepository) listByCategory(ctx context.Context, categoryID string, limit int, cursor string, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("CATEGORY#" + categoryID)).
		And(expression.Key("GSI1SK").BeginsWith("PRODUCT#"))

	return r.queryProducts(ctx, "GSI1", keyExpr, func(p *domain.Product) map[string]types.AttributeValue {
		return map[string]types.AttributeValue{
			"PK":     &types.AttributeValueMemberS{Value: p.PK},
			"SK":     &types.AttributeValueMemberS{Value: p.SK},
			"GSI1PK": &types.AttributeValueMemberS{Value: p.GSI1PK},
			"GSI1SK": &types.AttributeValueMemberS{Value: p.GSI1SK},
		}
	}, limit, cursor, req)
}

// listAllProducts queries all products using GSI2 (GSI2PK = "PRODUCT#ALL") with cursor pagination.
func (r *ProductRepository) listAllProducts(ctx context.Context, limit int, cursor string, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	keyExpr := expression.Key("GSI2PK").Equal(expression.Value("PRODUCT#ALL"))

	return r.queryProducts(ctx, "GSI2", keyExpr, func(p *domain.Product) map[string]types.AttributeValue {
		return map[string]types.AttributeValue{
			"PK":     &types.AttributeValueMemberS{Value: p.PK},
			"SK":     &types.AttributeValueMemberS{Value: p.SK},
			"GSI2PK": &types.AttributeValueMemberS{Value: p.GSI2PK},
			"GSI2SK": &types.AttributeValueMemberS{Value: p.GSI2SK},
		}
	}, limit, cursor, req)
}

// queryProducts executes a paginated GSI query for products with optional filters.
func (r *ProductRepository) queryProducts(
	ctx context.Context,
	indexName string,
	keyExpr expression.KeyConditionBuilder,
	buildCursorKeys func(*domain.Product) map[string]types.AttributeValue,
	limit int,
	cursor string,
	req domain.ListProductsRequest,
) (*domain.ListProductsResponse, error) {
	exclusiveStartKey, err := DecodeCursor(cursor)
	if err != nil {
		return nil, err
	}

	builder := expression.NewBuilder().WithKeyCondition(keyExpr)

	hasFilter := false
	filterExpr := r.buildProductFilterExpr(req, &hasFilter)
	if hasFilter {
		builder = builder.WithFilter(filterExpr)
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, errors.Internal(err)
	}

	fetchLimit := int32(limit)
	if hasFilter {
		fetchLimit = int32(limit * 3)
		if fetchLimit > 300 {
			fetchLimit = 300
		}
	}

	var collected []*domain.Product
	var lastEvaluatedKey map[string]types.AttributeValue
	currentStartKey := exclusiveStartKey

	for {
		queryInput := &dynamodb.QueryInput{
			TableName:                 aws.String(r.client.coreTable),
			IndexName:                 aws.String(indexName),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			Limit:                     aws.Int32(fetchLimit),
			ExclusiveStartKey:         currentStartKey,
		}
		if expr.Filter() != nil {
			queryInput.FilterExpression = expr.Filter()
		}

		result, err := r.client.db.Query(ctx, queryInput)
		if err != nil {
			return nil, errors.Internal(err)
		}

		var batch []*domain.Product
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &batch); err != nil {
			return nil, errors.Internal(err)
		}

		collected = append(collected, batch...)
		lastEvaluatedKey = result.LastEvaluatedKey

		if len(collected) >= limit || lastEvaluatedKey == nil {
			break
		}
		currentStartKey = lastEvaluatedKey
	}

	if len(collected) > limit {
		collected = collected[:limit]
		lastEvaluatedKey = buildCursorKeys(collected[limit-1])
	}

	return &domain.ListProductsResponse{
		Products:   collected,
		Pagination: BuildPaginationResponse(limit, lastEvaluatedKey),
	}, nil
}

// buildProductFilterExpr builds a DynamoDB filter expression from ListProductsRequest fields.
func (r *ProductRepository) buildProductFilterExpr(req domain.ListProductsRequest, hasFilter *bool) expression.ConditionBuilder {
	var filterExpr expression.ConditionBuilder

	addFilter := func(cond expression.ConditionBuilder) {
		if *hasFilter {
			filterExpr = filterExpr.And(cond)
		} else {
			filterExpr = cond
			*hasFilter = true
		}
	}

	if req.Status != nil {
		addFilter(expression.Name("status").Equal(expression.Value(string(*req.Status))))
	}
	if req.Material != nil {
		addFilter(expression.Name("material").Equal(expression.Value(*req.Material)))
	}
	if req.Color != nil {
		addFilter(expression.Name("color").Equal(expression.Value(*req.Color)))
	}
	if req.MinPrice != nil {
		addFilter(expression.Name("selling_price").GreaterThanEqual(expression.Value(*req.MinPrice)))
	}
	if req.MaxPrice != nil {
		addFilter(expression.Name("selling_price").LessThanEqual(expression.Value(*req.MaxPrice)))
	}
	if req.InStock != nil && *req.InStock {
		addFilter(expression.Name("available_qty").GreaterThan(expression.Value(0)))
	}
	if req.LowStock != nil && *req.LowStock {
		addFilter(expression.Name("available_qty").LessThanEqual(expression.Name("low_stock_threshold")))
	}
	if req.Search != "" {
		addFilter(
			expression.Name("name").Contains(req.Search).
				Or(expression.Name("sku").Contains(req.Search)),
		)
	}

	return filterExpr
}

// UpdateInventory updates the inventory fields
func (r *ProductRepository) UpdateInventory(ctx context.Context, id string, quantity, reservedQty, availableQty int) error {
	update := expression.Set(
		expression.Name("quantity"), expression.Value(quantity),
	).Set(
		expression.Name("reserved_qty"), expression.Value(reservedQty),
	).Set(
		expression.Name("available_qty"), expression.Value(availableQty),
	).Set(
		expression.Name("updated_at"), expression.Value(time.Now()),
	)

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

// BatchGetByIDs retrieves multiple products by IDs
func (r *ProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return []*domain.Product{}, nil
	}

	keys := make([]map[string]types.AttributeValue, len(ids))
	for i, id := range ids {
		keys[i] = map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		}
	}

	result, err := r.client.db.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			r.client.coreTable: {Keys: keys},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	var products []*domain.Product
	if err := attributevalue.UnmarshalListOfMaps(result.Responses[r.client.coreTable], &products); err != nil {
		return nil, errors.Internal(err)
	}

	return products, nil
}

// CreateWithAttributeIndexes creates a product with its searchable attribute indexes in a transaction.
// If inventory is non-nil it is included in the same transaction.
func (r *ProductRepository) CreateWithAttributeIndexes(ctx context.Context, product *domain.Product, searchableAttrs map[string][]string, inventory *domain.Inventory) error {
	product.SetKeys()
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	if product.Currency == "" {
		product.Currency = "INR"
	}

	// Build transaction items
	var transactItems []types.TransactWriteItem

	// 1. Main product item
	productAV, err := attributevalue.MarshalMap(product)
	if err != nil {
		return errors.Internal(err)
	}

	transactItems = append(transactItems, types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(r.client.coreTable),
			Item:                productAV,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		},
	})

	// 2. SKU uniqueness item
	skuItem := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: "SKU#" + product.SKU},
		"SK":          &types.AttributeValueMemberS{Value: "METADATA"},
		"product_id":  &types.AttributeValueMemberS{Value: product.ID},
		"entity_type": &types.AttributeValueMemberS{Value: "PRODUCT_SKU"},
	}
	transactItems = append(transactItems, types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(r.client.coreTable),
			Item:                skuItem,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		},
	})

	// 3. Inventory item (if provided)
	if inventory != nil {
		inventory.SetKeys()
		inventory.CreatedAt = time.Now()
		inventory.UpdatedAt = time.Now()

		invAV, err := attributevalue.MarshalMap(inventory)
		if err != nil {
			return errors.Internal(err)
		}

		transactItems = append(transactItems, types.TransactWriteItem{
			Put: &types.Put{
				TableName: aws.String(r.client.coreTable),
				Item:      invAV,
			},
		})
	}

	// 4. Attribute index items (only for searchable attributes)
	for attrName, values := range searchableAttrs {
		for _, value := range values {
			if value == "" {
				continue
			}
			index := &domain.ProductAttributeIndex{
				ProductID:  product.ID,
				CategoryID: product.CategoryID,
				AttrName:   attrName,
				AttrValue:  value,
			}
			index.SetKeys()

			indexAV, err := attributevalue.MarshalMap(index)
			if err != nil {
				return errors.Internal(err)
			}

			transactItems = append(transactItems, types.TransactWriteItem{
				Put: &types.Put{
					TableName: aws.String(r.client.coreTable),
					Item:      indexAV,
				},
			})
		}
	}

	// DynamoDB transaction limit is 100 items
	if len(transactItems) > 100 {
		return errors.New(errors.ErrCodeValidation, "Too many attributes to index in a single transaction")
	}

	_, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		var ccf *types.TransactionCanceledException
		if stderrors.As(err, &ccf) {
			for i, reason := range ccf.CancellationReasons {
				if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
					if i == 1 {
						return errors.Conflict("Product with this SKU already exists")
					}
					return errors.Conflict("Product already exists")
				}
			}
		}
		return errors.Internal(err)
	}

	return nil
}

// UpdateWithAttributeIndexes updates a product and syncs its attribute indexes
func (r *ProductRepository) UpdateWithAttributeIndexes(ctx context.Context, product *domain.Product, oldSearchableAttrs, newSearchableAttrs map[string][]string) error {
	product.UpdatedAt = time.Now()
	product.SetKeys()

	var transactItems []types.TransactWriteItem

	// 1. Update product
	productAV, err := attributevalue.MarshalMap(product)
	if err != nil {
		return errors.Internal(err)
	}

	transactItems = append(transactItems, types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(r.client.coreTable),
			Item:                productAV,
			ConditionExpression: aws.String("attribute_exists(PK)"),
		},
	})

	// 2. Delete old attribute indexes that are no longer valid
	for attrName, oldValues := range oldSearchableAttrs {
		newValues := newSearchableAttrs[attrName]
		newValuesSet := make(map[string]bool)
		for _, v := range newValues {
			newValuesSet[v] = true
		}

		for _, oldValue := range oldValues {
			if !newValuesSet[oldValue] {
				// This value is being removed
				transactItems = append(transactItems, types.TransactWriteItem{
					Delete: &types.Delete{
						TableName: aws.String(r.client.coreTable),
						Key: map[string]types.AttributeValue{
							"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + product.ID},
							"SK": &types.AttributeValueMemberS{Value: "ATTR#" + attrName + "#" + oldValue},
						},
					},
				})
			}
		}
	}

	// 3. Add new attribute indexes
	for attrName, newValues := range newSearchableAttrs {
		oldValues := oldSearchableAttrs[attrName]
		oldValuesSet := make(map[string]bool)
		for _, v := range oldValues {
			oldValuesSet[v] = true
		}

		for _, newValue := range newValues {
			if newValue == "" {
				continue
			}
			if !oldValuesSet[newValue] {
				// This is a new value
				index := &domain.ProductAttributeIndex{
					ProductID:  product.ID,
					CategoryID: product.CategoryID,
					AttrName:   attrName,
					AttrValue:  newValue,
				}
				index.SetKeys()

				indexAV, err := attributevalue.MarshalMap(index)
				if err != nil {
					return errors.Internal(err)
				}

				transactItems = append(transactItems, types.TransactWriteItem{
					Put: &types.Put{
						TableName: aws.String(r.client.coreTable),
						Item:      indexAV,
					},
				})
			}
		}
	}

	if len(transactItems) > 100 {
		return errors.New(errors.ErrCodeValidation, "Too many attribute changes in a single transaction")
	}

	_, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

// DeleteWithAttributeIndexes deletes a product, its SKU uniqueness item, and its attribute indexes
func (r *ProductRepository) DeleteWithAttributeIndexes(ctx context.Context, id string, sku string, searchableAttrs map[string][]string) error {
	var transactItems []types.TransactWriteItem

	// 1. Delete product
	transactItems = append(transactItems, types.TransactWriteItem{
		Delete: &types.Delete{
			TableName: aws.String(r.client.coreTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
			ConditionExpression: aws.String("attribute_exists(PK)"),
		},
	})

	// 2. Delete SKU uniqueness item
	transactItems = append(transactItems, types.TransactWriteItem{
		Delete: &types.Delete{
			TableName: aws.String(r.client.coreTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "SKU#" + sku},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
		},
	})

	// 3. Delete all attribute indexes
	for attrName, values := range searchableAttrs {
		for _, value := range values {
			if value == "" {
				continue
			}
			transactItems = append(transactItems, types.TransactWriteItem{
				Delete: &types.Delete{
					TableName: aws.String(r.client.coreTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: "PRODUCT#" + id},
						"SK": &types.AttributeValueMemberS{Value: "ATTR#" + attrName + "#" + value},
					},
				},
			})
		}
	}

	if len(transactItems) > 100 {
		// If too many items, batch delete the indexes separately
		_, err := r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: transactItems[:1], // Just delete the product first
		})
		if err != nil {
			return errors.Internal(err)
		}

		// Then delete indexes in batches (non-transactional, best effort)
		for _, item := range transactItems[1:] {
			_, _ = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: item.Delete.TableName,
				Key:       item.Delete.Key,
			})
		}
		return nil
	}

	_, err := r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		var ccf *types.TransactionCanceledException
		if stderrors.As(err, &ccf) {
			for _, reason := range ccf.CancellationReasons {
				if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
					return errors.New(errors.ErrCodeProductNotFound, "Product not found")
				}
			}
		}
		return errors.Internal(err)
	}

	return nil
}

// FilterByAttributes retrieves products by category and attribute filters
func (r *ProductRepository) FilterByAttributes(ctx context.Context, categoryID string, filters map[string][]string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	if len(filters) == 0 {
		// No filters, just get by category
		return r.GetByCategory(ctx, categoryID, pagination)
	}

	// Query each filter attribute and intersect results
	var resultSets []map[string]bool

	for attrName, values := range filters {
		productIDsSet := make(map[string]bool)

		// Query for each value (OR logic within same attribute)
		for _, value := range values {
			result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
				TableName:              aws.String(r.client.coreTable),
				IndexName:              aws.String("GSI1"),
				KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :sk)"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":pk": &types.AttributeValueMemberS{Value: "ATTR#" + categoryID + "#" + attrName},
					":sk": &types.AttributeValueMemberS{Value: value + "#PRODUCT#"},
				},
				ProjectionExpression: aws.String("product_id"),
			})
			if err != nil {
				return nil, errors.Internal(err)
			}

			for _, item := range result.Items {
				if pidAttr, ok := item["product_id"]; ok {
					if pidStr, ok := pidAttr.(*types.AttributeValueMemberS); ok {
						productIDsSet[pidStr.Value] = true
					}
				}
			}
		}

		resultSets = append(resultSets, productIDsSet)
	}

	// AND logic: intersect all result sets
	if len(resultSets) == 0 {
		return &domain.ListProductsResponse{
			Products:   []*domain.Product{},
			Pagination: domain.PaginationResponse{Limit: DefaultLimit(pagination.Limit)},
		}, nil
	}

	// Start with first set and intersect with others
	finalSet := resultSets[0]
	for i := 1; i < len(resultSets); i++ {
		newSet := make(map[string]bool)
		for id := range finalSet {
			if resultSets[i][id] {
				newSet[id] = true
			}
		}
		finalSet = newSet
	}

	// Convert to slice
	var productIDs []string
	for id := range finalSet {
		productIDs = append(productIDs, id)
	}

	if len(productIDs) == 0 {
		return &domain.ListProductsResponse{
			Products:   []*domain.Product{},
			Pagination: domain.PaginationResponse{Limit: DefaultLimit(pagination.Limit)},
		}, nil
	}

	// Batch get product details
	products, err := r.BatchGetByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	// TODO: migrate to real DynamoDB cursor-based pagination
	paged, pg := InMemoryPaginate(products, pagination)

	return &domain.ListProductsResponse{
		Products:   paged,
		Pagination: pg,
	}, nil
}

// attrFieldPrefix is prepended to attribute names to form top-level DynamoDB field names
// e.g. attribute "color" is stored as field "attr_color" (a String Set).
const attrFieldPrefix = "attr_"

// AddAttributeValues adds values to the stored distinct value sets for a category.
// Uses a single DynamoDB UpdateItem with ADD on top-level String Set fields.
// ADD on a String Set merges new values atomically and creates the item if it doesn't exist.
// No read required — pure single write.
func (r *ProductRepository) AddAttributeValues(ctx context.Context, categoryID string, attrValues map[string][]string) error {
	if len(attrValues) == 0 {
		return nil
	}

	pk := "CATEGORY#" + categoryID
	sk := "ATTR_VALUES"

	// SET entity_type and category_id (idempotent metadata)
	setParts := "SET entity_type = :et, category_id = :cid"
	exprNames := map[string]string{}
	exprValues := map[string]types.AttributeValue{
		":et":  &types.AttributeValueMemberS{Value: "CATEGORY_ATTR_VALUES"},
		":cid": &types.AttributeValueMemberS{Value: categoryID},
	}

	// ADD each attribute's values as a top-level String Set field
	// e.g. ADD #attr_color :v_color, #attr_type :v_type
	var addParts []string
	for attrName, values := range attrValues {
		if len(values) == 0 {
			continue
		}
		ss := make([]string, 0, len(values))
		for _, v := range values {
			if v != "" {
				ss = append(ss, v)
			}
		}
		if len(ss) == 0 {
			continue
		}

		fieldName := attrFieldPrefix + attrName
		nameAlias := "#f_" + attrName
		valueAlias := ":v_" + attrName
		exprNames[nameAlias] = fieldName
		exprValues[valueAlias] = &types.AttributeValueMemberSS{Value: ss}
		addParts = append(addParts, nameAlias+" "+valueAlias)
	}

	if len(addParts) == 0 {
		return nil
	}

	// Build: SET entity_type = :et, category_id = :cid ADD #attr_color :v_color, #attr_type :v_type
	addExpr := "ADD "
	for j, part := range addParts {
		if j > 0 {
			addExpr += ", "
		}
		addExpr += part
	}
	fullExpr := setParts + " " + addExpr

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          aws.String(fullExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
	})
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

// GetAttributeValues reads the stored distinct value sets for a category.
// Each searchable attribute is stored as a top-level String Set field named "attr_<name>".
func (r *ProductRepository) GetAttributeValues(ctx context.Context, categoryID string) (map[string][]string, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CATEGORY#" + categoryID},
			"SK": &types.AttributeValueMemberS{Value: "ATTR_VALUES"},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return make(map[string][]string), nil
	}

	out := make(map[string][]string)
	for fieldName, av := range result.Item {
		if len(fieldName) <= len(attrFieldPrefix) {
			continue
		}
		if fieldName[:len(attrFieldPrefix)] != attrFieldPrefix {
			continue
		}
		attrName := fieldName[len(attrFieldPrefix):]
		if ss, ok := av.(*types.AttributeValueMemberSS); ok {
			out[attrName] = ss.Value
		}
	}

	return out, nil
}

// Ensure interface compliance
var _ domain.ProductRepository = (*ProductRepository)(nil)
