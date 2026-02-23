# Storefront Analytics Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a self-hosted analytics pipeline that captures storefront events and produces business metrics across funnel, revenue, customers, products, and engagement.

**Architecture:** Frontend batches tracking events → `POST /api/v1/store/events` → raw event storage + live counter updates. Backend transactional events flow through existing SNS pipeline → analytics worker. Daily cron aggregates raw events into queryable metrics. Admin dashboard reads pre-computed aggregates.

**Tech Stack:** Go 1.24, DynamoDB, Chi router, Next.js 16, navigator.sendBeacon

**Design Doc:** `docs/plans/2026-02-23-storefront-analytics-design.md`

---

## Phase 1: Infrastructure — New Events Table

### Task 1: Add events table to local setup script

**Files:**
- Modify: `scripts/init-local-db.sh`

**Step 1:** Add the `handloom-events` table creation after the existing analytics table block (around line 209). Follow the same pattern as other tables:

```bash
# Events table (raw event store)
echo "Creating events table..."
TABLE_EXISTS=$(aws dynamodb describe-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-events 2>/dev/null)

if [ -z "$TABLE_EXISTS" ]; then
    aws dynamodb create-table \
        --endpoint-url $ENDPOINT \
        --region $REGION \
        --table-name handloom-events \
        --attribute-definitions \
            AttributeName=PK,AttributeType=S \
            AttributeName=SK,AttributeType=S \
        --key-schema \
            AttributeName=PK,KeyType=HASH \
            AttributeName=SK,KeyType=RANGE \
        --billing-mode PAY_PER_REQUEST > /dev/null 2>&1
    echo "Created handloom-events table"
else
    echo "Table handloom-events already exists"
fi
```

Also add TTL enablement below the existing TTL block:

```bash
aws dynamodb update-time-to-live \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-events \
    --time-to-live-specification "Enabled=true, AttributeName=ttl"
echo "Enabled TTL on handloom-events table"
```

**Step 2:** Run `make setup-local` to verify the table is created.

**Step 3:** Commit.

```bash
git add scripts/init-local-db.sh
git commit -m "infra: add handloom-events table to local setup"
```

### Task 2: Add events table to config and DynamoDB client

**Files:**
- Modify: `internal/config/config.go:77-85` (DynamoDBConfig)
- Modify: `internal/config/config.go` (Load function, around the DynamoDB env var block)
- Modify: `internal/repository/dynamodb/client.go:18-27` (Client struct)
- Modify: `internal/repository/dynamodb/client.go` (NewClient and accessor)

**Step 1:** Add `EventsTable` field to `DynamoDBConfig`:

```go
type DynamoDBConfig struct {
	CoreTable          string
	CatalogTable       string
	OrdersTable        string
	SessionsTable      string
	AuditTable         string
	AnalyticsTable     string
	NotificationsTable string
	EventsTable        string // Raw event store
}
```

**Step 2:** Add env var loading in `Load()` — find where `DYNAMODB_NOTIFICATIONS_TABLE` is loaded and add:

```go
EventsTable: getEnv("DYNAMODB_EVENTS_TABLE", "handloom-events"),
```

**Step 3:** Add `eventsTable` to the DynamoDB `Client` struct and its accessor:

```go
type Client struct {
	db                 *dynamodb.Client
	coreTable          string
	catalogTable       string
	ordersTable        string
	sessionsTable      string
	auditTable         string
	analyticsTable     string
	notificationsTable string
	eventsTable        string
}
```

Add the `eventsTable: cfg.DynamoDB.EventsTable` assignment in `NewClient` and the accessor:

```go
func (c *Client) EventsTable() string {
	return c.eventsTable
}
```

**Step 4:** Run `go build ./...` to verify.

**Step 5:** Commit.

```bash
git add internal/config/config.go internal/repository/dynamodb/client.go
git commit -m "infra: add events table config and DynamoDB client accessor"
```

### Task 3: Add events table to CDK database stack

**Files:**
- Modify: `infra/stacks/database.go` (after analyticsTable creation, ~line 306)

**Step 1:** Add the events table following the analytics table pattern:

