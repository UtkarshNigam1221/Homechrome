package dynamodb

import (
	"context"
	"strconv"
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

// CouponRepository implements domain.CouponRepository
type CouponRepository struct {
	client *Client
}

// NewCouponRepository creates a new CouponRepository
func NewCouponRepository(client *Client) *CouponRepository {
	return &CouponRepository{
		client: client,
	}
}

// couponCodeIndex points a code at the coupon that currently holds it. A separate item
// rather than a GSI because only a conditional put can make a code unique, and only a
// real item is strongly consistent — a GSI is neither.
type couponCodeIndex struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	EntityType string `dynamodbav:"entity_type"`
	CouponID   string `dynamodbav:"coupon_id"`
}

func marshalCouponCodeIndex(code, couponID string) (map[string]types.AttributeValue, error) {
	idx := couponCodeIndex{
		PK:         "CODE#" + strings.ToUpper(code),
		SK:         skMetadata,
		EntityType: "COUPON_CODE_INDEX",
		CouponID:   couponID,
	}
	av, err := attributevalue.MarshalMap(idx)
	if err != nil {
		return nil, errors.Internal("Failed to marshal coupon code index")
	}
	return av, nil
}

// Create writes the coupon and its code pointer in one transaction. Written separately,
// a failure on the second leaves a coupon GetByCode cannot find.
func (r *CouponRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	now := time.Now()
	coupon.CreatedAt = now
	coupon.UpdatedAt = now
	coupon.Code = strings.ToUpper(coupon.Code)
	coupon.SetKeys()

	av, err := attributevalue.MarshalMap(coupon)
	if err != nil {
		return errors.Internal("Failed to marshal coupon")
	}

	codeAV, err := marshalCouponCodeIndex(coupon.Code, coupon.ID)
	if err != nil {
		return err
	}

	if _, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{
				TableName:           aws.String(r.client.couponsTable),
				Item:                av,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &types.Put{
				TableName:           aws.String(r.client.couponsTable),
				Item:                codeAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
		},
	}); err != nil {
		if isTransactionCanceled(err) {
			// Either the id is taken or the code is. Both mean pick another code.
			return errors.New(errors.ErrCodeAlreadyExists, "Coupon code already exists")
		}
		return errors.Wrap(err, "Failed to create coupon")
	}

	return nil
}

// GetByID retrieves a coupon by ID
func (r *CouponRepository) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUPON#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get coupon")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Coupon")
	}

	var coupon domain.Coupon
	if err := attributevalue.UnmarshalMap(result.Item, &coupon); err != nil {
		return nil, errors.Internal("Failed to unmarshal coupon")
	}

	return &coupon, nil
}

// GetByCode resolves a code through the pointer, then reads the coupon. Two O(1) reads,
// both strongly consistent, and no partition holds more than one code.
func (r *CouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CODE#" + strings.ToUpper(strings.TrimSpace(code))},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to look up coupon code")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Coupon")
	}

	var idx couponCodeIndex
	if err := attributevalue.UnmarshalMap(result.Item, &idx); err != nil {
		return nil, errors.Internal("Failed to unmarshal coupon code index")
	}

	return r.GetByID(ctx, idx.CouponID)
}

