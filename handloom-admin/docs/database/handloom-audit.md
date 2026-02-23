# handloom-audit Table

The audit table stores all system audit logs with automatic expiration via TTL.

## Table Configuration

```
Table Name: handloom-audit
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
TTL Attribute: ttl
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |
| GSI2 | GSI2PK | GSI2SK | ALL |

---

## Design Philosophy

### Date-Based Partitioning

The audit table uses **date-based partitioning** for optimal query performance and data management:

```
PK: AUDIT#<YYYY-MM-DD>
SK: <HH:MM:SS.sssZ>#<uuid>
```

**Benefits:**
- Efficient daily audit queries
- Natural data distribution across partitions
- Simplified date-range queries
- Easy retention management via TTL

### TTL-Based Retention

- **Default Retention**: 90 days
- **TTL Attribute**: `ttl` (Unix timestamp)
- **Auto-Deletion**: DynamoDB automatically deletes expired items

---

## Entity: Audit Log

### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `AUDIT#<date>` | `AUDIT#2024-01-15` |
| SK | `<time>#<id>` | `10:30:45.123Z#audit-001` |
| GSI1PK | `<entity_type>#<entity_id>` | `PRODUCT#prod-001` |
| GSI1SK | `<timestamp>` | `2024-01-15T10:30:45.123Z` |
| GSI2PK | `USER#<user_id>` | `USER#user-001` |
| GSI2SK | `<timestamp>` | `2024-01-15T10:30:45.123Z` |

### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| UserID | String | Yes | User who performed the action |
| UserEmail | String | Yes | User email (denormalized) |
| UserRole | String | Yes | User role at time of action |
| Action | String | Yes | Action type |
| EntityType | String | Yes | Entity type affected |
| EntityID | String | Yes | Entity ID affected |
| Changes | Map | No | Field-level changes |
| OldValues | Map | No | Previous values |
| NewValues | Map | No | New values |
| IPAddress | String | No | Client IP address |
| UserAgent | String | No | Client user agent |
| RequestID | String | No | Request correlation ID |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp for TTL |

### Action Enum

| Action | Description |
|--------|-------------|
| `CREATE` | Entity created |
| `UPDATE` | Entity updated |
| `DELETE` | Entity deleted |
| `LOGIN` | User logged in |
| `LOGOUT` | User logged out |
| `LOGIN_FAILED` | Failed login attempt |
| `PASSWORD_CHANGE` | Password changed |
| `PERMISSION_CHANGE` | Permissions modified |
| `EXPORT` | Data exported |
| `IMPORT` | Data imported |
| `BULK_OPERATION` | Bulk operation performed |

### EntityType Enum

| EntityType | Description |
|------------|-------------|
| `USER` | Admin user |
| `CATEGORY` | Product category |
| `DESIGN` | Design template |
| `PRODUCT` | Product |
| `INVENTORY` | Inventory record |
| `PRICING_RULE` | Pricing rule |
| `COUPON` | Discount coupon |
| `ARTISAN` | Artisan |
| `ORDER` | Customer order |
| `CUSTOMER` | Customer |
| `ASSET` | Media asset |
| `REPORT` | Generated report |
| `NOTIFICATION` | Notification |
| `BULK_JOB` | Bulk operation job |
| `SYSTEM` | System-level action |

### Changes Structure

```json
{
  "Status": {
    "OldValue": "DRAFT",
    "NewValue": "ACTIVE"
  },
  "Price": {
    "OldValue": 100000,
    "NewValue": 120000
  }
}
```

---

## Access Patterns

### 1. Get Audit Logs for a Specific Date

Query all actions that occurred on a specific date.

```
Table: handloom-audit
KeyCondition:
  PK = "AUDIT#2024-01-15"
ScanIndexForward: false (newest first)
```

**Use Case**: Daily audit review, compliance reporting

### 2. Get Audit Logs in Time Range for a Date

Query actions within a specific time window.