```go
eventsTable := awsdynamodb.NewTable(stack, jsii.String("EventsTable"), &awsdynamodb.TableProps{
	TableName:     jsii.String("handloom-events-" + props.Environment),
	BillingMode:   billingMode,
	RemovalPolicy: removalPolicy,
	PartitionKey: &awsdynamodb.Attribute{
		Name: jsii.String("PK"),
		Type: awsdynamodb.AttributeType_STRING,
	},
	SortKey: &awsdynamodb.Attribute{
		Name: jsii.String("SK"),
		Type: awsdynamodb.AttributeType_STRING,
	},
	TimeToLiveAttribute: jsii.String("ttl"),
})
```

No GSIs needed. Add table to the `DatabaseStackOutputs` if that pattern is used.

**Step 2:** Run `go build ./infra/...` to verify.

**Step 3:** Commit.

```bash
git add infra/stacks/database.go
git commit -m "infra: add events table to CDK database stack"
```

---

## Phase 2: Domain & Data Model Changes

### Task 4: Add CategoryID to CartItem and OrderItem

**Files:**
- Modify: `internal/domain/cart.go:34-51` (CartItem struct)
- Modify: `internal/domain/order.go:84-101` (OrderItem struct)

**Step 1:** Add `CategoryID` and `CategoryName` to both structs.

In `CartItem` (after `ProductImage` field):

```go
CategoryID   string `json:"category_id" dynamodbav:"category_id"`
CategoryName string `json:"category_name" dynamodbav:"category_name"`
```

In `OrderItem` (after `ProductImage` field):

```go
CategoryID   string `json:"category_id" dynamodbav:"category_id"`
CategoryName string `json:"category_name" dynamodbav:"category_name"`
```

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/domain/cart.go internal/domain/order.go
git commit -m "feat: add CategoryID/CategoryName to CartItem and OrderItem"
```

### Task 5: Populate CategoryID in cart and checkout flows

**Files:**
- Modify: `internal/service/cart_service.go:63-76` (AddItem)
- Modify: `internal/service/checkout_service.go:129-141` (order item construction)

**Step 1:** In `CartService.AddItem()` at line 63, the product is already fetched on line 49. Add category fields to the CartItem construction:

```go
item := &domain.CartItem{
	ProductID:    req.ProductID,
	ProductName:  product.Name,
	ProductSKU:   product.SKU,
	ProductImage: primaryImage(product.Images),
	CategoryID:   product.CategoryID,
	CategoryName: product.CategoryName,
	// ... rest unchanged
}
```

Check what field name the product uses for category name — it may be stored on the Category entity, not the Product. If Product doesn't have `CategoryName`, just populate `CategoryID` and leave `CategoryName` empty for now.

**Step 2:** In `CheckoutService.Initiate()` at line 129, add the category fields when constructing OrderItems from cart items:

```go
orderItems = append(orderItems, domain.OrderItem{
	// ... existing fields ...
	CategoryID:   item.CategoryID,
	CategoryName: item.CategoryName,
	// ... rest unchanged
})
```

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/service/cart_service.go internal/service/checkout_service.go
git commit -m "feat: populate CategoryID in cart and checkout flows"
```

### Task 6: Define event ingestion request types

**Files:**
- Create: `internal/domain/store_event.go`

**Step 1:** Create the domain types for frontend event ingestion:

