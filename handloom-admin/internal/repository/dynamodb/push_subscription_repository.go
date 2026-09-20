package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// broadcastHistoryMaxLimit caps one page of broadcast history, so a bad limit
// cannot turn the admin console into a full-partition read.
const broadcastHistoryMaxLimit int32 = 100

// PushSubscriptionRepository implements domain.PushSubscriptionRepository
type PushSubscriptionRepository struct {
	client *Client
}

// NewPushSubscriptionRepository creates a new PushSubscriptionRepository
func NewPushSubscriptionRepository(client *Client) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{client: client}
}

// subKey builds the primary key for an endpoint's subscription item.
func subKey(endpoint string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "PUSH_SUB#" + domain.PushEndpointID(endpoint)},
		"SK": &types.AttributeValueMemberS{Value: skMetadata},
	}
}

// custPointerKey addresses the row that lets a customer's devices be found
// without a second GSI on a table whose GSI1 is already spent on status.
func custPointerKey(customerID, subID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "PUSH_CUST#" + customerID},
		"SK": &types.AttributeValueMemberS{Value: "PUSH_SUB#" + subID},
	}
}

// Save upserts a subscription keyed by its endpoint hash. Re-subscribing the
// same browser refreshes the keys and reactivates the row rather than adding a
// duplicate, so CreatedAt carries over from the existing item.
//
// The read-then-write is deliberately not atomic: the only thing racing
// callers can disagree on is isNew, and a duplicate welcome push collapses in
// the browser anyway because both carry the same notification tag.
func (r *PushSubscriptionRepository) Save(ctx context.Context, sub *domain.PushSubscription) (bool, error) {
	existing, err := r.GetByEndpoint(ctx, sub.Endpoint)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}

	isNew := existing == nil
	if !isNew {
		sub.CreatedAt = existing.CreatedAt
	}

	sub.ID = domain.PushEndpointID(sub.Endpoint)
	sub.SetKeys()

	av, err := attributevalue.MarshalMap(sub)
	if err != nil {
		return false, errors.Internal("Failed to marshal push subscription")
	}

	writes := []types.TransactWriteItem{
		{Put: &types.Put{TableName: aws.String(r.client.notificationsTable), Item: av}},
	}

	// The subscription and its pointer must land together, or ListByCustomer
	// permanently misses a device the caller believes is linked.
	if sub.CustomerID != "" {
		pointer := custPointerKey(sub.CustomerID, sub.ID)
		pointer["endpoint"] = &types.AttributeValueMemberS{Value: sub.Endpoint}
		pointer["entity_type"] = &types.AttributeValueMemberS{Value: "PUSH_SUB_CUSTOMER"}
		writes = append(writes, types.TransactWriteItem{
			Put: &types.Put{TableName: aws.String(r.client.notificationsTable), Item: pointer},
		})
	}

	// Whoever held this device before must stop being notified for it — an
	// unlink (sub.CustomerID == "") is as much a hand-off as re-linking.
	if existing != nil && existing.CustomerID != "" && existing.CustomerID != sub.CustomerID {
		writes = append(writes, types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String(r.client.notificationsTable),
				Key:       custPointerKey(existing.CustomerID, sub.ID),
			},
		})
	}

	if len(writes) == 1 {
		if _, err := r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(r.client.notificationsTable),
			Item:      av,
		}); err != nil {
			return false, errors.Wrap(err, "Failed to save push subscription")
		}
		return isNew, nil
	}

	if _, err := r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: writes,
	}); err != nil {
		return false, errors.Wrap(err, "Failed to save push subscription")
	}

	return isNew, nil
}

// GetByEndpoint retrieves a subscription by endpoint URL.
func (r *PushSubscriptionRepository) GetByEndpoint(ctx context.Context, endpoint string) (*domain.PushSubscription, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.notificationsTable),
		Key:       subKey(endpoint),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get push subscription")
	}
	if result.Item == nil {
		return nil, errors.NotFound("Push subscription")
	}

	var sub domain.PushSubscription
	if err := attributevalue.UnmarshalMap(result.Item, &sub); err != nil {
		return nil, errors.Internal("Failed to unmarshal push subscription")
	}
	return &sub, nil
}

// Deactivate marks an endpoint INACTIVE, moving it out of the ACTIVE GSI1
// partition so broadcasts stop targeting it. Unsubscribing an endpoint that was
// never stored is a no-op: the caller's intent (stop sending here) already holds.
func (r *PushSubscriptionRepository) Deactivate(ctx context.Context, endpoint string) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           aws.String(r.client.notificationsTable),
		Key:                 subKey(endpoint),
		UpdateExpression:    aws.String("SET #s = :status, GSI1PK = :gsi1pk, last_seen_at = :now"),
		ConditionExpression: aws.String("attribute_exists(PK)"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(domain.PushSubscriptionInactive)},
			":gsi1pk": &types.AttributeValueMemberS{Value: "PUSH_SUB#" + string(domain.PushSubscriptionInactive)},
			":now":    &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339Nano)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return nil
		}
		return errors.Wrap(err, "Failed to deactivate push subscription")
	}
	return nil
}

