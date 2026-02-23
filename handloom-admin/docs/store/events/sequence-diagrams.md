# Store Events - Sequence Diagrams

## Overview

This document contains sequence diagrams for the B2C Store Events (Tracking) service, covering the event ingestion flow and the daily aggregation cron.

---

## 1. Event Ingestion Sequence

```
+--------+   +-------------+   +-------------+   +---------------+   +-----------------+
| Browser|   |EventsHandler|   |EventsHandler|   |EventsRepository|   |AnalyticsRepository|
+---+----+   +------+------+   +------+------+   +-------+-------+   +--------+--------+
    |                |                |                  |                     |
    | POST           |                |                  |                     |
    | /api/v1/store/ |                |                  |                     |
    | events         |                |                  |                     |
    | {events: [...]}|                |                  |                     |
    |--------------->|                |                  |                     |
    |                |                |                  |                     |
    |                | Validate batch |                  |                     |
    |                | (struct        |                  |                     |
    |                |  validation)   |                  |                     |
    |                |------+         |                  |                     |
    |                |      |         |                  |                     |
    |                |<-----+         |                  |                     |
    |                |                |                  |                     |
    |                | Filter:        |                  |                     |
    |                | - Remove >24h  |                  |                     |
    |                |   old events   |                  |                     |
    |                | - Remove       |                  |                     |
    |                |   invalid types|                  |                     |
    |                |------+         |                  |                     |
    |                |      |         |                  |                     |
    |                |<-----+         |                  |                     |
    |                |                |                  |                     |
    |                | BatchWrite     |                  |                     |
    |                | Events         |                  |                     |
    |                | (valid events) |                  |                     |
    |                |---------------------------------->|                     |
    |                |                |                  |                     |
    |                |                |                  | BatchWriteItem      |
    |                |                |                  | (chunked, 25 max)   |
    |                |                |                  |------+              |
    |                |                |                  |      | DynamoDB     |
    |                |                |                  |      | (events)     |
    |                |                |                  |<-----+              |
    |                |                |                  |                     |
    |                |                |                  | Retry unprocessed   |
    |                |                |                  | items (if any)      |
    |                |                |                  |------+              |
    |                |                |                  |      |              |
    |                |                |                  |<-----+              |
    |                |                |                  |                     |
    |                | Success        |                  |                     |
    |                | (best-effort)  |                  |                     |
    |                |<----------------------------------|                     |
    |                |                |                  |                     |
    |                | --- For each accepted event with counter mapping ---   |
    |                |                |                  |                     |
    |                | IncrementDashboardCounter         |                     |
    |                | (field, count) |                  |                     |
    |                |----------------------------------------------->|        |
    |                |                |                  |                     |
    |                |                |                  |            UpdateItem|
    |                |                |                  |            ADD on   |
    |                |                |                  |            DASHBOARD|
    |                |                |                  |            #CURRENT |
    |                |                |                  |             +-------|
    |                |                |                  |             |       |
    |                |                |                  |             +------>|
    |                |                |                  |                     |
    |                | Success        |                  |                     |
    |                | (best-effort)  |                  |                     |
    |                |<-----------------------------------------------|        |
    |                |                |                  |                     |
    |                | --- End loop --|------------------|---------------------|
    |                |                |                  |                     |
    | 202 Accepted   |                |                  |                     |
    | {"accepted": N}|                |                  |                     |
    |<---------------|                |                  |                     |
    |                |                |                  |                     |
```

---

## 2. Daily Aggregation Sequence

```
+-----------+   +------------------+   +--------------------+   +---------------+   +-------------------+
|EventBridge|   |worker-analytics  |   |AnalyticsAggregator |   |EventsRepository|   |AnalyticsRepository|
+-----+-----+   +--------+---------+   +----------+---------+   +-------+-------+   +---------+---------+
      |                   |                       |                     |                      |
      | Scheduled event   |                       |                     |                      |
      | (daily cron)      |                       |                     |                      |
      |------------------>|                       |                     |                      |
      |                   |                       |                     |                      |
      |                   | AggregateDate         |                     |                      |
      |                   | (yesterday)           |                     |                      |
      |                   |---------------------->|                     |                      |
      |                   |                       |                     |                      |
      |                   |                       | QueryByDate         |                      |
      |                   |                       | (yesterday)         |                      |
      |                   |                       |-------------------->|                      |
      |                   |                       |                     |                      |
      |                   |                       |                     | Query                |
      |                   |                       |                     | PK=EVENT#YYYY-MM-DD  |
      |                   |                       |                     | (paginated)          |
      |                   |                       |                     |------+               |
      |                   |                       |                     |      | DynamoDB      |
      |                   |                       |                     |      | (events)      |
      |                   |                       |                     |<-----+               |
      |                   |                       |                     |                      |
      |                   |                       | Raw events          |                      |
      |                   |                       |<--------------------|                      |
      |                   |                       |                     |                      |
      |                   |                       | Compute 5           |                      |
      |                   |                       | aggregates:         |                      |
      |                   |                       | - funnel             |                      |
      |                   |                       | - revenue            |                      |
      |                   |                       | - customers          |                      |
      |                   |                       | - engagement         |                      |
      |                   |                       | - products           |                      |
      |                   |                       |------+              |                      |
      |                   |                       |      |              |                      |
      |                   |                       |<-----+              |                      |
      |                   |                       |                     |                      |
      |                   |                       | PutDailyAggregate   |                      |
      |                   |                       | (x5 aggregate types)|                      |
      |                   |                       |--------------------------------------------->|
      |                   |                       |                     |                      |
      |                   |                       | Success             |                      |
      |                   |                       |<---------------------------------------------|
      |                   |                       |                     |                      |
      |                   |                       | PutDailyStats       |                      |
      |                   |                       | (yesterday,         |                      |
      |                   |                       |  counters)          |                      |
      |                   |                       |--------------------------------------------->|
      |                   |                       |                     |                      |
      |                   |                       | Success             |                      |
      |                   |                       |<---------------------------------------------|
      |                   |                       |                     |                      |
      |                   |                       | ResetDashboard      |                      |
      |                   |                       | Current()           |                      |
      |                   |                       |--------------------------------------------->|
      |                   |                       |                     |                      |
      |                   |                       |                     |            UpdateItem |
      |                   |                       |                     |            SET all    |
      |                   |                       |                     |            counters=0 |
      |                   |                       |                     |            on DASHBOARD|
      |                   |                       |                     |            #CURRENT   |
      |                   |                       |                     |              +--------|
      |                   |                       |                     |              |        |
      |                   |                       |                     |              +------->|
      |                   |                       |                     |                      |
      |                   |                       | Success             |                      |
      |                   |                       |<---------------------------------------------|
      |                   |                       |                     |                      |
      |                   | Aggregation complete  |                     |                      |
      |                   |<----------------------|                     |                      |
      |                   |                       |                     |                      |
```
