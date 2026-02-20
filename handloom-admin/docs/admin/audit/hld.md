# Audit Lambda - High-Level Design (HLD)

## 1. Overview

The Audit Lambda service provides comprehensive audit logging capabilities for the Handloom Admin system. It tracks all significant actions performed by users, capturing who did what, when, and what changed. This enables compliance tracking, security monitoring, and troubleshooting capabilities.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      AUDIT LOGGING SYSTEM                                    │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   React Frontend  │
                                    │   (Admin Portal)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway     │
                                    │   (REST API)      │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │  Audit Lambda   │  │  Other Lambdas  │  │  Audit Lambda   │
              │  (Read API)     │  │  (Log Writers)  │  │  (Query API)    │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                         ┌──────────────────┼──────────────────┐
                         │                  │                  │
                         ▼                  ▼                  ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   CloudWatch    │  │   S3 (Archive)  │
              │   (Audit Table) │  │   (Metrics)     │  │   (Long-term)   │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Audit Handler

```
┌─────────────────────────────────────────────────────────────────────┐
│                         AUDIT HANDLER                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   List      │  │  GetByID    │  │ GetByEntity │  │ GetByUser  │ │
│  │   Handler   │  │  Handler    │  │   Handler   │  │  Handler   │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │               │         │
│         └────────────────┼────────────────┼───────────────┘         │
│                          │                │                          │
│                          ▼                ▼                          │
│                   ┌─────────────────────────────┐                   │
│                   │       Audit Service         │                   │
│                   │  - Log()                    │                   │
│                   │  - GetByID()                │                   │
│                   │  - List()                   │                   │
│                   │  - GetByEntity()            │                   │
│                   │  - GetByUser()              │                   │
│                   └──────────────┬──────────────┘                   │
│                                  │                                   │
│                                  ▼                                   │
│                        ┌───────────────┐                            │
│                        │ Audit         │                            │
│                        │ Repository    │                            │
│                        └───────────────┘                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Audit Log Entry Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                         AUDIT LOG ENTRY                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Identification:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ id           │ audit_abc12345                                 │  │
│  │ created_at   │ 2024-01-15T10:30:00Z                          │  │
│  │ ttl          │ 1713178200 (90 days from creation)            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Actor Information:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ user_id      │ user_xyz789                                    │  │
│  │ user_email   │ admin@example.com                              │  │
│  │ user_role    │ admin                                          │  │
│  │ ip_address   │ 192.168.1.100                                  │  │
│  │ user_agent   │ Mozilla/5.0...                                 │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Action Details:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ action       │ UPDATE                                         │  │
│  │ entity_type  │ ORDER                                          │  │
│  │ entity_id    │ order_123456                                   │  │
│  │ request_id   │ req_abcdef                                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Changes:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ changes: {                                                    │  │
│  │   "status": {                                                 │  │
│  │     "old_value": "pending",                                   │  │
│  │     "new_value": "shipped"                                    │  │
│  │   },                                                          │  │
│  │   "tracking_number": {                                        │  │
│  │     "old_value": null,                                        │  │
│  │     "new_value": "TRK123456"                                  │  │
│  │   }                                                           │  │
│  │ }                                                             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-audit                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  AUDIT LOG RECORDS                                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ PK: AUDIT#<date>  (e.g., AUDIT#2024-01-15)                  │    │
│  │ SK: <time>#<audit_id>  (e.g., 10:30:00.000Z#audit_abc123)   │    │
│  │                                                             │    │
│  │ Attributes:                                                 │    │
│  │   - id                                                      │    │
│  │   - user_id                                                 │    │
│  │   - user_email                                              │    │
│  │   - user_role                                               │    │
│  │   - action (CREATE, UPDATE, DELETE, LOGIN, LOGOUT)          │    │
│  │   - entity_type_audit (ORDER, PRODUCT, USER, etc.)          │    │
│  │   - entity_id                                               │    │
│  │   - changes (map of field changes)                          │    │
│  │   - old_values                                              │    │
│  │   - new_values                                              │    │
│  │   - ip_address                                              │    │
│  │   - user_agent                                              │    │
│  │   - request_id                                              │    │
│  │   - created_at                                              │    │
│  │   - ttl (90 days expiry)                                    │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  GSI1: Entity Index (Query by Entity)                                │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ GSI1PK: <entity_type>#<entity_id>                           │    │
│  │         (e.g., ORDER#order_123)                             │    │
│  │ GSI1SK: <timestamp>                                         │    │
│  │         (e.g., 2024-01-15T10:30:00Z)                        │    │
│  │                                                             │    │
│  │ Use case: Get complete history of changes for an entity     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  GSI2: User Index (Query by User)                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ GSI2PK: USER#<user_id>                                      │    │
│  │         (e.g., USER#user_456)                               │    │
│  │ GSI2SK: <timestamp>                                         │    │
│  │         (e.g., 2024-01-15T10:30:00Z)                        │    │
│  │                                                             │    │
│  │ Use case: Get all actions performed by a specific user      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Key Design Rationale

```
┌─────────────────────────────────────────────────────────────────────┐
│                      KEY DESIGN DECISIONS                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Date-Based Partitioning:                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK = AUDIT#<date>                                             │  │
│  │                                                               │  │
│  │ Benefits:                                                     │  │
│  │ • Even distribution of data across partitions                │  │
│  │ • Efficient queries for specific date ranges                 │  │
│  │ • Natural archiving boundary                                 │  │
│  │ • Prevents hot partition issues                              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Time-Based Sort Key:                                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ SK = <time>#<audit_id>                                        │  │
│  │                                                               │  │
│  │ Benefits:                                                     │  │
│  │ • Chronological ordering within each day                     │  │
│  │ • Unique identification with audit_id suffix                 │  │
│  │ • Efficient range queries for time windows                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  TTL for Automatic Cleanup:                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ TTL = created_at + 90 days                                    │  │
│  │                                                               │  │
│  │ Benefits:                                                     │  │
│  │ • Automatic data lifecycle management                        │  │
│  │ • Cost optimization                                          │  │
│  │ • Compliance with retention policies                         │  │
│  │ • No manual cleanup required                                 │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Audit Actions

