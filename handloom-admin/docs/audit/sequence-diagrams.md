# Audit Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for all audit logging operations.

---

## 1. Create Audit Log Sequence

```
┌────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│Service │          │Audit Service│          │Audit Repo    │          │ DynamoDB │
└───┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                      │                        │                       │
    │ Log(action, entity,  │                        │                       │
    │    userID, changes)  │                        │                       │
    │─────────────────────▶│                        │                       │
    │                      │                        │                       │
    │                      │  Generate audit ID     │                       │
    │                      │──────────┐             │                       │
    │                      │          │             │                       │
    │                      │◀─────────┘             │                       │
    │                      │                        │                       │
    │                      │  Build AuditLog        │                       │
    │                      │  - Set timestamps      │                       │
    │                      │  - Map field changes   │                       │
    │                      │  - Calculate TTL       │                       │
    │                      │──────────┐             │                       │
    │                      │          │             │                       │
    │                      │◀─────────┘             │                       │
    │                      │                        │                       │
    │                      │  Create(auditLog)      │                       │
    │                      │───────────────────────▶│                       │
    │                      │                        │                       │
    │                      │                        │  SetKeys()            │
    │                      │                        │  - PK: AUDIT#date     │
    │                      │                        │  - SK: time#id        │
    │                      │                        │  - GSI1: USER#userID  │
    │                      │                        │──────────┐            │
    │                      │                        │          │            │
    │                      │                        │◀─────────┘            │
    │                      │                        │                       │
    │                      │                        │  PutItem(log)         │
    │                      │                        │──────────────────────▶│
    │                      │                        │                       │
    │                      │                        │  Success              │
    │                      │                        │◀──────────────────────│
    │                      │                        │                       │
    │                      │  Success               │                       │
    │                      │◀───────────────────────│                       │
    │                      │                        │                       │
    │  Success             │                        │                       │
    │◀─────────────────────│                        │                       │
    │                      │                        │                       │
```

---

## 2. List Audit Logs Sequence

```
┌────────┐          ┌─────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │Audit Lambda  │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                         │                       │
    │ GET /admin/audit   │                      │                         │                       │
    │ ?action=UPDATE     │                      │                         │                       │
    │ &entity_type=ORDER │                      │                         │                       │
    │───────────────────▶│                      │                         │                       │
    │                    │                      │                         │                       │
    │                    │  Forward request     │                         │                       │
    │                    │─────────────────────▶│                         │                       │
    │                    │                      │                         │                       │
    │                    │                      │  Parse query params     │                       │
    │                    │                      │  - action               │                       │
    │                    │                      │  - entity_type          │                       │
    │                    │                      │  - entity_id            │                       │
    │                    │                      │  - user_id              │                       │
    │                    │                      │  - pagination           │                       │
    │                    │                      │──────────┐              │                       │
    │                    │                      │          │              │                       │
    │                    │                      │◀─────────┘              │                       │
    │                    │                      │                         │                       │
    │                    │                      │  List(request)          │                       │
    │                    │                      │────────────────────────▶│                       │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Build filter expr    │
    │                    │                      │                         │──────────┐            │
    │                    │                      │                         │          │            │
    │                    │                      │                         │◀─────────┘            │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Scan with filters    │
    │                    │                      │                         │──────────────────────▶│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Audit log items      │
    │                    │                      │                         │◀──────────────────────│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Apply pagination     │
    │                    │                      │                         │──────────┐            │
    │                    │                      │                         │          │            │
    │                    │                      │                         │◀─────────┘            │
    │                    │                      │                         │                       │
    │                    │                      │  {logs, pagination}     │                       │
    │                    │                      │◀────────────────────────│                       │
    │                    │                      │                         │                       │
    │                    │  {logs, pagination}  │                         │                       │
    │                    │◀─────────────────────│                         │                       │
    │                    │                      │                         │                       │
    │ 200 OK             │                      │                         │                       │
    │ {logs, pagination} │                      │                         │                       │
    │◀───────────────────│                      │                         │                       │
    │                    │                      │                         │                       │
```

---

## 3. Get Audit Log by ID Sequence

```
┌────────┐          ┌─────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │Audit Lambda  │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                         │                       │
    │ GET /admin/audit/  │                      │                         │                       │
    │     audit_abc123   │                      │                         │                       │
    │───────────────────▶│                      │                         │                       │
    │                    │                      │                         │                       │
    │                    │  Forward request     │                         │                       │
    │                    │─────────────────────▶│                         │                       │
    │                    │                      │                         │                       │
    │                    │                      │  Extract audit ID       │                       │
    │                    │                      │──────────┐              │                       │
    │                    │                      │          │              │                       │
    │                    │                      │◀─────────┘              │                       │
    │                    │                      │                         │                       │
    │                    │                      │  GetByID(id)            │                       │
    │                    │                      │────────────────────────▶│                       │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Query                │
    │                    │                      │                         │  PK: AUDIT#id        │
    │                    │                      │                         │──────────────────────▶│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Audit log item       │
    │                    │                      │                         │◀──────────────────────│
    │                    │                      │                         │                       │
    │                    │                      │  Audit log              │                       │
    │                    │                      │◀────────────────────────│                       │
    │                    │                      │                         │                       │
    │                    │  {audit_log}         │                         │                       │
    │                    │◀─────────────────────│                         │                       │
    │                    │                      │                         │                       │
    │ 200 OK             │                      │                         │                       │
    │ {audit_log}        │                      │                         │                       │
    │◀───────────────────│                      │                         │                       │
    │                    │                      │                         │                       │
```