```go
package domain

import "time"

// StoreEvent represents a single frontend tracking event
type StoreEvent struct {
	EventType  string                 `json:"event_type" validate:"required"`
	Timestamp  time.Time              `json:"timestamp" validate:"required"`
	SessionID  string                 `json:"session_id" validate:"required"`
	VisitorID  string                 `json:"visitor_id" validate:"required"`
	DeviceType string                 `json:"device_type"`
	PagePath   string                 `json:"page_path"`
	Properties map[string]interface{} `json:"properties"`
}

// StoreEventBatch is the request body for POST /api/v1/store/events
type StoreEventBatch struct {
	Events []StoreEvent `json:"events" validate:"required,min=1,max=25"`
}
```

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/domain/store_event.go
git commit -m "feat: add StoreEvent domain types for frontend tracking"
```

---

## Phase 3: Raw Event Repository

### Task 7: Create events repository

**Files:**
- Create: `internal/repository/dynamodb/events_repository.go`
- Modify: `internal/domain/repository.go` (add interface)

**Step 1:** Add the repository interface to `internal/domain/repository.go`:

```go
// EventsRepository stores and queries raw tracking events
type EventsRepository interface {
	BatchWriteEvents(ctx context.Context, events []StoreEvent) error
	QueryByDate(ctx context.Context, date string) ([]StoreEvent, error)
}
```

**Step 2:** Create the repository implementation. Key patterns:
- PK: `EVENT#YYYY-MM-DD` (partition by date)
- SK: `{timestamp}#{uuid}` (sort within day)
- TTL: 30 days from event timestamp
- Use `BatchWriteItem` for bulk writes (max 25 per call, matches our batch limit)
- `QueryByDate` uses `Query` on the date partition — needed for daily aggregation

```go
package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
)

const eventTTLDays = 30

type EventsRepository struct {
	client *Client
}

func NewEventsRepository(client *Client) *EventsRepository {
	return &EventsRepository{client: client}
}

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

	_, err := r.client.DB().BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			r.client.EventsTable(): requests,
		},
	})
	return err
}

func (r *EventsRepository) QueryByDate(ctx context.Context, date string) ([]domain.StoreEvent, error) {
	var events []domain.StoreEvent
	var lastKey map[string]types.AttributeValue

	for {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.client.EventsTable()),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "EVENT#" + date},
			},
			ExclusiveStartKey: lastKey,
		}

		result, err := r.client.DB().Query(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, item := range result.Items {
			var evt domain.StoreEvent
			if err := attributevalue.UnmarshalMap(item, &evt); err != nil {
				continue
			}
			// Parse properties from JSON string
			if propsStr, ok := item["properties"].(*types.AttributeValueMemberS); ok {
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(propsStr.Value), &props); err == nil {
					evt.Properties = props
				}
			}
			events = append(events, evt)
		}

		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}

	return events, nil
}
```

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/domain/repository.go internal/repository/dynamodb/events_repository.go
git commit -m "feat: add events repository for raw event storage"
```

### Task 8: Wire events repository into DI

**Files:**
- Modify: `internal/wire/providers.go` (add provider, add to RepositorySet)

**Step 1:** Add provider function:

```go
func ProvideEventsRepository(client *dynamodb.Client) *dynamodb.EventsRepository {
	return dynamodb.NewEventsRepository(client)
}
```

Add `ProvideEventsRepository` to `RepositorySet`.

**Step 2:** Run `make wire` to regenerate wire_gen.go.

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/wire/providers.go internal/wire/wire_gen.go
git commit -m "feat: wire events repository into DI"
```

---

## Phase 4: Event Ingestion Endpoint

### Task 9: Create store events handler

**Files:**
- Create: `internal/handler/store/events_handler.go`

**Step 1:** Follow the store handler pattern (see `cart_handler.go`). The events endpoint is public (no auth), accepts a batch of events, writes to the raw events table, and updates live counters.

```go
package store

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
	dynamodbrepo "github.com/handloom/admin/internal/repository/dynamodb"
)

type EventsHandler struct {
	eventsRepo *dynamodbrepo.EventsRepository
	validation *middleware.Validation
	logger     *logger.Logger
}

func NewEventsHandler(
	eventsRepo *dynamodbrepo.EventsRepository,
	validation *middleware.Validation,
	logger *logger.Logger,
) *EventsHandler {
	return &EventsHandler{
		eventsRepo: eventsRepo,
		validation: validation,
		logger:     logger,
	}
}

func (h *EventsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(middleware.ValidateJSONTyped[domain.StoreEventBatch](h.validation)).
		Post("/", h.IngestEvents)
	return r
}

func (h *EventsHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	batch := middleware.MustGetValidatedBody[domain.StoreEventBatch](ctx)

	// Filter out events older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	valid := make([]domain.StoreEvent, 0, len(batch.Events))
	for _, evt := range batch.Events {
		if evt.Timestamp.After(cutoff) {
			valid = append(valid, evt)
		}
	}

	if len(valid) == 0 {
		response.JSON(w, http.StatusAccepted, map[string]int{"accepted": 0})
		return
	}

	// Write raw events (best-effort, don't fail the request)
	if err := h.eventsRepo.BatchWriteEvents(ctx, valid); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("failed to write raw events")
	}

	// TODO: Phase 5 — update live counters on DASHBOARD#CURRENT

	response.JSON(w, http.StatusAccepted, map[string]int{"accepted": len(valid)})
}
```

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/handler/store/events_handler.go
git commit -m "feat: add store events handler for frontend tracking"
```

### Task 10: Add store events router and Lambda entry point

**Files:**
- Create: `internal/router/store_events.go`
- Create: `cmd/lambda/store-events/main.go`

**Step 1:** Create the router (public, no auth — follows `store_catalog.go` pattern):

```go
package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler/store"
)

