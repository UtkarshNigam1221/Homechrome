# Event-Driven Async Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace risky goroutines with SNS fan-out + SQS + worker Lambdas for notifications, reports, analytics, and audit logging.

**Architecture:** An `EventPublisher` interface abstracts event publishing. In monolith mode, `LocalPublisher` calls worker handlers synchronously in-process. In Lambda mode, `SNSPublisher` publishes to an SNS topic that fans out to 4 SQS queues, each triggering a dedicated worker Lambda. Services emit events via fire-and-forget calls; worker failures go to DLQs.

**Tech Stack:** Go 1.24, AWS SNS/SQS, AWS Lambda (SQS triggers), AWS CDK (Go), Google Wire DI, LocalStack for local dev.

**Design doc:** `docs/plans/2026-02-21-event-driven-async-design.md`

---

### Task 1: Event Types and Publisher Interface

**Files:**
- Create: `internal/event/types.go`
- Create: `internal/event/publisher.go`
- Test: `internal/event/publisher_test.go`

**Step 1: Create event types**

Create `internal/event/types.go`:

```go
package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType is a dot-namespaced event identifier used for SNS filter policies.
type EventType string

// Order events
const (
	OrderCreated       EventType = "order.created"
	OrderStatusChanged EventType = "order.status_changed"
	OrderCancelled     EventType = "order.cancelled"
)

// Payment events
const (
	PaymentReceived EventType = "payment.received"
	PaymentFailed   EventType = "payment.failed"
	PaymentRefunded EventType = "payment.refunded"
)

// Shipment events
const (
	ShipmentCreated   EventType = "shipment.created"
	ShipmentUpdated   EventType = "shipment.updated"
	ShipmentDelivered EventType = "shipment.delivered"
)

// Product events
const (
	ProductCreated EventType = "product.created"
	ProductUpdated EventType = "product.updated"
	ProductDeleted EventType = "product.deleted"
)

// Inventory events
const (
	InventoryLowStock   EventType = "inventory.low_stock"
	InventoryOutOfStock EventType = "inventory.out_of_stock"
	InventoryRestocked  EventType = "inventory.restocked"
)

// Customer events
const (
	CustomerRegistered EventType = "customer.registered"
	CustomerUpdated    EventType = "customer.updated"
)

// Admin events
const (
	AdminEntityModified EventType = "admin.entity_modified"
	AdminUserLogin      EventType = "admin.user_login"
)

// Event is the envelope published to SNS / consumed by workers.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// New creates a new Event. The data argument is JSON-marshalled into Event.Data.
func New(eventType EventType, data interface{}) Event {
	payload, _ := json.Marshal(data)
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    "handloom-api",
		Timestamp: time.Now(),
		Data:      payload,
	}
}
```

**Step 2: Create publisher interface and implementations**

Create `internal/event/publisher.go`:

```go
package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/handloom/admin/pkg/logger"
)

// EventPublisher publishes events to the event bus.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

// --- SNS Publisher (Lambda mode) ---

// SNSPublisher publishes events to an SNS topic.
type SNSPublisher struct {
	client   *sns.Client
	topicARN string
}

// NewSNSPublisher creates a publisher that sends to the given SNS topic.
// Pass a non-empty endpoint for LocalStack.
func NewSNSPublisher(topicARN, region, endpoint string) (*SNSPublisher, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if endpoint != "" {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint, SigningRegion: region}, nil
			}),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &SNSPublisher{
		client:   sns.NewFromConfig(cfg),
		topicARN: topicARN,
	}, nil
}

func (p *SNSPublisher) Publish(ctx context.Context, evt Event) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: &p.topicARN,
		Message:  aws.String(string(body)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(string(evt.Type)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("sns publish: %w", err)
	}
	return nil
}

// --- Local Publisher (monolith mode) ---

// EventHandler processes events synchronously in-process.
type EventHandler interface {
	CanHandle(eventType EventType) bool
	Handle(ctx context.Context, event Event) error
}

// LocalPublisher calls registered EventHandlers synchronously.
// Used in monolith mode (make run) — no SNS/SQS needed.
type LocalPublisher struct {
	handlers []EventHandler
	logger   *logger.Logger
}

// NewLocalPublisher creates a publisher that dispatches to in-process handlers.
func NewLocalPublisher(log *logger.Logger, handlers ...EventHandler) *LocalPublisher {
	return &LocalPublisher{handlers: handlers, logger: log}
}

func (p *LocalPublisher) Publish(ctx context.Context, evt Event) error {
	for _, h := range p.handlers {
		if h.CanHandle(evt.Type) {
			if err := h.Handle(ctx, evt); err != nil {
				p.logger.WithContext(ctx).WithError(err).Errorf(
					"local handler failed for event %s: %s", evt.Type, evt.ID)
			}
		}
	}
	return nil
}

// --- No-op Publisher (testing) ---

// NoopPublisher discards all events. Useful in tests and as a default.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ Event) error { return nil }
```

