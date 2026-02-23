package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
)

const eventTTLDays = 30

// EventsRepository implements domain.EventsRepository
type EventsRepository struct {
	client *Client
}

// NewEventsRepository creates a new EventsRepository
func NewEventsRepository(client *Client) *EventsRepository {
	return &EventsRepository{client: client}
}

// BatchWriteEvents writes a batch of tracking events to the events table.
// Events are partitioned by date (PK=EVENT#YYYY-MM-DD) with a unique sort key
// composed of the timestamp and a UUID. Each item has a 30-day TTL.
func (r *EventsRepository) BatchWriteEvents(ctx context.Context, events []domain.StoreEvent) error {
	if len(events) == 0 {
		return nil
	}

	requests := make([]types.WriteRequest, 0, len(events))
	for _, evt := range events {
		date := evt.Timestamp.Format("2006-01-02")
		sk := evt.Timestamp.Format(time.RFC3339Nano) + "#" + uuid.New().String()
		ttl := evt.Timestamp.Add(eventTTLDays * 24 * time.Hour).Unix()

		propsJSON, _ := json.Marshal(evt.Properties)

		item := map[string]types.AttributeValue{
			"PK":          &types.AttributeValueMemberS{Value: "EVENT#" + date},
			"SK":          &types.AttributeValueMemberS{Value: sk},
			"event_type":  &types.AttributeValueMemberS{Value: evt.EventType},
			"timestamp":   &types.AttributeValueMemberS{Value: evt.Timestamp.Format(time.RFC3339)},
			"session_id":  &types.AttributeValueMemberS{Value: evt.SessionID},
			"visitor_id":  &types.AttributeValueMemberS{Value: evt.VisitorID},
			"device_type": &types.AttributeValueMemberS{Value: evt.DeviceType},
			"page_path":   &types.AttributeValueMemberS{Value: evt.PagePath},
			"properties":  &types.AttributeValueMemberS{Value: string(propsJSON)},
			"ttl":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttl)},
		}

		requests = append(requests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}

	_, err := r.client.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			r.client.eventsTable: requests,
		},
	})
	return err
}

// QueryByDate retrieves all events for a given date string (YYYY-MM-DD).
// It paginates through all items in the date partition.
func (r *EventsRepository) QueryByDate(ctx context.Context, date string) ([]domain.StoreEvent, error) {
	var events []domain.StoreEvent
	var lastKey map[string]types.AttributeValue

	for {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.client.eventsTable),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "EVENT#" + date},
			},
			ExclusiveStartKey: lastKey,
		}

		result, err := r.client.db.Query(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, item := range result.Items {
			evt := parseStoreEvent(item)
			events = append(events, evt)
		}

		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}

	return events, nil
}

// parseStoreEvent converts a DynamoDB item map back into a domain.StoreEvent.
func parseStoreEvent(item map[string]types.AttributeValue) domain.StoreEvent {
	evt := domain.StoreEvent{}

	if v, ok := item["event_type"].(*types.AttributeValueMemberS); ok {
		evt.EventType = v.Value
	}
	if v, ok := item["timestamp"].(*types.AttributeValueMemberS); ok {
		evt.Timestamp, _ = time.Parse(time.RFC3339, v.Value)
	}
	if v, ok := item["session_id"].(*types.AttributeValueMemberS); ok {
		evt.SessionID = v.Value
	}
	if v, ok := item["visitor_id"].(*types.AttributeValueMemberS); ok {
		evt.VisitorID = v.Value
	}
	if v, ok := item["device_type"].(*types.AttributeValueMemberS); ok {
		evt.DeviceType = v.Value
	}
	if v, ok := item["page_path"].(*types.AttributeValueMemberS); ok {
		evt.PagePath = v.Value
	}
	if v, ok := item["properties"].(*types.AttributeValueMemberS); ok {
		var props map[string]interface{}
		if err := json.Unmarshal([]byte(v.Value), &props); err == nil {
			evt.Properties = props
		}
	}

	return evt
}

// Ensure interface compliance
var _ domain.EventsRepository = (*EventsRepository)(nil)