// statusQuery builds the GSI1 query for one subscription status.
func (r *PushSubscriptionRepository) statusQuery(status domain.PushSubscriptionStatus) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName:              aws.String(r.client.notificationsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "PUSH_SUB#" + string(status)},
		},
		ScanIndexForward: aws.Bool(false),
	}
}

// ListActive retrieves every ACTIVE subscription. A broadcast must reach every
// live endpoint, so this reads the partition to the end rather than one page.
func (r *PushSubscriptionRepository) ListActive(ctx context.Context) ([]*domain.PushSubscription, error) {
	return QueryAll[domain.PushSubscription](
		ctx, r.client.db,
		r.statusQuery(domain.PushSubscriptionActive),
		"Failed to list active push subscriptions",
	)
}

// ListByCustomer queries the pointer rows, then reads each subscription — a
// customer has few enough devices that the round trips beat a write-heavy GSI.
func (r *PushSubscriptionRepository) ListByCustomer(
	ctx context.Context, customerID string,
) ([]*domain.PushSubscription, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.notificationsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "PUSH_CUST#" + customerID},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list a customer's push subscriptions")
	}

	subs := make([]*domain.PushSubscription, 0, len(result.Items))
	for _, item := range result.Items {
		endpoint, ok := item["endpoint"].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		sub, err := r.GetByEndpoint(ctx, endpoint.Value)
		if err != nil {
			if errors.IsNotFound(err) {
				// The pointer outlived its subscription; the device is just gone.
				continue
			}
			return nil, err
		}
		if sub.Status == domain.PushSubscriptionActive {
			subs = append(subs, sub)
		}
	}
	return subs, nil
}

// LinkCustomer sets customer_id on the subscription and writes the pointer row
// that ListByCustomer reads. A device that has gone away links to nothing,
// which is the same outcome the caller wanted.
func (r *PushSubscriptionRepository) LinkCustomer(
	ctx context.Context, endpoint, customerID string,
) error {
	sub, err := r.GetByEndpoint(ctx, endpoint)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	sub.CustomerID = customerID
	if _, err := r.Save(ctx, sub); err != nil {
		return err
	}
	return nil
}

// UnlinkCustomer gives a device back to nobody, clearing customer_id and the
// pointer in one write so a signed-out shopper's next order update cannot
// reach a browser somebody else is now holding.
func (r *PushSubscriptionRepository) UnlinkCustomer(ctx context.Context, endpoint string) error {
	sub, err := r.GetByEndpoint(ctx, endpoint)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if sub.CustomerID == "" {
		return nil
	}

	sub.CustomerID = ""
	_, err = r.Save(ctx, sub)
	return err
}

// List retrieves subscriptions of one status, newest first.
func (r *PushSubscriptionRepository) List(
	ctx context.Context,
	status domain.PushSubscriptionStatus,
	pagination domain.PaginationRequest,
) (*domain.ListPushSubscriptionsResponse, error) {
	subs, page, err := QueryPage[domain.PushSubscription](
		ctx, r.client.db,
		r.statusQuery(status),
		pagination,
		"Failed to list push subscriptions",
	)
	if err != nil {
		return nil, err
	}

	if subs == nil {
		subs = []*domain.PushSubscription{}
	}

	return &domain.ListPushSubscriptionsResponse{
		Subscriptions: subs,
		Pagination:    page,
	}, nil
}

// SaveBroadcast records the outcome of a fan-out.
func (r *PushSubscriptionRepository) SaveBroadcast(ctx context.Context, broadcast *domain.PushBroadcast) error {
	broadcast.SetKeys()

	av, err := attributevalue.MarshalMap(broadcast)
	if err != nil {
		return errors.Internal("Failed to marshal push broadcast")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.notificationsTable),
		Item:      av,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to save push broadcast")
	}
	return nil
}

// ListBroadcasts retrieves broadcast history, newest first.
func (r *PushSubscriptionRepository) ListBroadcasts(ctx context.Context, limit int32) ([]*domain.PushBroadcast, error) {
	if limit <= 0 || limit > broadcastHistoryMaxLimit {
		limit = broadcastHistoryMaxLimit
	}

	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.notificationsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "PUSH_BROADCAST"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list push broadcasts")
	}

	broadcasts := []*domain.PushBroadcast{}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &broadcasts); err != nil {
		return nil, errors.Internal("Failed to unmarshal push broadcasts")
	}
	return broadcasts, nil
}

// Ensure interface compliance
var _ domain.PushSubscriptionRepository = (*PushSubscriptionRepository)(nil)