**Step 3: Write tests for publisher**

Create `internal/event/publisher_test.go`:

```go
package event

import (
	"context"
	"testing"

	"github.com/handloom/admin/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	evt := New(OrderCreated, map[string]string{"order_id": "ord_123"})

	assert.Equal(t, OrderCreated, evt.Type)
	assert.Equal(t, "handloom-api", evt.Source)
	assert.NotEmpty(t, evt.ID)
	assert.NotZero(t, evt.Timestamp)
	assert.Contains(t, string(evt.Data), "ord_123")
}

type spyHandler struct {
	handled []Event
	accept  EventType
}

func (h *spyHandler) CanHandle(t EventType) bool { return h.accept == t }
func (h *spyHandler) Handle(_ context.Context, e Event) error {
	h.handled = append(h.handled, e)
	return nil
}

func TestLocalPublisher_DispatchesToMatchingHandlers(t *testing.T) {
	log := logger.NewNoop()

	orderHandler := &spyHandler{accept: OrderCreated}
	paymentHandler := &spyHandler{accept: PaymentReceived}

	pub := NewLocalPublisher(log, orderHandler, paymentHandler)

	evt := New(OrderCreated, map[string]string{"id": "1"})
	err := pub.Publish(context.Background(), evt)
	require.NoError(t, err)

	assert.Len(t, orderHandler.handled, 1)
	assert.Len(t, paymentHandler.handled, 0)
}

func TestNoopPublisher(t *testing.T) {
	err := NoopPublisher{}.Publish(context.Background(), New(OrderCreated, nil))
	assert.NoError(t, err)
}
```

**Step 4: Run tests**

Run: `cd handloom-admin && go test -v ./internal/event/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/event/
git commit -m "feat(event): add EventPublisher interface, SNS/Local/Noop implementations"
```

---

### Task 2: Config Changes

**Files:**
- Modify: `internal/config/config.go` — add `EventConfig` struct and load from env

**Step 1: Add EventConfig to config**

Add to `internal/config/config.go`:

In the `Config` struct, add:
```go
Event EventConfig
```

Add the struct definition:
```go
// EventConfig holds event bus configuration
type EventConfig struct {
	SNSTopicARN string
	Enabled     bool
}
```

In the `Load()` function, add inside the return:
```go
Event: EventConfig{
	SNSTopicARN: getEnv("SNS_TOPIC_ARN", ""),
	Enabled:     getBoolEnv("EVENT_PUBLISHING_ENABLED", false),
},
```

**Step 2: Verify compilation**

Run: `cd handloom-admin && go build ./...`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add EventConfig for SNS topic and publishing toggle"
```

---

### Task 3: Wire Provider for EventPublisher

**Files:**
- Modify: `internal/wire/providers.go` — add `ProvideEventPublisher`

**Step 1: Add provider function**

Add import for `"github.com/handloom/admin/internal/event"` and the provider:

```go
// ProvideEventPublisher creates an EventPublisher based on config.
// When Event.Enabled is true (Lambda mode), uses SNS. Otherwise uses a no-op
// publisher (monolith mode uses LocalPublisher wired in cmd/api/main.go).
func ProvideEventPublisher(cfg *config.Config, log *logger.Logger) (event.EventPublisher, error) {
	if cfg.Event.Enabled {
		return event.NewSNSPublisher(cfg.Event.SNSTopicARN, cfg.AWS.Region, cfg.AWS.Endpoint)
	}
	return event.NoopPublisher{}, nil
}
```

Note: For Lambda mode, Wire provides `SNSPublisher`. For monolith mode (`cmd/api/main.go`), the publisher is constructed manually with `LocalPublisher` + registered handlers — not via Wire. The Wire provider returns `NoopPublisher` as a safe default for any context where the monolith's manual wiring hasn't been set up.

**Step 2: Verify compilation**

Run: `cd handloom-admin && go build ./...`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add internal/wire/providers.go
git commit -m "feat(wire): add ProvideEventPublisher provider"
```