### 5.1 Supported Actions

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AUDIT ACTIONS                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Data Modification Actions:                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ CREATE  │ New entity created                                  │  │
│  │         │ - Full entity stored in new_values                  │  │
│  │         │ - old_values is null                                │  │
│  ├─────────┼─────────────────────────────────────────────────────┤  │
│  │ UPDATE  │ Existing entity modified                            │  │
│  │         │ - changes map shows field-level diffs               │  │
│  │         │ - Both old and new values captured                  │  │
│  ├─────────┼─────────────────────────────────────────────────────┤  │
│  │ DELETE  │ Entity removed                                      │  │
│  │         │ - Full entity stored in old_values                  │  │
│  │         │ - new_values is null                                │  │
│  └─────────┴─────────────────────────────────────────────────────┘  │
│                                                                      │
│  Authentication Actions:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ LOGIN   │ User successfully authenticated                     │  │
│  │         │ - IP address and user agent captured                │  │
│  │         │ - Session information logged                        │  │
│  ├─────────┼─────────────────────────────────────────────────────┤  │
│  │ LOGOUT  │ User session ended                                  │  │
│  │         │ - Session duration can be calculated                │  │
│  │         │ - Logout method (manual/timeout) captured           │  │
│  └─────────┴─────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 Audited Entity Types