---

## 4. Get Audit Logs by Entity Sequence

```
┌────────┐          ┌─────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │Audit Lambda  │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                         │                       │
    │ GET /admin/audit/  │                      │                         │                       │
    │   entity/ORDER/    │                      │                         │                       │
    │   order_123        │                      │                         │                       │
    │───────────────────▶│                      │                         │                       │
    │                    │                      │                         │                       │
    │                    │  Forward request     │                         │                       │
    │                    │─────────────────────▶│                         │                       │
    │                    │                      │                         │                       │
    │                    │                      │  Parse URL params       │                       │
    │                    │                      │  - entityType: ORDER    │                       │
    │                    │                      │  - entityID: order_123  │                       │
    │                    │                      │──────────┐              │                       │
    │                    │                      │          │              │                       │
    │                    │                      │◀─────────┘              │                       │
    │                    │                      │                         │                       │
    │                    │                      │  GetByEntity(type,id)   │                       │
    │                    │                      │────────────────────────▶│                       │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Query GSI1           │
    │                    │                      │                         │  GSI1PK: ORDER#       │
    │                    │                      │                         │          order_123    │
    │                    │                      │                         │  ScanForward: false   │
    │                    │                      │                         │──────────────────────▶│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Audit logs (sorted   │
    │                    │                      │                         │  by timestamp desc)   │
    │                    │                      │                         │◀──────────────────────│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Apply pagination     │
    │                    │                      │                         │──────────┐            │
    │                    │                      │                         │          │            │
    │                    │                      │                         │◀─────────┘            │
    │                    │                      │                         │                       │
    │                    │                      │  {logs, pagination}     │                       │
    │                    │                      │◀────────────────────────│                       │
    │                    │                      │                         │                       │
    │                    │  {logs, pagination}  │                         │                       │
    │                    │◀─────────────────────│                         │                       │
    │                    │                      │                         │                       │
    │ 200 OK             │                      │                         │                       │
    │ {entity_history}   │                      │                         │                       │
    │◀───────────────────│                      │                         │                       │
    │                    │                      │                         │                       │
```

---

## 5. Get Audit Logs by User Sequence

```
┌────────┐          ┌─────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │Audit Lambda  │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                         │                       │
    │ GET /admin/audit/  │                      │                         │                       │
    │   user/user_456    │                      │                         │                       │
    │───────────────────▶│                      │                         │                       │
    │                    │                      │                         │                       │
    │                    │  Forward request     │                         │                       │
    │                    │─────────────────────▶│                         │                       │
    │                    │                      │                         │                       │
    │                    │                      │  Extract user ID        │                       │
    │                    │                      │──────────┐              │                       │
    │                    │                      │          │              │                       │
    │                    │                      │◀─────────┘              │                       │
    │                    │                      │                         │                       │
    │                    │                      │  GetByUser(userID)      │                       │
    │                    │                      │────────────────────────▶│                       │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Query GSI2           │
    │                    │                      │                         │  GSI2PK: USER#        │
    │                    │                      │                         │          user_456     │
    │                    │                      │                         │  ScanForward: false   │
    │                    │                      │                         │──────────────────────▶│
    │                    │                      │                         │                       │
    │                    │                      │                         │  User's audit logs    │
    │                    │                      │                         │  (sorted by time)     │
    │                    │                      │                         │◀──────────────────────│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Apply pagination     │
    │                    │                      │                         │──────────┐            │
    │                    │                      │                         │          │            │
    │                    │                      │                         │◀─────────┘            │
    │                    │                      │                         │                       │
    │                    │                      │  {logs, pagination}     │                       │
    │                    │                      │◀────────────────────────│                       │
    │                    │                      │                         │                       │
    │                    │  {logs, pagination}  │                         │                       │
    │                    │◀─────────────────────│                         │                       │
    │                    │                      │                         │                       │
    │ 200 OK             │                      │                         │                       │
    │ {user_activity}    │                      │                         │                       │
    │◀───────────────────│                      │                         │                       │
    │                    │                      │                         │                       │
```

---

## 6. Automatic Audit Logging from Service Sequence