---

### Task 4: Add EventPublisher to CheckoutService

**Files:**
- Modify: `internal/service/checkout_service.go`
- Modify: `internal/wire/providers.go` — update `ProvideCheckoutService`

**Step 1: Add publisher field and constructor param**

In `checkout_service.go`, add field to struct:
```go
publisher event.EventPublisher
```

Update `NewCheckoutService` to accept `publisher event.EventPublisher` and assign it.

**Step 2: Emit event after successful checkout**

In the `Initiate` method, after the cart is cleared (line ~224) and before the return statement, add:

```go
// Publish order.created event (fire-and-forget)
if pubErr := s.publisher.Publish(ctx, event.New(event.OrderCreated, order)); pubErr != nil {
	s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish order.created event")
}
```

**Step 3: Update Wire provider**

In `providers.go`, update `ProvideCheckoutService` to accept `event.EventPublisher` and pass it through.

**Step 4: Verify compilation**

Run: `cd handloom-admin && go build ./...`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add internal/service/checkout_service.go internal/wire/providers.go
git commit -m "feat(checkout): emit order.created event on successful checkout"
```

---

### Task 5: Add EventPublisher to PaymentService

**Files:**
- Modify: `internal/service/payment_service.go`
- Modify: `internal/wire/providers.go` — update `ProvidePaymentService`

**Step 1: Add publisher field and constructor param**

Add `publisher event.EventPublisher` to `PaymentService` struct and `NewPaymentService`.

**Step 2: Emit events in webhook handler**

In `handlePaymentSuccess` (after order status update, before the final log):
```go
if pubErr := s.publisher.Publish(ctx, event.New(event.PaymentReceived, map[string]interface{}{
	"payment_id": payment.ID,
	"order_id":   payment.OrderID,
	"amount":     payment.Amount,
})); pubErr != nil {
	s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish payment.received event")
}
```

In `handlePaymentFailure` (after inventory release):
```go
if pubErr := s.publisher.Publish(ctx, event.New(event.PaymentFailed, map[string]interface{}{
	"payment_id": payment.ID,
	"order_id":   payment.OrderID,
})); pubErr != nil {
	s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish payment.failed event")
}
```

**Step 3: Update Wire provider**

**Step 4: Verify compilation**

Run: `cd handloom-admin && go build ./...`

**Step 5: Commit**

```bash
git add internal/service/payment_service.go internal/wire/providers.go
git commit -m "feat(payment): emit payment.received/failed events on webhook"
```

---

### Task 6: Add EventPublisher to ProductService + InventoryService

**Files:**
- Modify: `internal/service/product_service.go`
- Modify: `internal/service/inventory_service.go`
- Modify: `internal/wire/providers.go`

**Step 1: ProductService — add publisher and emit events**

Add `publisher event.EventPublisher` to struct + constructor.

Emit `event.ProductCreated` at end of `Create()`, `event.ProductUpdated` at end of `Update()`, `event.ProductDeleted` at end of `Delete()`. All fire-and-forget with logged errors.

**Step 2: InventoryService — add publisher and emit events**

Add `publisher event.EventPublisher` to struct + constructor.

In `AddStock()`, after successful stock addition, check if the stock was previously low/out and is now restocked:
```go
if pubErr := s.publisher.Publish(ctx, event.New(event.InventoryRestocked, map[string]interface{}{
	"product_id":   productID,
	"new_quantity": txn.NewQty,
})); pubErr != nil {
	s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish inventory.restocked event")
}
```

In `RemoveStock()`, after successful removal, check low stock threshold:
```go
if inventory != nil && inventory.Quantity <= inventory.LowStockThreshold {
	evtType := event.InventoryLowStock
	if inventory.AvailableQty <= 0 {
		evtType = event.InventoryOutOfStock
	}
	if pubErr := s.publisher.Publish(ctx, event.New(evtType, map[string]interface{}{
		"product_id":   productID,
		"quantity":     inventory.Quantity,
		"available":    inventory.AvailableQty,
		"threshold":    inventory.LowStockThreshold,
	})); pubErr != nil {
		s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish inventory event")
	}
}
```

**Step 3: Update Wire providers for both services**

**Step 4: Verify compilation**

Run: `cd handloom-admin && go build ./...`

**Step 5: Commit**

```bash
git add internal/service/product_service.go internal/service/inventory_service.go internal/wire/providers.go
git commit -m "feat(catalog): emit product and inventory events"
```

---

### Task 7: Add EventPublisher to CustomerAuthService + Remove Goroutines

**Files:**
- Modify: `internal/service/customer_auth_service.go`
- Modify: `internal/service/notification_service.go`
- Modify: `internal/service/report_service.go`
- Modify: `internal/wire/providers.go`

**Step 1: CustomerAuthService — emit customer.registered**

Add `publisher event.EventPublisher` to struct + constructor.

In `VerifyOTP()`, after a new customer is created (when `isNew` is true), emit:
```go
if pubErr := s.publisher.Publish(ctx, event.New(event.CustomerRegistered, customer)); pubErr != nil {
	s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish customer.registered event")
}
```

**Step 2: NotificationService — remove goroutine**

In `notification_service.go`, in the `Send()` method, **remove**:
```go
// TODO: In production, this would be pushed to a queue (SQS) for async processing
// For now, we'll attempt to send synchronously
go s.processNotification(context.Background(), notification)
```

The notification will now be processed by the worker-notification Lambda (via the SQS queue triggered by events from other services). The `Send()` method becomes a pure CRUD operation for admin-triggered notifications. The `processNotification` and related methods can remain for now — they'll be used by the worker handler.

**Step 3: ReportService — remove goroutine**

In `report_service.go`, in the `Generate()` method, **remove**:
```go
// TODO: In production, push to SQS for async processing
// For now, we'll just create the report record
// The actual generation would be handled by a worker service
go s.processReport(context.Background(), report)
```

The report will now be processed by the worker-report Lambda. The `processReport` method remains — it will be called by the worker handler.

**Step 4: Update Wire providers**

Update `ProvideCustomerAuthService` to accept and pass `event.EventPublisher`.

**Step 5: Update existing tests**

In `internal/service/notification_service_test.go`, the `TestNotificationService_Send` test has:
```go
// processNotification runs in a goroutine - use AnyTimes for async Update
mockNotifRepo.EXPECT().
    Update(gomock.Any(), gomock.Any()).
    AnyTimes()