```
Table: handloom-audit
KeyCondition:
  PK = "AUDIT#2024-01-15"
  SK BETWEEN "10:00:00.000Z" AND "18:00:00.000Z"
```

**Use Case**: Investigating incidents, shift-based reviews

### 3. Get Actions for a Specific Entity

Query audit trail for a specific entity via GSI1.

```
Table: handloom-audit
Index: GSI1
KeyCondition:
  GSI1PK = "PRODUCT#prod-001"
ScanIndexForward: false (newest first)
```

**Use Case**: Entity change history, investigating modifications

### 4. Get All Actions by a User

Query all actions performed by a specific user via GSI2.

```
Table: handloom-audit
Index: GSI2
KeyCondition:
  GSI2PK = "USER#user-001"
ScanIndexForward: false (newest first)
```

**Use Case**: User activity audit, permission reviews

### 5. Get User Actions in Date Range

Query user actions within a specific period.

```
Table: handloom-audit
Index: GSI2
KeyCondition:
  GSI2PK = "USER#user-001"
  GSI2SK BETWEEN "2024-01-01T00:00:00Z" AND "2024-01-31T23:59:59Z"
```

**Use Case**: Monthly user activity reports

---

## TTL Configuration

### How TTL Works

1. Each audit log has a `ttl` attribute (Unix timestamp)
2. DynamoDB periodically scans for expired items
3. Expired items are deleted automatically (within 48 hours of expiration)
4. No read/write capacity consumed for deletions

### Calculating TTL

```go
// 90 days retention
ttl := time.Now().Add(90 * 24 * time.Hour).Unix()
```

### Retention Periods by Environment

| Environment | Retention | Reason |
|-------------|-----------|--------|
| Development | 7 days | Reduce storage costs |
| Staging | 30 days | Testing retention |
| Production | 90 days | Compliance requirements |

### Extending Retention

For compliance or legal holds:
1. Update TTL attribute to extend retention
2. Remove TTL attribute to prevent deletion
3. Export to S3 for long-term archival

---

## Example Audit Log Entries

### User Login

```json
{
  "PK": "AUDIT#2024-01-15",
  "SK": "10:30:45.123Z#audit-001",
  "GSI1PK": "USER#user-001",
  "GSI1SK": "2024-01-15T10:30:45.123Z",
  "GSI2PK": "USER#user-001",
  "GSI2SK": "2024-01-15T10:30:45.123Z",
  "ID": "audit-001",
  "UserID": "user-001",
  "UserEmail": "admin@handloom.com",
  "UserRole": "ADMIN",
  "Action": "LOGIN",
  "EntityType": "USER",
  "EntityID": "user-001",
  "IPAddress": "192.168.1.100",
  "UserAgent": "Mozilla/5.0...",
  "CreatedAt": "2024-01-15T10:30:45.123Z",
  "ttl": 1713171045
}
```

### Product Update

```json
{
  "PK": "AUDIT#2024-01-15",
  "SK": "14:22:30.456Z#audit-002",
  "GSI1PK": "USER#user-001",
  "GSI1SK": "2024-01-15T14:22:30.456Z",
  "ID": "audit-002",
  "UserID": "user-001",
  "UserEmail": "admin@handloom.com",
  "UserRole": "ADMIN",
  "Action": "UPDATE",
  "EntityType": "PRODUCT",
  "EntityID": "prod-001",
  "Changes": {
    "Status": {
      "OldValue": "DRAFT",
      "NewValue": "ACTIVE"
    },
    "SellingPrice": {
      "OldValue": 150000,
      "NewValue": 175000
    }
  },
  "OldValues": {
    "Status": "DRAFT",
    "SellingPrice": 150000
  },
  "NewValues": {
    "Status": "ACTIVE",
    "SellingPrice": 175000
  },
  "RequestID": "req-abc-123",
  "CreatedAt": "2024-01-15T14:22:30.456Z",
  "ttl": 1713192150
}
```

### Order Status Change