// Update writes only the fields an admin edit may change.
//
// It is an UpdateItem, not a PutItem, because usage_count must survive it. CouponService
// reads the coupon, applies the request and writes it back; a whole-item put carries the
// count it read moments earlier, so any ADD usage_count the PhonePe webhook landed in
// between was silently erased. The racing writer is a payment webhook, not a second
// operator, so this is not the single-operator case. Not naming usage_count here is what
// makes the two writers disjoint, the same way IncrementUsage only ever touches it.
//
// Also absent: code, type, value, audience, customer_id and everything created-at-write
// time. None of them is reachable through UpdateCouponRequest, and the code in particular
// must not move without repointing CODE#<code> (see CouponService.Update's guard).
func (r *CouponRepository) Update(ctx context.Context, coupon *domain.Coupon) error {
	coupon.UpdatedAt = time.Now()
	coupon.SetKeys()

	update := expression.
		Set(expression.Name("name"), expression.Value(coupon.Name)).
		Set(expression.Name("min_order_value"), expression.Value(coupon.MinOrderValue)).
		Set(expression.Name("usage_limit"), expression.Value(coupon.UsageLimit)).
		Set(expression.Name("usage_per_user"), expression.Value(coupon.UsagePerUser)).
		Set(expression.Name("combines_with_offers"), expression.Value(coupon.CombinesWithOffers)).
		Set(expression.Name("valid_from"), expression.Value(coupon.ValidFrom)).
		Set(expression.Name("status"), expression.Value(coupon.Status)).
		Set(expression.Name("updated_at"), expression.Value(coupon.UpdatedAt)).
		// SetKeys derives these four. An audience change moves the GSI1 partition and a
		// rename moves search_key, so they have to travel with the edit or the coupon
		// stays indexed under its old life — findable by its old name, not its new one.
		// GSI1SK is created_at, so it is recomputed identically rather than moved.
		Set(expression.Name("GSI1PK"), expression.Value(coupon.GSI1PK)).
		Set(expression.Name("GSI1SK"), expression.Value(coupon.GSI1SK)).
		Set(expression.Name("search_key"), expression.Value(coupon.SearchKey)).
		Set(expression.Name("entity_type"), expression.Value(coupon.EntityType))

	// These four carry dynamodbav omitempty, so Create leaves them out of the item
	// entirely. Clearing one has to remove the attribute rather than store a zero, or an
	// edited coupon and a fresh one end up with different shapes for the same state.
	update = setOrRemove(update, "description", coupon.Description, coupon.Description == "")
	update = setOrRemove(update, "max_discount", coupon.MaxDiscount, coupon.MaxDiscount == 0)
	update = setOrRemove(update, "updated_by", coupon.UpdatedBy, coupon.UpdatedBy == "")
	update = setOrRemove(update, "valid_until", coupon.ValidUntil, coupon.ValidUntil == nil)

	expr, err := expression.NewBuilder().
		WithUpdate(update).
		WithCondition(expression.AttributeExists(expression.Name("PK"))).
		Build()
	if err != nil {
		return errors.Internal("Failed to build the coupon update")
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUPON#" + coupon.ID},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Coupon")
		}
		return errors.Wrap(err, "Failed to update coupon")
	}

	return nil
}

// setOrRemove sets an attribute, or removes it when the value is the empty one that
// dynamodbav omitempty would have dropped on a fresh write.
func setOrRemove(
	u expression.UpdateBuilder, attr string, value interface{}, empty bool,
) expression.UpdateBuilder {
	if empty {
		return u.Remove(expression.Name(attr))
	}
	return u.Set(expression.Name(attr), expression.Value(value))
}

// Delete removes the coupon and its code pointer in one transaction. Removing only
// COUPON#<id> leaves CODE#<code> behind, and Create writes that pointer under
// attribute_not_exists(PK) — so the code would be refused forever, for a coupon that no
// longer exists.
//
// The code is read here rather than passed down from the service: the pointer item is
// this repository's own invention, so nothing above it should have to know it needs
// cleaning up. GetByID already returns NotFound, which keeps Delete's error contract.
func (r *CouponRepository) Delete(ctx context.Context, id string) error {
	coupon, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if _, err = r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Delete: &types.Delete{
				TableName: aws.String(r.client.couponsTable),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: "COUPON#" + id},
					"SK": &types.AttributeValueMemberS{Value: skMetadata},
				},
				ConditionExpression: aws.String("attribute_exists(PK)"),
			}},
			{Delete: &types.Delete{
				TableName: aws.String(r.client.couponsTable),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: "CODE#" + strings.ToUpper(coupon.Code)},
					"SK": &types.AttributeValueMemberS{Value: skMetadata},
				},
				// An absent pointer is fine — nothing to clean up. One pointing at a
				// different coupon is not ours to remove.
				ConditionExpression: aws.String("attribute_not_exists(PK) OR coupon_id = :id"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":id": &types.AttributeValueMemberS{Value: id},
				},
			}},
		},
	}); err != nil {
		if isTransactionCanceled(err) {
			return errors.NotFound("Coupon")
		}
		return errors.Wrap(err, "Failed to delete coupon")
	}

	return nil
}