```
Remove that expectation — the goroutine no longer runs.

**Step 6: Run tests**

Run: `cd handloom-admin && go test -v ./internal/service/...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/service/customer_auth_service.go internal/service/notification_service.go \
  internal/service/report_service.go internal/service/notification_service_test.go \
  internal/wire/providers.go
git commit -m "feat(event): emit customer.registered, remove risky goroutines from notification/report"
```

---

### Task 8: Worker Event Handlers

**Files:**
- Create: `internal/event/handlers/notification.go`
- Create: `internal/event/handlers/report.go`
- Create: `internal/event/handlers/analytics.go`
- Create: `internal/event/handlers/audit.go`
- Test: `internal/event/handlers/handlers_test.go`

**Step 1: Create notification handler**

Create `internal/event/handlers/notification.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

// NotificationHandler processes notification events from SQS.
type NotificationHandler struct {
	notifRepo domain.NotificationRepository
	logger    *logger.Logger
}

func NewNotificationHandler(notifRepo domain.NotificationRepository, log *logger.Logger) *NotificationHandler {
	return &NotificationHandler{notifRepo: notifRepo, logger: log}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *NotificationHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.WithContext(ctx).WithError(err).Errorf("failed to process notification event: %s", record.MessageId)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *NotificationHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var evt event.Event
	if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
		return err
	}
	h.logger.WithContext(ctx).Infof("Processing notification for event %s: %s", evt.Type, evt.ID)
	// TODO: implement actual notification sending (SMS/email) based on event type
	return nil
}

// CanHandle returns true for events the notification worker cares about.
func (h *NotificationHandler) CanHandle(t event.EventType) bool {
	return strings.HasPrefix(string(t), "order.") ||
		strings.HasPrefix(string(t), "payment.") ||
		strings.HasPrefix(string(t), "shipment.") ||
		t == event.CustomerRegistered
}

