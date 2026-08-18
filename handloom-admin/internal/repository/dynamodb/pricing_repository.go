package dynamodb

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// PricingRuleRepository implements domain.PricingRuleRepository
type PricingRuleRepository struct {
	client *Client
}

// NewPricingRuleRepository creates a new PricingRuleRepository
func NewPricingRuleRepository(client *Client) *PricingRuleRepository {
	return &PricingRuleRepository{client: client}
}

// Create creates a new pricing rule
func (r *PricingRuleRepository) Create(ctx context.Context, rule *domain.PricingRule) error {
	rule.SetKeys()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(rule)
	if err != nil {
		return errors.Internal(err)
	}

	condition := expression.AttributeNotExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(r.client.coreTable),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.Conflict("Pricing rule already exists")
		}
		return errors.Internal(err)
	}

	return nil
}

// GetByID retrieves a pricing rule by ID
func (r *PricingRuleRepository) GetByID(ctx context.Context, id string) (*domain.PricingRule, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRICING_RULE#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodePricingRuleNotFound, "Pricing rule not found")
	}

	var rule domain.PricingRule
	if err := attributevalue.UnmarshalMap(result.Item, &rule); err != nil {
		return nil, errors.Internal(err)
	}

	return &rule, nil
}

// Update updates an existing pricing rule
func (r *PricingRuleRepository) Update(ctx context.Context, rule *domain.PricingRule) error {
	rule.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(rule)
	if err != nil {
		return errors.Internal(err)
	}

	condition := expression.AttributeExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(r.client.coreTable),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.New(errors.ErrCodePricingRuleNotFound, "Pricing rule not found")
		}
		return errors.Internal(err)
	}

	return nil
}

// Delete deletes a pricing rule by ID
func (r *PricingRuleRepository) Delete(ctx context.Context, id string) error {
	condition := expression.AttributeExists(expression.Name("PK"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PRICING_RULE#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if stderrors.As(err, &ccf) {
			return errors.New(errors.ErrCodePricingRuleNotFound, "Pricing rule not found")
		}
		return errors.Internal(err)
	}

	return nil
}

// List retrieves pricing rules with filters
func (r *PricingRuleRepository) List(ctx context.Context, req domain.ListPricingRulesRequest) (*domain.ListPricingRulesResponse, error) {
	req.Limit = DefaultLimit(req.Limit)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.coreTable),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "PRICING_RULE#ALL"},
		},
		ScanIndexForward: aws.Bool(true),
	}

	// Build filter expressions
	var filters []string
	exprAttrNames := map[string]string{}

	if req.ScopeType != nil {
		filters = append(filters, "scope_type = :scopeType")
		input.ExpressionAttributeValues[":scopeType"] = &types.AttributeValueMemberS{Value: string(*req.ScopeType)}
	}
	if req.CategoryID != nil {
		filters = append(filters, "category_id = :catID")
		input.ExpressionAttributeValues[":catID"] = &types.AttributeValueMemberS{Value: *req.CategoryID}
	}
	if req.PricingType != nil {
		filters = append(filters, "pricing_type = :pricingType")
		input.ExpressionAttributeValues[":pricingType"] = &types.AttributeValueMemberS{Value: string(*req.PricingType)}
	}
	if req.IsActive != nil {
		filters = append(filters, "is_active = :isActive")
		input.ExpressionAttributeValues[":isActive"] = &types.AttributeValueMemberBOOL{Value: *req.IsActive}
	}

	if len(filters) > 0 {
		filterExpr := strings.Join(filters, " AND ")
		input.FilterExpression = aws.String(filterExpr)
	}
	if len(exprAttrNames) > 0 {
		input.ExpressionAttributeNames = exprAttrNames
	}

	rules, err := QueryAll[domain.PricingRule](ctx, r.client.db, input, "Failed to list pricing rules")
	if err != nil {
		return nil, err
	}

	// Filter by search if provided
	if req.Search != "" {
		var filtered []*domain.PricingRule
		for _, rule := range rules {
			if containsIgnoreCase(rule.Name, req.Search) {
				filtered = append(filtered, rule)
			}
		}
		rules = filtered
	}

	// Paginated in memory because the search above is applied in Go: a cursor
	// would page the unfiltered index and hand back short, uneven pages.
	paged, pg := InMemoryPaginate(rules, req.PaginationRequest)

	return &domain.ListPricingRulesResponse{
		Rules:      paged,
		Pagination: pg,
	}, nil
}