// ListPublic reads the advertisable coupons, dropping any expiring before cutoff.
// Status and audience filter in DynamoDB; the validity window is checked in Go.
func (r *CouponRepository) ListPublic(ctx context.Context, cutoff time.Time) ([]*domain.Coupon, error) {
	all, err := QueryAll[domain.Coupon](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		FilterExpression:       aws.String("#status = :status AND #audience = :audience"),
		ExpressionAttributeNames: map[string]string{
			nameStatus: attrStatus,
			// Named rather than inline: cheaper than being wrong about the reserved list.
			"#audience": "audience",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:      &types.AttributeValueMemberS{Value: "COUPON#ALL"},
			valStatus:   &types.AttributeValueMemberS{Value: string(domain.CouponStatusActive)},
			":audience": &types.AttributeValueMemberS{Value: string(domain.AudienceAll)},
		},
		ScanIndexForward: aws.Bool(false),
	}, "Failed to list public coupons")
	if err != nil {
		return nil, err
	}

	// valid_until marshals as RFC3339Nano, which trims trailing zeros, so the stored
	// string is variable-width and compares wrong inside one second. Filtered here.
	now := time.Now()
	live := make([]*domain.Coupon, 0, len(all))
	for _, c := range all {
		if now.Before(c.ValidFrom) {
			continue
		}
		if c.ValidUntil != nil && c.ValidUntil.Before(cutoff) {
			continue
		}
		live = append(live, c)
	}
	return live, nil
}

// List reads the public listing partition. Personal codes carry a different GSI1PK, so
// they are absent by construction rather than by filter.
//
// Reads the partition whole because the filters and sort below are applied in Go — a
// DynamoDB cursor would only order the page it returned, not the whole list. Correct
// while public coupons number in the dozens; past a few hundred, status belongs in the
// partition key.
func (r *CouponRepository) List(ctx context.Context, req domain.ListCouponsRequest) (*domain.ListCouponsResponse, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK: &types.AttributeValueMemberS{Value: "COUPON#ALL"},
		},
		// GSI1SK is created_at, so descending is newest-first — the order the admin
		// list wants, out of the index rather than out of a sort in Go.
		ScanIndexForward: aws.Bool(false),
	}

	// Filters run in DynamoDB. They are applied after the read, so a page can come
	// back short — QueryPage re-queries for exactly the shortfall, which is why the
	// cursor stays exact and the caller still gets a full page.
	var filters []string
	names := map[string]string{}

	if req.Status != nil {
		filters = append(filters, "#status = :status")
		names[nameStatus] = attrStatus
		input.ExpressionAttributeValues[valStatus] = &types.AttributeValueMemberS{Value: string(*req.Status)}
	}
	if req.Type != nil {
		// "type" is reserved in DynamoDB expressions.
		filters = append(filters, "#type = :type")
		names["#type"] = "type"
		input.ExpressionAttributeValues[":type"] = &types.AttributeValueMemberS{Value: string(*req.Type)}
	}
	if req.IsActive != nil {
		// Active is a status, not a column: asking for inactive means anything but.
		op := "="
		if !*req.IsActive {
			op = "<>"
		}
		filters = append(filters, "#status "+op+" :activeStatus")
		names[nameStatus] = attrStatus
		input.ExpressionAttributeValues[":activeStatus"] = &types.AttributeValueMemberS{
			Value: string(domain.CouponStatusActive),
		}
	}
	if req.Search != "" {
		// search_key is lower(code + " " + name) — see Coupon.SetKeys. contains() is
		// case-sensitive, so the needle is lowered to match.
		filters = append(filters, "contains(search_key, :search)")
		input.ExpressionAttributeValues[":search"] = &types.AttributeValueMemberS{
			Value: strings.ToLower(req.Search),
		}
	}

	if len(filters) > 0 {
		input.FilterExpression = aws.String(strings.Join(filters, " AND "))
	}
	if len(names) > 0 {
		input.ExpressionAttributeNames = names
	}

	coupons, pagination, err := QueryPage[domain.Coupon](
		ctx, r.client.db, input, req.PaginationRequest, "Failed to list coupons")
	if err != nil {
		return nil, err
	}

	return &domain.ListCouponsResponse{
		Coupons:    coupons,
		Pagination: pagination,
	}, nil
}

// RecordUsage records coupon usage
func (r *CouponRepository) RecordUsage(ctx context.Context, usage *domain.CouponUsage) error {
	usage.SetKeys()

	av, err := attributevalue.MarshalMap(usage)
	if err != nil {
		return errors.Internal("Failed to marshal coupon usage")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.couponsTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to record coupon usage")
	}

	return nil
}

