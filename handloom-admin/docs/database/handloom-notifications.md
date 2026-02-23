# handloom-notifications Table

The notifications table stores system notifications and alerts.

## Table Configuration

```
Table Name: handloom-notifications
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
TTL Attribute: ttl
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |

---

## Entities

### 1. Notification

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `NOTIFICATION#<id>` | `NOTIFICATION#notif-001` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Type | String | Yes | `EMAIL`, `SMS`, `PUSH` |
| Status | String | Yes | `PENDING`, `SENT`, `FAILED`, `DELIVERED` |
| RecipientID | String | Yes | Recipient user/customer ID |
| RecipientEmail | String | No | Recipient email |
| RecipientPhone | String | No | Recipient phone |
| Subject | String | No | Notification subject |
| Body | String | Yes | Notification content |
| TemplateID | String | No | Template used |
| TemplateData | Map | No | Template variables |
| TriggerType | String | Yes | `ORDER_STATUS`, `ORDER_CREATED`, `SHIPMENT`, `PAYMENT`, `REFUND`, `PASSWORD_RESET`, `MANUAL` |
| ReferenceType | String | No | Related entity type |
| ReferenceID | String | No | Related entity ID |
| SentAt | String | No | Send timestamp |
| DeliveredAt | String | No | Delivery timestamp |
| FailedAt | String | No | Failure timestamp |
| FailureReason | String | No | Failure reason |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get notification by ID | PK = `NOTIFICATION#<id>`, SK = `METADATA` |

#### Write Patterns

| Operation | Method | Condition |
|-----------|--------|-----------|
| Create | `PutItem` | `attribute_not_exists(PK)` |
| Update | `PutItem` | `attribute_exists(PK)` |
