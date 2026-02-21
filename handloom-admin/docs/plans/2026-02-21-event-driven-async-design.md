# Event-Driven Async Layer Design

**Date:** 2026-02-21
**Status:** Implemented
**Scope:** SNS fan-out + SQS + Worker Lambdas for async processing

## Problem

Services currently use `go s.processNotification()` and `go s.processReport()` goroutines for async work. This is dangerous in Lambda — goroutines get killed when the Lambda freezes after responding. There is also no retry mechanism, no dead-letter queue, and no observability into failures.

Additionally, cross-cutting concerns (audit logging, analytics aggregation) are either synchronous in the request path or missing entirely. Decoupling these via events improves request latency and separates concerns.

## Decision

**Approach: SNS fan-out + SQS queues + dedicated worker Lambdas.**

A single SNS topic receives all system events. Each async workload (notification, report, analytics, audit) has its own SQS queue subscribed to the topic with filter policies. Each queue triggers a dedicated worker Lambda.

Local dev (monolith mode) uses a `LocalPublisher` that calls worker logic synchronously in-process — no SNS/SQS needed for `make run`.

## Architecture

```
Producer Services (Checkout, Order, Payment, Shipping, Catalog, Inventory, CustomerAuth)
    │
    │  publisher.Publish(ctx, event)
    │
    ├── [Lambda mode] ──► SNSPublisher ──► SNS: handloom-events-{env}
    │                                          │
    │                         ┌─────────┬──────┴───────┬──────────┐
    │                         ▼         ▼              ▼          ▼
    │                   notification  report      analytics    audit
    │                   queue + DLQ   queue + DLQ  queue + DLQ  queue + DLQ
    │                         │         │              │          │
    │                         ▼         ▼              ▼          ▼
    │                   worker-       worker-      worker-    worker-
    │                   notification  report       analytics  audit
    │
    └── [Monolith mode] ──► LocalPublisher ──► calls handler functions directly
```

## Event Types

Dot-namespaced strings used as SNS message attributes for filter policies.

### Order Events
- `order.created` — emitted by CheckoutService after successful order placement
- `order.status_changed` — emitted by OrderService on status transitions
- `order.cancelled` — emitted by OrderService on cancellation

### Payment Events
- `payment.received` — emitted by PaymentService on successful payment
- `payment.failed` — emitted by PaymentService on payment failure
- `payment.refunded` — emitted by PaymentService on refund

### Shipment Events
- `shipment.created` — emitted by ShippingService when shipment is created
- `shipment.updated` — emitted by ShippingService on tracking updates
- `shipment.delivered` — emitted by ShippingService on delivery confirmation

### Product Events
- `product.created` — emitted by ProductService
- `product.updated` — emitted by ProductService
- `product.deleted` — emitted by ProductService

### Inventory Events
- `inventory.low_stock` — emitted by InventoryService when stock falls below threshold
- `inventory.out_of_stock` — emitted by InventoryService when stock reaches zero
- `inventory.restocked` — emitted by InventoryService when stock is added

### Customer Events
- `customer.registered` — emitted by CustomerAuthService on new registration
- `customer.updated` — emitted by ProfileHandler on profile changes

### Admin Events
- `admin.entity_modified` — emitted by admin catalog/order services on CRUD operations
- `admin.user_login` — emitted by AuthService on login

## Event Struct

```go
package event

type EventType string

type Event struct {
    ID        string          `json:"id"`        // uuid
    Type      EventType       `json:"type"`      // e.g. "order.created"
    Source    string          `json:"source"`     // e.g. "checkout-service"
    Timestamp time.Time       `json:"timestamp"`
    Data      json.RawMessage `json:"data"`      // typed payload per event type
}

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

## Publisher Interface

```go
type EventPublisher interface {
    Publish(ctx context.Context, event Event) error
}
```

### SNSPublisher

Used in Lambda mode. Publishes JSON-encoded event to SNS with `event_type` message attribute for filter-based routing.

```go
type SNSPublisher struct {
    client   *sns.Client
    topicARN string
}

func (p *SNSPublisher) Publish(ctx context.Context, evt Event) error {
    body, err := json.Marshal(evt)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    _, err = p.client.Publish(ctx, &sns.PublishInput{
        TopicArn: &p.topicARN,
        Message:  aws.String(string(body)),
        MessageAttributes: map[string]types.MessageAttributeValue{
            "event_type": {
                DataType:    aws.String("String"),
                StringValue: aws.String(string(evt.Type)),
            },
        },
    })
    return err
}
```

### LocalPublisher

Used in monolith mode (`make run`). Calls worker handler functions synchronously in the same process. No SNS/SQS infrastructure needed.

```go
type LocalPublisher struct {
    handlers []EventHandler
    logger   *logger.Logger
}