func NewStoreEventsRouter(r *chi.Mux, h *store.EventsHandler) {
	r.Mount("/api/v1/store/events", h.Routes())
}
```

**Step 2:** Create the Lambda entry point following `cmd/lambda/store-catalog/main.go` pattern. Wire function: `InitializeStoreEventsDeps`.

**Step 3:** Add `InitializeStoreEventsDeps` to `internal/wire/wire.go` — needs `EventsRepository`, `Validation`, `Logger`.

**Step 4:** Run `make wire` then `go build ./...`.

**Step 5:** Mount the router in `cmd/api/main.go` (monolith mode) alongside other store routes.

**Step 6:** Commit.

```bash
git add internal/router/store_events.go cmd/lambda/store-events/ internal/wire/ cmd/api/main.go
git commit -m "feat: add store events router and Lambda entry point"
```

---

## Phase 5: Live Counter Updates

### Task 11: Add live counter update logic to analytics repository

**Files:**
- Modify: `internal/repository/dynamodb/analytics_repository.go`

**Step 1:** Implement the `RecordEvent` method (currently a stub) to atomically update `DASHBOARD#CURRENT` using `UpdateItem` with `ADD` expressions. Each event type updates different counters:

- `page_view` → increment `today_visitors` (use a SET to deduplicate by visitor_id — or just increment for simplicity in v1)
- `add_to_cart` → increment `today_add_to_carts`
- `product_viewed` → increment `today_product_views`
- `order.created` → increment `today_orders`, add to `today_revenue`
- `payment.received` → increment `today_payments_success`
- `payment.failed` → increment `today_payments_failed`
- `customer.registered` → increment `today_new_customers`

Use DynamoDB `UpdateExpression` with `ADD` for atomic increments:

```go
func (r *AnalyticsRepository) IncrementDashboardCounter(ctx context.Context, field string, amount int64) error {
	_, err := r.client.DB().UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.AnalyticsTable()),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DASHBOARD#CURRENT"},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("ADD #field :val"),
		ExpressionAttributeNames: map[string]string{
			"#field": field,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", amount)},
		},
	})
	return err
}
```

**Step 2:** Call this from `EventsHandler.IngestEvents()` for each event in the batch (replace the TODO from Task 9).

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/repository/dynamodb/analytics_repository.go internal/handler/store/events_handler.go
git commit -m "feat: add live counter updates on DASHBOARD#CURRENT"
```

### Task 12: Implement GetDashboardStats to read live counters

**Files:**
- Modify: `internal/repository/dynamodb/analytics_repository.go`

**Step 1:** Replace the mock `GetDashboardStats` with a real `GetItem` on `DASHBOARD#CURRENT`:

```go
func (r *AnalyticsRepository) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	result, err := r.client.DB().GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.AnalyticsTable()),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DASHBOARD#CURRENT"},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return &domain.DashboardStats{}, nil
	}

	var stats domain.DashboardStats
	if err := attributevalue.UnmarshalMap(result.Item, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
```