```
┌────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │Order Service │          │Audit Service │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └──────┬───────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                      │                         │                         │                       │
    │ Update order status  │                         │                         │                       │
    │─────────────────────▶│                         │                         │                       │
    │                      │                         │                         │                       │
    │                      │  Load existing order    │                         │                       │
    │                      │──────────┐              │                         │                       │
    │                      │          │              │                         │                       │
    │                      │◀─────────┘              │                         │                       │
    │                      │                         │                         │                       │
    │                      │  Calculate changes      │                         │                       │
    │                      │  - oldStatus: pending   │                         │                       │
    │                      │  - newStatus: shipped   │                         │                       │
    │                      │──────────┐              │                         │                       │
    │                      │          │              │                         │                       │
    │                      │◀─────────┘              │                         │                       │
    │                      │                         │                         │                       │
    │                      │  Update order in DB     │                         │                       │
    │                      │──────────┐              │                         │                       │
    │                      │          │              │                         │                       │
    │                      │◀─────────┘              │                         │                       │
    │                      │                         │                         │                       │
    │                      │  Log(UPDATE, ORDER,     │                         │                       │
    │                      │      orderID, userID,   │                         │                       │
    │                      │      changes)           │                         │                       │
    │                      │────────────────────────▶│                         │                       │
    │                      │                         │                         │                       │
    │                      │                         │  Build audit entry      │                       │
    │                      │                         │  - action: UPDATE       │                       │
    │                      │                         │  - entity: ORDER        │                       │
    │                      │                         │  - changes: status      │                       │
    │                      │                         │──────────┐              │                       │
    │                      │                         │          │              │                       │
    │                      │                         │◀─────────┘              │                       │
    │                      │                         │                         │                       │
    │                      │                         │  Create(auditLog)       │                       │
    │                      │                         │────────────────────────▶│                       │
    │                      │                         │                         │                       │
    │                      │                         │                         │  PutItem              │
    │                      │                         │                         │──────────────────────▶│
    │                      │                         │                         │                       │
    │                      │                         │                         │  Success              │
    │                      │                         │                         │◀──────────────────────│
    │                      │                         │                         │                       │
    │                      │                         │  Success                │                       │
    │                      │                         │◀────────────────────────│                       │
    │                      │                         │                         │                       │
    │                      │  Success                │                         │                       │
    │                      │◀────────────────────────│                         │                       │
    │                      │                         │                         │                       │
    │ Order updated        │                         │                         │                       │
    │◀─────────────────────│                         │                         │                       │
    │                      │                         │                         │                       │
```

---

## 7. TTL-Based Automatic Cleanup Sequence

```
┌──────────────┐          ┌──────────┐          ┌────────────┐
│  DynamoDB    │          │ TTL Svc  │          │ CloudWatch │
│  (Audit Tbl) │          │          │          │  (Metrics) │
└──────┬───────┘          └────┬─────┘          └─────┬──────┘
       │                       │                      │
       │  TTL expiry check     │                      │
       │  (runs periodically)  │                      │
       │◀──────────────────────│                      │
       │                       │                      │
       │  Scan for expired     │                      │
       │  items (TTL < now)    │                      │
       │──────────┐            │                      │
       │          │            │                      │
       │◀─────────┘            │                      │
       │                       │                      │
       │  Delete expired       │                      │
       │  audit logs           │                      │
       │  (90+ days old)       │                      │
       │──────────┐            │                      │
       │          │            │                      │
       │◀─────────┘            │                      │
       │                       │                      │
       │  Deletion stream      │                      │
       │  event                │                      │
       │──────────────────────▶│                      │
       │                       │                      │
       │                       │  Emit metrics        │
       │                       │  - records_expired   │
       │                       │  - cleanup_count     │
       │                       │─────────────────────▶│
       │                       │                      │
       │                       │                      │  Log metrics
       │                       │                      │──────────┐
       │                       │                      │          │
       │                       │                      │◀─────────┘
       │                       │                      │
```

---

## 8. Error Handling - Audit Log Not Found Sequence

```
┌────────┐          ┌─────────┐          ┌──────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │Audit Lambda  │          │ Audit Repo   │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬───────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                         │                       │
    │ GET /admin/audit/  │                      │                         │                       │
    │     invalid_id     │                      │                         │                       │
    │───────────────────▶│                      │                         │                       │
    │                    │                      │                         │                       │
    │                    │  Forward request     │                         │                       │
    │                    │─────────────────────▶│                         │                       │
    │                    │                      │                         │                       │
    │                    │                      │  GetByID(invalid_id)    │                       │
    │                    │                      │────────────────────────▶│                       │
    │                    │                      │                         │                       │
    │                    │                      │                         │  Query                │
    │                    │                      │                         │  PK: AUDIT#invalid_id │
    │                    │                      │                         │──────────────────────▶│
    │                    │                      │                         │                       │
    │                    │                      │                         │  Empty result         │
    │                    │                      │                         │  (0 items)            │
    │                    │                      │                         │◀──────────────────────│
    │                    │                      │                         │                       │
    │                    │                      │  NotFoundError          │                       │
    │                    │                      │  "Audit log not found"  │                       │
    │                    │                      │◀────────────────────────│                       │
    │                    │                      │                         │                       │
    │                    │  404 Not Found       │                         │                       │
    │                    │◀─────────────────────│                         │                       │
    │                    │                      │                         │                       │
    │ 404 Not Found      │                      │                         │                       │
    │ {error: "Audit     │                      │                         │                       │
    │  log not found"}   │                      │                         │                       │
    │◀───────────────────│                      │                         │                       │
    │                    │                      │                         │                       │
```