```
┌─────────────────────────────────────────────────────────────────────┐
│                    AUDITED ENTITIES                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┬───────────────────────────────────────────────┐    │
│  │ Entity Type │ Audited Operations                            │    │
│  ├─────────────┼───────────────────────────────────────────────┤    │
│  │ ORDER       │ Create, Update status, Cancel, Refund         │    │
│  │ PRODUCT     │ Create, Update, Delete, Inventory changes     │    │
│  │ CATEGORY    │ Create, Update, Delete                        │    │
│  │ DESIGN      │ Create, Update, Delete                        │    │
│  │ USER        │ Create, Update, Deactivate, Role changes      │    │
│  │ ARTISAN     │ Create, Update, Delete                        │    │
│  │ COUPON      │ Create, Update, Deactivate                    │    │
│  │ PRICING     │ Rule create, Update, Delete                   │    │
│  │ INVENTORY   │ Stock adjustments, Reservations               │    │
│  │ CUSTOMER    │ Profile updates                               │    │
│  │ SETTINGS    │ Configuration changes                         │    │
│  └─────────────┴───────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. API Endpoints

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AUDIT API ENDPOINTS                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  List Audit Logs (with filters):                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/audit                                              │  │
│  │                                                               │  │
│  │ Query Parameters:                                             │  │
│  │   - action: Filter by action type (CREATE, UPDATE, etc.)     │  │
│  │   - entity_type: Filter by entity type (ORDER, PRODUCT)      │  │
│  │   - entity_id: Filter by specific entity                     │  │
│  │   - user_id: Filter by user who performed action             │  │
│  │   - page: Page number (default: 1)                           │  │
│  │   - per_page: Items per page (default: 20)                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Get Single Audit Log:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/audit/{id}                                         │  │
│  │                                                               │  │
│  │ Returns complete audit log entry with all changes             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Get Entity History:                                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/audit/entity/{type}/{id}                           │  │
│  │                                                               │  │
│  │ Returns chronological history of all changes to an entity     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Get User Activity:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/audit/user/{id}                                    │  │
│  │                                                               │  │
│  │ Returns all actions performed by a specific user              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Security Design

### 7.1 Access Control

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AUDIT ACCESS CONTROL                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Read Permissions:                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Role        │ Access Level                                    │  │
│  ├─────────────┼─────────────────────────────────────────────────┤  │
│  │ Admin       │ Full read access to all audit logs              │  │
│  │ Manager     │ No access                                       │  │
│  │ Staff       │ No access                                       │  │
│  └─────────────┴─────────────────────────────────────────────────┘  │
│                                                                      │
│  Write Permissions:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Audit logs are write-only from application perspective      │  │
│  │ • No API endpoint for manual audit log creation               │  │
│  │ • Logs created automatically by services                      │  │
│  │ • No update or delete operations allowed                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Data Protection:                                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Audit logs are append-only (immutable)                      │  │
│  │ • No user can modify historical audit records                 │  │
│  │ • TTL-based deletion only (automatic after 90 days)           │  │
│  │ • Encryption at rest using AWS KMS                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 7.2 Sensitive Data Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                   SENSITIVE DATA HANDLING                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Fields NOT Logged:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Passwords and password hashes                               │  │
│  │ • API keys and tokens                                         │  │
│  │ • Credit card numbers                                         │  │
│  │ • Full social security numbers                                │  │
│  │ • Raw authentication credentials                              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Fields Masked/Redacted:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Email addresses (partial masking for non-admin views)       │  │
│  │ • Phone numbers (last 4 digits only)                          │  │
│  │ • Bank account numbers (masked)                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Data Retention & Compliance

```
┌─────────────────────────────────────────────────────────────────────┐
│                    DATA RETENTION POLICY                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Retention Timeline:                                                 │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                                                             │    │
│  │  Day 0         Day 30        Day 60        Day 90           │    │
│  │    │             │             │             │              │    │
│  │    ▼             ▼             ▼             ▼              │    │
│  │  ┌───┐         ┌───┐         ┌───┐         ┌───┐           │    │
│  │  │ + │ ─────── │ . │ ─────── │ . │ ─────── │ X │           │    │
│  │  └───┘         └───┘         └───┘         └───┘           │    │
│  │  Created       Active        Active        TTL Expires     │    │
│  │                                            Auto-deleted    │    │
│  │                                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Archival Strategy (Future Enhancement):                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Before TTL expiry, archive to S3 (Glacier)                  │  │
│  │ • Compressed JSON format                                      │  │
│  │ • Long-term retention: 7 years (compliance)                   │  │
│  │ • Query via Athena for historical analysis                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. Error Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Query Errors:                                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ AUD001 │ Audit log not found                                │    │
│  │ AUD002 │ Invalid audit log ID format                        │    │
│  │ AUD003 │ Invalid entity type                                │    │
│  │ AUD004 │ Invalid action filter                              │    │
│  │ AUD005 │ Invalid date range                                 │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Internal Errors:                                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ AUD101 │ Failed to create audit log                         │    │
│  │ AUD102 │ Database query failed                              │    │
│  │ AUD103 │ Failed to serialize audit data                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Authorization Errors:                                               │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ AUD201 │ Insufficient permissions to view audit logs        │    │
│  │ AUD202 │ Cannot access audit logs for other users           │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 10. Monitoring & Observability

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MONITORING DESIGN                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Metrics:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Audit logs created per minute/hour                          │  │
│  │ • Audit log query latency                                     │  │
│  │ • TTL deletions per day                                       │  │
│  │ • Storage utilization                                         │  │
│  │ • Query patterns by entity type                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Alerts:                                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Unusually high audit log volume (potential attack)          │  │
│  │ • Audit log creation failures                                 │  │
│  │ • Suspicious deletion patterns                                │  │
│  │ • Unauthorized access attempts                                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Dashboards:                                                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Activity by action type (CREATE/UPDATE/DELETE)              │  │
│  │ • Most active users                                           │  │
│  │ • Most modified entities                                      │  │
│  │ • Login/logout patterns                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 11. Scalability Considerations

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SCALABILITY DESIGN                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Lambda Configuration:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Memory: 256 MB                                              │  │
│  │ • Timeout: 30 seconds                                         │  │
│  │ • Concurrent executions: 50 (reserved)                        │  │
│  │ • Fire-and-forget logging (async)                             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  DynamoDB Configuration:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Separate audit table (not main table)                       │  │
│  │ • On-demand capacity mode                                     │  │
│  │ • GSI for entity and user queries                             │  │
│  │ • TTL enabled for automatic cleanup                           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Write Optimization:                                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Asynchronous logging (non-blocking)                         │  │
│  │ • Batch writes for bulk operations                            │  │
│  │ • Retry with exponential backoff                              │  │
│  │ • Dead letter queue for failed writes                         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 12. Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                      DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  External Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • AWS DynamoDB - Audit log storage                            │  │
│  │ • AWS CloudWatch - Logging & monitoring                       │  │
│  │ • AWS S3 - Long-term archival (future)                        │  │
│  │ • AWS KMS - Encryption at rest                                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Internal Services (Consumers of Audit):                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Auth Lambda - Login/logout events                           │  │
│  │ • Order Lambda - Order lifecycle events                       │  │
│  │ • Catalog Lambda - Product/category changes                   │  │
│  │ • User Lambda - User management events                        │  │
│  │ • Inventory Lambda - Stock changes                            │  │
│  │ • Pricing Lambda - Price rule changes                         │  │
│  │ • Coupon Lambda - Coupon lifecycle events                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Libraries:                                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • aws-sdk-go-v2 - AWS service clients                         │  │
│  │ • google/uuid - Unique ID generation                          │  │
│  │ • go-chi/chi - HTTP routing                                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