```json
{
  "PK": "AUDIT#2024-01-15",
  "SK": "16:45:00.789Z#audit-003",
  "GSI1PK": "ORDER#ord-001",
  "GSI1SK": "2024-01-15T16:45:00.789Z",
  "GSI2PK": "USER#user-002",
  "GSI2SK": "2024-01-15T16:45:00.789Z",
  "ID": "audit-003",
  "UserID": "user-002",
  "UserEmail": "operator@handloom.com",
  "UserRole": "OPERATOR",
  "Action": "UPDATE",
  "EntityType": "ORDER",
  "EntityID": "ord-001",
  "Changes": {
    "Status": {
      "OldValue": "PROCESSING",
      "NewValue": "SHIPPED"
    },
    "TrackingNumber": {
      "OldValue": null,
      "NewValue": "TRACK123456"
    }
  },
  "CreatedAt": "2024-01-15T16:45:00.789Z",
  "ttl": 1713200700
}
```

### Bulk Import

```json
{
  "PK": "AUDIT#2024-01-15",
  "SK": "09:00:00.000Z#audit-004",
  "GSI1PK": "USER#user-001",
  "GSI1SK": "2024-01-15T09:00:00.000Z",
  "ID": "audit-004",
  "UserID": "user-001",
  "UserEmail": "admin@handloom.com",
  "UserRole": "ADMIN",
  "Action": "IMPORT",
  "EntityType": "BULK_JOB",
  "EntityID": "job-001",
  "NewValues": {
    "Type": "PRODUCT_IMPORT",
    "FileName": "products.csv",
    "TotalRows": 150,
    "SuccessCount": 148,
    "ErrorCount": 2
  },
  "CreatedAt": "2024-01-15T09:00:00.000Z",
  "ttl": 1713168000
}
```

---

## Compliance & Security

### What Gets Audited

| Category | Actions Logged |
|----------|---------------|
| Authentication | Login, logout, failed attempts, password changes |
| Authorization | Permission changes, role assignments |
| Data Changes | Create, update, delete on all entities |
| Data Access | Exports, bulk downloads, report generation |
| System | Configuration changes, bulk operations |

### What's NOT Logged

- Read-only queries (for performance)
- Health check endpoints
- Static asset requests
- Metrics/monitoring calls

### Security Considerations

1. **Immutability**: Audit logs cannot be modified after creation
2. **Access Control**: Only admins can view audit logs
3. **Data Masking**: Sensitive fields (passwords) are never logged
4. **Encryption**: Table encrypted at rest with AWS KMS

---

## Monitoring & Alerts

### Key Metrics

| Metric | Alert Threshold |
|--------|-----------------|
| Failed logins per user | > 5 in 10 minutes |
| Bulk deletes | > 100 items |
| Permission changes | Any change |
| After-hours activity | Outside business hours |

### CloudWatch Alarms

```yaml
# Example CloudWatch alarm for failed logins
FailedLoginAlarm:
  Type: AWS::CloudWatch::Alarm
  Properties:
    AlarmName: HighFailedLogins
    MetricName: FailedLoginCount
    Namespace: Handloom/Audit
    Statistic: Sum
    Period: 600
    EvaluationPeriods: 1
    Threshold: 5
    ComparisonOperator: GreaterThanThreshold
```

---

## Archival Strategy

### Short-Term (0-90 days)

- Stored in DynamoDB
- Fast query access
- Auto-deleted via TTL

### Long-Term (90+ days)

For compliance requiring longer retention:

1. **Export to S3**
   ```bash
   # Daily export job
   aws dynamodb export-table-to-s3 \
     --table-arn arn:aws:dynamodb:region:account:table/handloom-audit \
     --s3-bucket handloom-audit-archive \
     --export-format DYNAMODB_JSON
   ```

2. **S3 Lifecycle Rules**
   - Standard: 0-30 days
   - IA: 30-90 days
   - Glacier: 90+ days

3. **Query Archived Data**
   - Use Athena for ad-hoc queries
   - Use Glue for ETL to data warehouse