// GetByScope retrieves pricing rules by scope
func (r *PricingRuleRepository) GetByScope(ctx context.Context, scopeType domain.PricingRuleScope, scopeID string) ([]*domain.PricingRule, error) {
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("SCOPE#" + string(scopeType)))
	if scopeID != "" {
		keyExpr = keyExpr.And(expression.Key("GSI1SK").Equal(expression.Value(scopeID)))
	}

	filterExpr := expression.Name("is_active").Equal(expression.Value(true))

	expr, err := expression.NewBuilder().
		WithKeyCondition(keyExpr).
		WithFilter(filterExpr).
		Build()
	if err != nil {
		return nil, errors.Internal(err)
	}

	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(r.client.coreTable),
		IndexName:                 aws.String("GSI1"),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	var rules []*domain.PricingRule
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &rules); err != nil {
		return nil, errors.Internal(err)
	}

	return rules, nil
}

// GetApplicableRules retrieves all applicable rules for a category/product
func (r *PricingRuleRepository) GetApplicableRules(ctx context.Context, categoryID string, productID *string, material *string) ([]*domain.PricingRule, error) {
	var allRules []*domain.PricingRule

	// Get global rules
	globalRules, err := r.GetByScope(ctx, domain.PricingRuleScopeGlobal, "")
	if err != nil {
		return nil, err
	}
	allRules = append(allRules, globalRules...)

	// Get category rules
	categoryRules, err := r.GetByScope(ctx, domain.PricingRuleScopeCategory, categoryID)
	if err != nil {
		return nil, err
	}
	allRules = append(allRules, categoryRules...)

	// Get product rules if product ID provided
	if productID != nil {
		productRules, err := r.GetByScope(ctx, domain.PricingRuleScopeProduct, *productID)
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, productRules...)
	}

	// Get material rules if material provided
	if material != nil {
		materialRules, err := r.GetByScope(ctx, domain.PricingRuleScopeMaterial, *material)
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, materialRules...)
	}

	// Filter by validity period
	now := time.Now()
	var validRules []*domain.PricingRule
	for _, rule := range allRules {
		if rule.ValidFrom != nil && rule.ValidFrom.After(now) {
			continue
		}
		if rule.ValidUntil != nil && rule.ValidUntil.Before(now) {
			continue
		}
		validRules = append(validRules, rule)
	}

	return validRules, nil
}

// GetGlobalRule retrieves the global fallback rule
func (r *PricingRuleRepository) GetGlobalRule(ctx context.Context) (*domain.PricingRule, error) {
	rules, err := r.GetByScope(ctx, domain.PricingRuleScopeGlobal, "GLOBAL")
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, errors.New(errors.ErrCodePricingRuleNotFound, "Global pricing rule not found")
	}

	// Return highest priority
	var best *domain.PricingRule
	for _, rule := range rules {
		if best == nil || rule.Priority > best.Priority {
			best = rule
		}
	}

	return best, nil
}

// Ensure interface compliance
var _ domain.PricingRuleRepository = (*PricingRuleRepository)(nil)

// ==================== PRICE QUOTE REPOSITORY ====================

// PriceQuoteRepository implements domain.PriceQuoteRepository
type PriceQuoteRepository struct {
	client *Client
}

// NewPriceQuoteRepository creates a new PriceQuoteRepository
func NewPriceQuoteRepository(client *Client) *PriceQuoteRepository {
	return &PriceQuoteRepository{client: client}
}

// Create creates a new price quote
func (r *PriceQuoteRepository) Create(ctx context.Context, quote *domain.PriceQuote) error {
	quote.SetKeys()
	// Set TTL (24 hours from now)
	quote.TTL = quote.ValidUntil.Unix()

	item, err := attributevalue.MarshalMap(quote)
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.ordersTable),
		Item:      item,
	})
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

// GetByID retrieves a price quote by ID
func (r *PriceQuoteRepository) GetByID(ctx context.Context, id string) (*domain.PriceQuote, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "QUOTE#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Internal(err)
	}

	if result.Item == nil {
		return nil, errors.New(errors.ErrCodeQuoteNotFound, "Price quote not found")
	}

	var quote domain.PriceQuote
	if err := attributevalue.UnmarshalMap(result.Item, &quote); err != nil {
		return nil, errors.Internal(err)
	}

	return &quote, nil
}

// MarkAsUsed marks a quote as used in an order
func (r *PriceQuoteRepository) MarkAsUsed(ctx context.Context, id string, orderID string) error {
	update := expression.Set(expression.Name("used_in_order"), expression.Value(orderID))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return errors.Internal(err)
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "QUOTE#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
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

// Ensure interface compliance
var _ domain.PriceQuoteRepository = (*PriceQuoteRepository)(nil)