// Handle processes a single event (used by LocalPublisher).
func (h *NotificationHandler) Handle(ctx context.Context, evt event.Event) error {
	h.logger.WithContext(ctx).Infof("[local] notification handler: %s %s", evt.Type, evt.ID)
	// TODO: implement actual notification sending for local dev
	return nil
}
```

**Step 2: Create report handler**

Create `internal/event/handlers/report.go` — same pattern. `CanHandle` matches `order.*` and `payment.*`. The handler is a stub for now — actual report generation logic will be wired in later using the existing `ReportService.processReport` method.

**Step 3: Create analytics handler**

Create `internal/event/handlers/analytics.go` — same pattern. `CanHandle` matches `order.*`, `payment.*`, `product.*`, `inventory.*`, `customer.*`.

**Step 4: Create audit handler**

Create `internal/event/handlers/audit.go` — same pattern. `CanHandle` returns `true` for all events.

**Step 5: Write tests**

Create `internal/event/handlers/handlers_test.go`:

```go
package handlers

import (
	"testing"

	"github.com/handloom/admin/internal/event"
	"github.com/stretchr/testify/assert"
)

func TestNotificationHandler_CanHandle(t *testing.T) {
	h := &NotificationHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.PaymentReceived))
	assert.True(t, h.CanHandle(event.ShipmentCreated))
	assert.True(t, h.CanHandle(event.CustomerRegistered))
	assert.False(t, h.CanHandle(event.ProductCreated))
	assert.False(t, h.CanHandle(event.AdminEntityModified))
}

func TestAuditHandler_CanHandle_AllEvents(t *testing.T) {
	h := &AuditHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.ProductDeleted))
	assert.True(t, h.CanHandle(event.AdminEntityModified))
}
```

**Step 6: Run tests**

Run: `cd handloom-admin && go test -v ./internal/event/...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/event/handlers/
git commit -m "feat(workers): add SQS event handlers for notification, report, analytics, audit"
```

---

### Task 9: Worker Lambda Entry Points

**Files:**
- Create: `cmd/lambda/worker-notification/main.go`
- Create: `cmd/lambda/worker-report/main.go`
- Create: `cmd/lambda/worker-analytics/main.go`
- Create: `cmd/lambda/worker-audit/main.go`

**Step 1: Create worker-notification entry point**

```go
package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Worker Notification Lambda")

	ctx := context.Background()
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	notifRepo := dynamodb.NewNotificationRepository(dbClient)
	handler := handlers.NewNotificationHandler(notifRepo, log)

	lambda.Start(handler.HandleSQSEvent)
}
```

**Step 2: Create worker-report, worker-analytics, worker-audit**

Same pattern — each constructs only the repos/services it needs and calls `lambda.Start(handler.HandleSQSEvent)`.

**Step 3: Verify compilation**

Run: `cd handloom-admin && go build ./cmd/lambda/worker-notification/ && go build ./cmd/lambda/worker-report/ && go build ./cmd/lambda/worker-analytics/ && go build ./cmd/lambda/worker-audit/`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/lambda/worker-notification/ cmd/lambda/worker-report/ cmd/lambda/worker-analytics/ cmd/lambda/worker-audit/
git commit -m "feat(workers): add Lambda entry points for 4 worker services"
```

---

### Task 10: Makefile Updates

**Files:**
- Modify: `Makefile`

**Step 1: Add worker build targets**

Add to the `LAMBDA_SERVICES` variable:
```
worker-notification worker-report worker-analytics worker-audit
```

Add to `.PHONY`:
```
build-workers
```

Add build targets:
```makefile
## Build all worker Lambda binaries
build-workers: build-lambda-worker-notification build-lambda-worker-report build-lambda-worker-analytics build-lambda-worker-audit
	@echo "All worker lambdas built successfully"
```

Add individual build targets following the existing pattern (e.g. `build-lambda-worker-notification`):
```makefile
build-lambda-worker-notification:
	@echo "Building worker-notification lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/worker-notification
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -o $(LAMBDA_OUT_DIR)/worker-notification/bootstrap ./cmd/lambda/worker-notification
```

Repeat for worker-report, worker-analytics, worker-audit.

**Step 2: Add create-events target**

```makefile
## Create SNS topic and SQS queues in LocalStack
create-events:
	@echo "Creating SNS topic and SQS queues..."
	@bash scripts/create-events.sh
```