Update the `DashboardStats` struct in `internal/domain/analytics.go` to have `dynamodbav` tags matching the field names used in `IncrementDashboardCounter` (e.g., `today_orders`, `today_revenue`, etc.).

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/repository/dynamodb/analytics_repository.go internal/domain/analytics.go
git commit -m "feat: implement GetDashboardStats reading DASHBOARD#CURRENT"
```

---

## Phase 6: Analytics Worker Enhancement

### Task 13: Enhance analytics event handler to write raw events and update counters

**Files:**
- Modify: `internal/event/handlers/analytics.go`

**Step 1:** The analytics handler currently logs events. Enhance it to:
1. Write the raw event to `handloom-events` table
2. Update live counters on `DASHBOARD#CURRENT`

Add `EventsRepository` and `AnalyticsRepository` as dependencies:

```go
type AnalyticsHandler struct {
	eventsRepo    *dynamodbrepo.EventsRepository
	analyticsRepo domain.AnalyticsRepository
	logger        *logger.Logger
}
```

**Step 2:** In `processRecord`, unmarshal the SNS event, convert to a `StoreEvent` for raw storage, and call `IncrementDashboardCounter` based on event type.

For `order.created`: unmarshal `Data` as `domain.Order`, increment `today_orders` by 1 and `today_revenue` by `order.TotalAmount`.

For `payment.received`: increment `today_payments_success`.

For `payment.failed`: increment `today_payments_failed`.

For `customer.registered`: increment `today_new_customers`.

**Step 3:** Update the Wire DI for the analytics worker Lambda to inject the new dependencies.

**Step 4:** Run `make wire` then `go build ./...`.

**Step 5:** Commit.

```bash
git add internal/event/handlers/analytics.go internal/wire/
git commit -m "feat: enhance analytics worker to store raw events and update counters"
```

---

## Phase 7: Daily Aggregation

### Task 14: Add daily aggregation logic

**Files:**
- Create: `internal/service/analytics_aggregator.go`

**Step 1:** Create an aggregation service that:
1. Queries all raw events for a given date from `handloom-events`
2. Groups and computes each aggregate type (funnel, revenue, customers, engagement, etc.)
3. Writes the computed records to `handloom-analytics`

```go
type AnalyticsAggregator struct {
	eventsRepo    *dynamodbrepo.EventsRepository
	analyticsRepo domain.AnalyticsRepository
	logger        *logger.Logger
}

func (a *AnalyticsAggregator) AggregateDate(ctx context.Context, date string) error {
	events, err := a.eventsRepo.QueryByDate(ctx, date)
	if err != nil {
		return err
	}

	// Compute each aggregate type
	a.aggregateFunnel(ctx, date, events)
	a.aggregateRevenue(ctx, date, events)
	a.aggregateCustomers(ctx, date, events)
	a.aggregateEngagement(ctx, date, events)
	a.aggregateProducts(ctx, date, events)
	a.aggregateCheckout(ctx, date, events)
	a.aggregateLocation(ctx, date, events)
	a.aggregateCatalog(ctx, date, events)

	return nil
}
```

Each `aggregate*` method groups events by type, computes the metrics defined in the design doc, and writes a single DynamoDB item (e.g., `PK=FUNNEL#DAILY#2026-02-23, SK=METADATA`).

**Step 2:** Add repository methods to write daily aggregate records (e.g., `PutDailyAggregate`).

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/service/analytics_aggregator.go internal/repository/dynamodb/analytics_repository.go
git commit -m "feat: add daily analytics aggregation logic"
```

### Task 15: Add scheduled aggregation trigger

**Files:**
- Modify: `internal/event/handlers/analytics.go` (add scheduled mode)
- Modify: `cmd/lambda/worker-analytics/main.go` (handle EventBridge trigger)

**Step 1:** The analytics worker Lambda needs to handle two event sources:
- SQS messages (real-time events from SNS)
- EventBridge scheduled events (daily aggregation)

Add a handler for CloudWatch Events / EventBridge schedule:

```go
func (h *AnalyticsHandler) HandleScheduledEvent(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	return h.aggregator.AggregateDate(ctx, yesterday)
}
```

In the Lambda main, check the event type to route between SQS and scheduled.

**Step 2:** Add EventBridge rule to CDK (in `infra/stacks/events.go`):

```go
// Daily aggregation at 00:30 UTC
rule := awsevents.NewRule(stack, jsii.String("DailyAnalyticsAggregation"), &awsevents.RuleProps{
	Schedule: awsevents.Schedule_Cron(&awsevents.CronOptions{
		Minute: jsii.String("30"),
		Hour:   jsii.String("0"),
	}),
})
rule.AddTarget(awseventstargets.NewLambdaFunction(analyticsWorkerLambda, nil))
```

**Step 3:** Run `go build ./...` to verify.

**Step 4:** Commit.

```bash
git add internal/event/handlers/analytics.go cmd/lambda/worker-analytics/ infra/stacks/events.go
git commit -m "feat: add scheduled daily aggregation trigger"
```

### Task 16: Add dashboard counter reset on new day

**Files:**
- Modify: `internal/service/analytics_aggregator.go`

**Step 1:** At the end of `AggregateDate()`, after computing all aggregates for yesterday, archive yesterday's `DASHBOARD#CURRENT` as `DASHBOARD#STATS#{date}` and reset the current counters to zero.