type EventHandler interface {
    CanHandle(eventType EventType) bool
    Handle(ctx context.Context, event Event) error
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
```

## SNS/SQS Configuration

### SNS Topic
- Name: `handloom-events-{env}`
- One topic for all event types

### SQS Queues

| Queue | Filter Policy | Batch Size | Visibility Timeout | DLQ Max Receives |
|-------|--------------|------------|-------------------|-----------------|
| `handloom-notification-{env}` | `order.*`, `payment.*`, `shipment.*`, `customer.registered` | 10 | 60s | 3 |
| `handloom-report-{env}` | `order.*`, `payment.*` | 1 | 120s | 3 |
| `handloom-analytics-{env}` | `order.*`, `payment.*`, `product.*`, `inventory.*`, `customer.*` | 10 | 60s | 3 |
| `handloom-audit-{env}` | `*` (all events) | 10 | 60s | 5 |

### DLQ Queues
- Name: `handloom-{workload}-dlq-{env}`
- Message retention: 14 days
- One per workload for isolated failure investigation

### SNS Filter Policy Example (notification queue)

```json
{
  "event_type": [
    {"prefix": "order."},
    {"prefix": "payment."},
    {"prefix": "shipment."},
    "customer.registered"
  ]
}
```

## Worker Lambdas

### Entry Point Pattern

Workers use `lambda.Start()` with SQS event handler (not the HTTP/Chi adapter).

```go
// cmd/lambda/worker-notification/main.go
func main() {
    cfg := config.Load()
    log := logger.New(cfg.App.Debug)
    ctx := context.Background()

    deps, err := wire.InitializeNotificationWorkerDeps(ctx, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize dependencies: %v", err)
    }

    handler := handlers.NewNotificationHandler(deps.NotificationService, deps.SMSGateway, log)
    lambda.Start(handler.HandleSQSEvent)
}
```

### SQS Batch Processing

Workers use `SQSBatchResponse` with `BatchItemFailures` for partial failure reporting. One bad message doesn't block the batch.

```go
func (h *NotificationHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
    var failures []events.SQSBatchItemFailure
    for _, record := range sqsEvent.Records {
        var evt event.Event
        if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
            failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
            continue
        }
        if err := h.process(ctx, evt); err != nil {
            failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
        }
    }
    return events.SQSEventResponse{BatchItemFailures: failures}, nil
}
```

### Worker Responsibilities

**worker-notification:**
- Dependencies: NotificationRepo, CustomerRepo, SMSGateway (MSG91)
- Processes: `order.created` (confirmation SMS), `payment.received` (receipt), `shipment.created/updated` (tracking), `customer.registered` (welcome)
- Replaces: `go s.processNotification()` goroutine in NotificationService

**worker-report:**
- Dependencies: ReportRepo, OrderRepo, ProductRepo, CustomerRepo, InventoryRepo, S3Client
- Processes: checks for pending report requests, generates file, uploads to S3
- Replaces: `go s.processReport()` goroutine in ReportService

**worker-analytics:**
- Dependencies: AnalyticsRepo, OrderRepo, ProductRepo, InventoryRepo
- Processes: all business events, updates aggregated metrics in analytics DynamoDB table
- Moves analytics from read-time computation to write-time aggregation

**worker-audit:**
- Dependencies: AuditRepo
- Processes: all events, writes AuditLog entries to audit table
- Decouples audit logging from the synchronous request path

### Wire Dependencies

```go
type NotificationWorkerDeps struct {
    NotificationService *service.NotificationService
    SMSGateway          domain.SMSGateway
    Logger              *logger.Logger
}

type ReportWorkerDeps struct {
    ReportService *service.ReportService
    Logger        *logger.Logger
}

type AnalyticsWorkerDeps struct {
    AnalyticsService *service.AnalyticsService
    Logger           *logger.Logger
}

type AuditWorkerDeps struct {
    AuditService *service.AuditService
    Logger       *logger.Logger
}
```

## Integration Points

Services receive `EventPublisher` as a constructor dependency and call `publisher.Publish()` at the appropriate points.

### Publishing Pattern

Event publishing is **fire-and-forget**. If SNS publish fails, the main operation still succeeds. The failure is logged but doesn't roll back the business operation.

```go
if pubErr := s.publisher.Publish(ctx, event.New(event.OrderCreated, order)); pubErr != nil {
    s.logger.WithContext(ctx).WithError(pubErr).Error("failed to publish order.created event")
}
```

### Services Modified to Emit Events

| Service | Events Emitted |
|---------|---------------|
| CheckoutService | `order.created` |
| OrderService | `order.status_changed`, `order.cancelled` |
| PaymentService | `payment.received`, `payment.failed`, `payment.refunded` |
| ShippingService | `shipment.created`, `shipment.updated`, `shipment.delivered` |
| ProductService | `product.created`, `product.updated`, `product.deleted` |
| InventoryService | `inventory.low_stock`, `inventory.out_of_stock`, `inventory.restocked` |
| CustomerAuthService | `customer.registered` |

### Services Modified to Remove Goroutines

| Service | Removed |
|---------|---------|
| NotificationService | `go s.processNotification()` — now handled by worker-notification |
| ReportService | `go s.processReport()` — now handled by worker-report |

## Config Changes

```go
// Added to config.go
type EventConfig struct {
    SNSTopicARN string // env: SNS_TOPIC_ARN
    Enabled     bool   // env: EVENT_PUBLISHING_ENABLED (true in Lambda, false in monolith)
}
```

```go
// Wire provider
func ProvideEventPublisher(cfg *config.Config, log *logger.Logger) (event.EventPublisher, error) {
    if cfg.Event.Enabled {
        return event.NewSNSPublisher(cfg.Event.SNSTopicARN, cfg.AWS.Region, cfg.AWS.Endpoint)
    }
    return event.NewLocalPublisher(log), nil
}
```

## CDK Infrastructure

New file: `infra/stacks/events.go`

Creates:
- 1 SNS topic
- 4 SQS queues + 4 DLQ queues
- 4 SNS-to-SQS subscriptions with filter policies
- 4 worker Lambda functions (SQS-triggered)
- IAM: producer Lambdas get `sns:Publish`, workers get `sqs:*` on their queue

Worker Lambda config:
- ARM64, `provided.al2023`, 128MB (dev) / 256MB (prod)
- Timeout: 60s
- Reserved concurrency: notification=5, report=2, analytics=5, audit=10

Producer Lambdas get new env vars:
- `SNS_TOPIC_ARN` — the topic ARN
- `EVENT_PUBLISHING_ENABLED=true`

## LocalStack Setup

New Makefile target `create-events` (called by `setup-local`):

```bash
# Create SNS topic
awslocal sns create-topic --name handloom-events-local