Update `setup-local` to include `create-events` after `create-tables`.

**Step 3: Verify**

Run: `cd handloom-admin && make build-workers`
Expected: All 4 worker binaries built

**Step 4: Commit**

```bash
git add Makefile
git commit -m "build: add worker Lambda build targets and create-events Makefile target"
```

---

### Task 11: LocalStack Event Infrastructure Script

**Files:**
- Create: `scripts/create-events.sh`

**Step 1: Create the script**

```bash
#!/bin/bash
set -euo pipefail

ENDPOINT="http://localhost:4566"
REGION="ap-south-1"
ACCOUNT="000000000000"
ENV="local"

alias awslocal="aws --endpoint-url=$ENDPOINT --region $REGION"

echo "=== Creating SNS topic ==="
TOPIC_ARN=$(awslocal sns create-topic --name "handloom-events-${ENV}" --query 'TopicArn' --output text)
echo "Topic: $TOPIC_ARN"

create_queue_pair() {
    local name=$1
    local max_receive=$2

    echo "--- Creating $name queue pair ---"

    # DLQ
    DLQ_URL=$(awslocal sqs create-queue --queue-name "handloom-${name}-dlq-${ENV}" --query 'QueueUrl' --output text)
    DLQ_ARN="arn:aws:sqs:${REGION}:${ACCOUNT}:handloom-${name}-dlq-${ENV}"

    # Main queue with redrive policy
    QUEUE_URL=$(awslocal sqs create-queue \
        --queue-name "handloom-${name}-${ENV}" \
        --attributes "{\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"${DLQ_ARN}\\\",\\\"maxReceiveCount\\\":\\\"${max_receive}\\\"}\"}" \
        --query 'QueueUrl' --output text)
    QUEUE_ARN="arn:aws:sqs:${REGION}:${ACCOUNT}:handloom-${name}-${ENV}"

    echo "  Queue: $QUEUE_URL"
    echo "  DLQ:   $DLQ_URL"
    echo "$QUEUE_ARN"
}

NOTIF_ARN=$(create_queue_pair "notification" 3)
REPORT_ARN=$(create_queue_pair "report" 3)
ANALYTICS_ARN=$(create_queue_pair "analytics" 3)
AUDIT_ARN=$(create_queue_pair "audit" 5)

echo ""
echo "=== Subscribing queues to SNS topic ==="

subscribe_with_filter() {
    local queue_arn=$1
    local filter=$2
    local name=$3

    awslocal sns subscribe \
        --topic-arn "$TOPIC_ARN" \
        --protocol sqs \
        --notification-endpoint "$queue_arn" \
        --attributes "{\"FilterPolicy\":\"$filter\"}" \
        > /dev/null

    echo "  Subscribed $name"
}

subscribe_with_filter "$NOTIF_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"},{\"prefix\":\"shipment.\"},{\"anything-but\":[]}]}' \
    "notification"

subscribe_with_filter "$REPORT_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"}]}' \
    "report"

subscribe_with_filter "$ANALYTICS_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"},{\"prefix\":\"product.\"},{\"prefix\":\"inventory.\"},{\"prefix\":\"customer.\"}]}' \
    "analytics"

# Audit gets ALL events — no filter policy
awslocal sns subscribe \
    --topic-arn "$TOPIC_ARN" \
    --protocol sqs \
    --notification-endpoint "$AUDIT_ARN" \
    > /dev/null
echo "  Subscribed audit (all events)"

echo ""
echo "=== Event infrastructure ready ==="
echo "SNS Topic ARN: $TOPIC_ARN"
```

**Step 2: Make executable**

Run: `chmod +x handloom-admin/scripts/create-events.sh`

**Step 3: Commit**

```bash
git add scripts/create-events.sh
git commit -m "infra(local): add LocalStack SNS/SQS bootstrap script"
```

---

### Task 12: CDK EventStack

**Files:**
- Create: `infra/stacks/events.go`
- Modify: `infra/cmd/main.go` — instantiate EventStack

**Step 1: Create events.go**

Create `infra/stacks/events.go` with CDK constructs for:
- 1 SNS topic (`handloom-events-{env}`)
- 4 SQS queues + 4 DLQs
- 4 SNS subscriptions with filter policies
- 4 worker Lambda functions (SQS event source mapping)
- IAM: worker Lambdas get SQS consume + DynamoDB read/write