```go
func (a *AnalyticsAggregator) resetDashboardCounters(ctx context.Context, date string) error {
	// 1. Read current counters
	stats, _ := a.analyticsRepo.GetDashboardStats(ctx)

	// 2. Write as historical record: DASHBOARD#STATS#YYYY-MM-DD
	a.analyticsRepo.PutDailyStats(ctx, date, stats)

	// 3. Reset DASHBOARD#CURRENT to zeros
	return a.analyticsRepo.ResetDashboardCurrent(ctx)
}
```

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/service/analytics_aggregator.go internal/repository/dynamodb/analytics_repository.go
git commit -m "feat: archive and reset dashboard counters on daily aggregation"
```

---

## Phase 8: Admin Analytics Endpoints

### Task 17: Add funnel and engagement endpoints to analytics handler

**Files:**
- Modify: `internal/handler/analytics_handler.go` (add GetFunnelAnalytics, GetEngagementAnalytics)
- Modify: `internal/router/analytics.go` (mount new routes)

**Step 1:** Add two new handler methods following the existing `GetSalesAnalytics` pattern. Both accept `start_date` and `end_date` query params, query the corresponding `DAILY` records from the analytics table, and return the aggregated data.

Add routes to `internal/router/analytics.go`:

```go
r.Get("/funnel", h.GetFunnelAnalytics)
r.Get("/engagement", h.GetEngagementAnalytics)
```

**Step 2:** Add corresponding service and repository methods:
- `AnalyticsService.GetFunnelAnalytics(ctx, startDate, endDate)`
- `AnalyticsService.GetEngagementAnalytics(ctx, startDate, endDate)`
- `AnalyticsRepository.GetDailyAggregates(ctx, prefix, startDate, endDate)` — generic method to query `{PREFIX}#DAILY#{date}` records for a date range

**Step 3:** Add new domain response types to `internal/domain/analytics.go`:

```go
type FunnelAnalytics struct {
	Period             DateRange       `json:"period"`
	Steps              []FunnelStep    `json:"steps"`
	OverallConversion  float64         `json:"overall_conversion"`
	CheckoutDropOff    map[string]int  `json:"checkout_drop_off"`
}

type FunnelStep struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate,omitempty"`
}

type EngagementAnalytics struct {
	TotalSessions          int                `json:"total_sessions"`
	AvgSessionDuration     int                `json:"avg_session_duration_seconds"`
	BounceRate             float64            `json:"bounce_rate"`
	DeviceBreakdown        map[string]float64 `json:"device_breakdown"`
	TopPages               []PageStats        `json:"top_pages"`
	TopExitPages           []PageStats        `json:"top_exit_pages"`
	ScrollDepthAvg         map[string]int     `json:"scroll_depth_avg"`
	ImageInteractionRate   float64            `json:"image_interaction_rate"`
	FilterUsage            []FilterStats      `json:"filter_usage"`
	Errors                 []ErrorStats       `json:"errors"`
}
```

**Step 4:** Run `go build ./...` to verify.

**Step 5:** Commit.

```bash
git add internal/handler/analytics_handler.go internal/router/analytics.go internal/service/analytics_service.go internal/repository/dynamodb/analytics_repository.go internal/domain/analytics.go
git commit -m "feat: add funnel and engagement analytics endpoints"
```

### Task 18: Implement remaining analytics repository stubs

**Files:**
- Modify: `internal/repository/dynamodb/analytics_repository.go`

**Step 1:** Replace all remaining mock/TODO methods with real DynamoDB queries:
- `GetSalesAnalytics` → Query `REVENUE#DAILY#` or `REVENUE#MONTHLY#` records
- `GetTopProducts` → Query `PRODUCT_VIEWS#DAILY#` records, aggregate top products
- `GetTopCategories` → Query `REVENUE#DAILY#` records, extract by_category arrays
- `GetCustomerAnalytics` → Query `CUSTOMERS#DAILY#` records
- `GetInventoryAnalytics` → Query `CATALOG#DAILY#` records