# Create DLQ + main queue pairs
awslocal sqs create-queue --queue-name handloom-notification-dlq-local
awslocal sqs create-queue --queue-name handloom-notification-local \
  --attributes '{"RedrivePolicy":"{\"deadLetterTargetArn\":\"...\",\"maxReceiveCount\":\"3\"}"}'
# ... repeat for report, analytics, audit

# Subscribe queues to SNS with filter policies
awslocal sns subscribe --topic-arn ... --protocol sqs --notification-endpoint ... \
  --attributes '{"FilterPolicy":"..."}'
```

Only needed for `make deploy-local` (Lambda mode). Monolith mode (`make run`) uses `LocalPublisher`.

## File Changes Summary

### New Files (14)

| File | Purpose |
|------|---------|
| `internal/event/publisher.go` | EventPublisher interface, SNSPublisher, LocalPublisher |
| `internal/event/types.go` | Event struct, EventType constants, New() helper |
| `internal/event/handlers/notification.go` | SQS handler for notification worker |
| `internal/event/handlers/report.go` | SQS handler for report worker |
| `internal/event/handlers/analytics.go` | SQS handler for analytics worker |
| `internal/event/handlers/audit.go` | SQS handler for audit worker |
| `cmd/lambda/worker-notification/main.go` | Lambda entry point |
| `cmd/lambda/worker-report/main.go` | Lambda entry point |
| `cmd/lambda/worker-analytics/main.go` | Lambda entry point |
| `cmd/lambda/worker-audit/main.go` | Lambda entry point |
| `infra/stacks/events.go` | CDK: SNS, SQS, DLQs, worker Lambdas |
| `internal/wire/wire_worker.go` | Wire injectors for worker deps |
| `scripts/create-events.sh` | LocalStack SNS/SQS bootstrap |
| `internal/event/handlers/handler_test.go` | Unit tests for event handlers |

### Modified Files (16)

| File | Change |
|------|--------|
| `internal/config/config.go` | Add EventConfig struct |
| `internal/wire/providers.go` | Add ProvideEventPublisher, update service constructors |
| `internal/wire/wire_gen.go` | Regenerated by `make wire` |
| `internal/service/checkout_service.go` | Add publisher.Publish(order.created) |
| `internal/service/order_service.go` | Add publisher.Publish(order.status_changed/cancelled) |
| `internal/service/payment_service.go` | Add publisher.Publish(payment.*) |
| `internal/service/shipping_service.go` | Add publisher.Publish(shipment.*) |
| `internal/service/notification_service.go` | Remove go s.processNotification() |
| `internal/service/report_service.go` | Remove go s.processReport() |
| `internal/service/product_service.go` | Add publisher.Publish(product.*) |
| `internal/service/inventory_service.go` | Add publisher.Publish(inventory.*) |
| `internal/service/customer_auth_service.go` | Add publisher.Publish(customer.registered) |
| `Makefile` | Add build-workers, create-events targets |
| `infra/main.go` | Instantiate EventStack |
| `infra/stacks/api.go` | Add SNS_TOPIC_ARN env var to producer Lambdas |
| `.env.example` | Add SNS_TOPIC_ARN, EVENT_PUBLISHING_ENABLED |

## Implementation Order

1. `internal/event/` — types + publisher interface + both implementations
2. Config changes — EventConfig struct
3. Wire provider for EventPublisher
4. Modify services — accept publisher, emit events, remove goroutines
5. Worker event handlers — `internal/event/handlers/`
6. Worker Lambda entry points — `cmd/lambda/worker-*/`
7. Wire injectors for workers
8. CDK `events.go` stack
9. LocalStack scripts + Makefile
10. Tests