Follow the existing patterns from `api.go` — same memory, ARM64, `provided.al2023`, log retention.

The stack should export the SNS topic ARN so the API stack can pass it to producer Lambdas as env var.

**Step 2: Modify infra/cmd/main.go**

In `createEnvironmentStacks`, add EventStack creation after StorageStack, before APIStack. Pass the EventStack's topic ARN to APIStack props.

**Step 3: Modify infra/stacks/api.go**

Add `SNS_TOPIC_ARN` and `EVENT_PUBLISHING_ENABLED=true` to `commonEnv` map.

Add `EventStack *EventStack` to `APIStackProps`.

**Step 4: Verify CDK synth**

Run: `cd handloom-admin/infra && go build ./...`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add infra/
git commit -m "infra(cdk): add EventStack with SNS topic, SQS queues, and worker Lambdas"
```

---

### Task 13: Wire Regeneration and Monolith Integration

**Files:**
- Modify: `internal/wire/wire.go` — add worker dep containers + update store injectors
- Modify: `cmd/api/main.go` — wire LocalPublisher with handlers in monolith mode

**Step 1: Update wire.go**

Add `ProvideEventPublisher` to the relevant `Initialize*Deps` functions that need it. Specifically:
- `InitializeStoreCheckoutDeps` — CheckoutService needs publisher
- `InitializeStoreWebhooksDeps` — PaymentService needs publisher
- `InitializeStoreCatalogDeps` — ProductService needs publisher
- `InitializeStoreCartDeps` — CartService (if adding events later)
- `InitializeCatalogDeps` (admin) — ProductService needs publisher

Also update `InitializeApiDeps` to include the publisher.

**Step 2: Regenerate wire_gen.go**

Run: `cd handloom-admin && make wire`
Expected: `wire_gen.go` regenerated without errors

**Step 3: Wire LocalPublisher in cmd/api/main.go**

In the monolith entry point, after all dependencies are created, construct the `LocalPublisher` with the event handlers and inject it into the services that need it. This is manual wiring since the monolith constructs things differently.

```go
// Create event handlers for local publisher
notifHandler := handlers.NewNotificationHandler(notifRepo, log)
reportHandler := handlers.NewReportHandler(reportRepo, log)
analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo, log)
auditHandler := handlers.NewAuditHandler(auditRepo, log)

localPublisher := event.NewLocalPublisher(log, notifHandler, reportHandler, analyticsHandler, auditHandler)

// Pass localPublisher to services that emit events
```

**Step 4: Verify full build**

Run: `cd handloom-admin && go build ./...`
Expected: Everything builds

**Step 5: Commit**

```bash
git add internal/wire/ cmd/api/main.go
git commit -m "feat(wire): integrate EventPublisher into DI and monolith entry point"
```

---

### Task 14: Update .env.example and Documentation

**Files:**
- Modify: `.env.example`
- Modify: `docs/plans/2026-02-21-event-driven-async-design.md` — mark as implemented

**Step 1: Add env vars**

Add to `.env.example`:
```
# Event Bus Configuration
SNS_TOPIC_ARN=arn:aws:sns:ap-south-1:000000000000:handloom-events-local
EVENT_PUBLISHING_ENABLED=false
```

**Step 2: Run full test suite**

Run: `cd handloom-admin && make test`
Expected: All tests pass

**Step 3: Final commit**

```bash
git add .env.example docs/
git commit -m "docs: update env example and mark event-driven design as implemented"
```

---

### Task 15: Integration Smoke Test

**Step 1: Start LocalStack**

Run: `cd handloom-admin && make docker-up`

**Step 2: Bootstrap local infra**

Run: `cd handloom-admin && make create-tables && make create-events`

**Step 3: Start monolith**

Run: `cd handloom-admin && make run`

**Step 4: Verify events fire locally**

Trigger a checkout flow and verify log output shows `[local] notification handler:` and `[local] audit handler:` lines, confirming the LocalPublisher is dispatching events to handlers in-process.

**Step 5: Verify Lambda build**

Run: `cd handloom-admin && make build-lambdas && make build-workers`
Expected: All binaries (including 4 workers) build successfully

---

Plan complete and saved to `docs/plans/2026-02-21-event-driven-async.md`. Two execution options:

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** — Open new session with executing-plans, batch execution with checkpoints

Which approach?