All follow the same pattern: Query by PK prefix for the date range, unmarshal, aggregate if spanning multiple days, return.

**Step 2:** Run `go build ./...` to verify.

**Step 3:** Commit.

```bash
git add internal/repository/dynamodb/analytics_repository.go
git commit -m "feat: implement analytics repository DynamoDB queries"
```

---

## Phase 9: Frontend Tracking Library

### Task 19: Create analytics tracking module for Next.js storefront

**Files:**
- Create: `homechrome-store/src/lib/analytics.ts`

**Step 1:** Create the tracking module with batching and sendBeacon support:

```typescript
const BATCH_SIZE = 10;
const FLUSH_INTERVAL_MS = 30000;
const API_URL = '/api/store/events';

interface TrackingEvent {
  event_type: string;
  timestamp: string;
  session_id: string;
  visitor_id: string;
  device_type: 'mobile' | 'desktop' | 'tablet';
  page_path: string;
  properties: Record<string, unknown>;
}

let eventBuffer: TrackingEvent[] = [];
let flushTimer: ReturnType<typeof setInterval> | null = null;

function getSessionId(): string {
  let id = sessionStorage.getItem('hc_session_id');
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem('hc_session_id', id);
  }
  return id;
}

function getVisitorId(): string {
  let id = localStorage.getItem('hc_visitor_id');
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem('hc_visitor_id', id);
  }
  return id;
}

function getDeviceType(): 'mobile' | 'desktop' | 'tablet' {
  const width = window.innerWidth;
  if (width < 768) return 'mobile';
  if (width < 1024) return 'tablet';
  return 'desktop';
}

export function track(eventType: string, properties: Record<string, unknown> = {}) {
  const event: TrackingEvent = {
    event_type: eventType,
    timestamp: new Date().toISOString(),
    session_id: getSessionId(),
    visitor_id: getVisitorId(),
    device_type: getDeviceType(),
    page_path: window.location.pathname,
    properties,
  };
  eventBuffer.push(event);

  if (eventBuffer.length >= BATCH_SIZE) {
    flush();
  }
}

function flush() {
  if (eventBuffer.length === 0) return;
  const batch = eventBuffer.splice(0);
  const body = JSON.stringify({ events: batch });

  // Use sendBeacon for reliability (works during page unload)
  if (navigator.sendBeacon) {
    navigator.sendBeacon(API_URL, new Blob([body], { type: 'application/json' }));
  } else {
    fetch(API_URL, { method: 'POST', body, headers: { 'Content-Type': 'application/json' }, keepalive: true });
  }
}

export function initAnalytics() {
  if (typeof window === 'undefined') return;

  // Periodic flush
  flushTimer = setInterval(flush, FLUSH_INTERVAL_MS);

  // Flush on page unload
  window.addEventListener('beforeunload', flush);

  // Flush on visibility change (tab backgrounded)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') flush();
  });
}

export function stopAnalytics() {
  if (flushTimer) clearInterval(flushTimer);
  flush();
}
```

**Step 2:** Commit.

```bash
git add homechrome-store/src/lib/analytics.ts
git commit -m "feat: add frontend analytics tracking library with batching"
```

### Task 20: Instrument storefront pages with tracking events