// IncrementUsage claims one redemption if the coupon has one left. The condition and the
// increment are one operation, so two simultaneous claims on a last-remaining code cannot
// both succeed. A condition failure means exhausted — an outcome, not an error.
func (r *CouponRepository) IncrementUsage(ctx context.Context, couponID string) (bool, error) {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUPON#" + couponID},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("ADD usage_count :one SET updated_at = :now"),
		// usage_limit 0 means unlimited. attribute_exists guards against claiming a
		// coupon that has been deleted since validation.
		ConditionExpression: aws.String(
			"attribute_exists(PK) AND (usage_limit = :zero OR usage_count < usage_limit)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprOne: &types.AttributeValueMemberN{Value: "1"},
			":zero": &types.AttributeValueMemberN{Value: "0"},
			":now":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "Failed to claim a coupon redemption")
	}
	return true, nil
}

// GetCustomerUsage reads one counter. O(1), against the O(redemptions) scan this replaces.
func (r *CouponRepository) GetCustomerUsage(ctx context.Context, customerID, couponID string) (int, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: "USE#" + couponID},
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read coupon usage")
	}
	if result.Item == nil {
		return 0, nil // never used is zero, not missing
	}

	var counter domain.CouponUseCounter
	if err := attributevalue.UnmarshalMap(result.Item, &counter); err != nil {
		return 0, errors.Internal("Failed to unmarshal coupon usage counter")
	}
	return counter.Count, nil
}

// GetCustomerUsageAll reads the whole CUSTOMER#<id> counter partition in one query.
// Keys come from the item's coupon_id, which IncrementCustomerUsage always sets.
func (r *CouponRepository) GetCustomerUsageAll(
	ctx context.Context, customerID string,
) (map[string]int, error) {
	counters, err := QueryAll[domain.CouponUseCounter](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:    &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
			":prefix": &types.AttributeValueMemberS{Value: "USE#"},
		},
	}, "Failed to read coupon usage")
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(counters))
	for _, c := range counters {
		counts[c.CouponID] = c.Count
	}
	return counts, nil
}

// IncrementCustomerUsage claims one of this customer's allowance, creating the counter
// on first use. Conditional, like IncrementUsage: an unconditional ADD made
// usage_per_user advisory only, and because redemptions are counted at payment success
// the bypass window was the whole initiate-to-payment interval rather than a race. A
// customer could initiate twice, pay both, and a limit of 1 would end at 2.
//
// "One per customer" is a stronger promise than "500 total", so unlike the global limit
// this overshoot is not accepted — the order still keeps its quoted price, but the
// counter stays truthful and the caller can see it happened.
func (r *CouponRepository) IncrementCustomerUsage(
	ctx context.Context, customerID, couponID string, limit int,
) (bool, error) {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.couponsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
			"SK": &types.AttributeValueMemberS{Value: "USE#" + couponID},
		},
		UpdateExpression: aws.String(
			"ADD #count :one SET entity_type = :et, customer_id = :cust, coupon_id = :coupon"),
		ExpressionAttributeNames: map[string]string{"#count": "count"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprOne:   &types.AttributeValueMemberN{Value: "1"},
			":et":     &types.AttributeValueMemberS{Value: "COUPON_USE_COUNTER"},
			":cust":   &types.AttributeValueMemberS{Value: customerID},
			":coupon": &types.AttributeValueMemberS{Value: couponID},
		},
	}
	// 0 is unlimited, so there is nothing to condition on. The counter still moves —
	// the audit value of knowing how often a customer used a code does not depend on
	// there being a cap.
	if limit > 0 {
		input.ConditionExpression = aws.String("attribute_not_exists(#count) OR #count < :limit")
		input.ExpressionAttributeValues[":limit"] =
			&types.AttributeValueMemberN{Value: strconv.Itoa(limit)}
	}

	if _, err := r.client.db.UpdateItem(ctx, input); err != nil {
		if isConditionalCheckFailed(err) {
			return false, nil // spent, not broken
		}
		return false, errors.Wrap(err, "Failed to record coupon usage for the customer")
	}
	return true, nil
}

// Ensure interface compliance
var _ domain.CouponRepository = (*CouponRepository)(nil)