**Files:**
- Modify: `homechrome-store/src/app/layout.tsx` (init analytics)
- Modify: `homechrome-store/src/app/page.tsx` (home page_view)
- Modify: `homechrome-store/src/app/p/[slug]/page.tsx` (product_viewed, product_image_interaction)
- Modify: `homechrome-store/src/app/c/[slug]/page.tsx` (page_view with category, filter_used, sort_used)
- Modify: cart components (add_to_cart, remove_from_cart)
- Modify: checkout components (checkout_step)
- Modify: login components (otp_flow)

**Step 1:** In the root layout, initialize analytics:

```tsx
'use client';
import { useEffect } from 'react';
import { initAnalytics, stopAnalytics } from '@/lib/analytics';

function AnalyticsProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    initAnalytics();
    return () => stopAnalytics();
  }, []);
  return <>{children}</>;
}
```

**Step 2:** Add `track()` calls to each page/component. Examples:

Product page:
```tsx
import { track } from '@/lib/analytics';

// On mount
track('product_viewed', { product_id: product.id, product_name: product.name, category_id: product.category_id, price: product.selling_price });
```

Add to cart:
```tsx
track('add_to_cart', { product_id, product_name, category_id, price, quantity, success: true });
```

Category page with UTM capture:
```tsx
const params = new URLSearchParams(window.location.search);
track('page_view', {
  page_type: 'category',
  category_slug: slug,
  utm_source: params.get('utm_source') || undefined,
  utm_medium: params.get('utm_medium') || undefined,
  utm_campaign: params.get('utm_campaign') || undefined,
});
```

**Step 3:** Add scroll depth tracking as a reusable hook:

```tsx
// src/hooks/useScrollDepth.ts
import { useEffect } from 'react';
import { track } from '@/lib/analytics';

export function useScrollDepth(pageType: string) {
  useEffect(() => {
    let maxDepth = 0;
    const handler = () => {
      const depth = Math.round((window.scrollY / (document.body.scrollHeight - window.innerHeight)) * 100);
      maxDepth = Math.max(maxDepth, depth);
    };
    window.addEventListener('scroll', handler, { passive: true });
    return () => {
      window.removeEventListener('scroll', handler);
      track('scroll_depth', { page_type: pageType, max_depth_percent: maxDepth });
    };
  }, [pageType]);
}
```

**Step 4:** Commit.

```bash
git add homechrome-store/src/
git commit -m "feat: instrument storefront pages with analytics tracking"
```

### Task 21: Add Next.js API rewrite for events endpoint

**Files:**
- Modify: `homechrome-store/next.config.ts`

**Step 1:** The storefront already rewrites `/api/*` to the backend. Verify the events endpoint is covered. The analytics library posts to `/api/store/events`, which should match the existing rewrite pattern for `/api/v1/store/*`. If the rewrite uses a different prefix, adjust the `API_URL` in `analytics.ts` to match.

Check the existing rewrite config and ensure `POST /api/v1/store/events` reaches the backend.

**Step 2:** Test by running `npm run dev` in the storefront and verifying events are sent to the backend.

**Step 3:** Commit if any changes were needed.

---

## Summary

| Phase | Tasks | What it builds |
|-------|-------|----------------|
| 1. Infrastructure | 1-3 | New events table (local + CDK + config) |
| 2. Domain | 4-6 | CategoryID on items, StoreEvent types |
| 3. Raw Event Storage | 7-8 | Events repository + DI wiring |
| 4. Event Ingestion | 9-10 | Store events handler, router, Lambda |
| 5. Live Counters | 11-12 | DASHBOARD#CURRENT updates + reads |
| 6. Analytics Worker | 13 | Process backend events through pipeline |
| 7. Daily Aggregation | 14-16 | Scheduled aggregation + counter reset |
| 8. Admin Endpoints | 17-18 | Funnel, engagement endpoints + implement stubs |
| 9. Frontend Tracking | 19-21 | Next.js tracking library + page instrumentation |

**Total: 21 tasks across 9 phases.** Each phase is independently testable. Phases 1-5 give you a working event pipeline with live dashboard. Phases 6-8 add historical analytics. Phase 9 connects the frontend.
