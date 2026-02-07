# handloom-analytics Table

The analytics table stores pre-aggregated analytics data for dashboards and reporting.

## Table Configuration

```
Table Name: handloom-analytics
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |

---

## Design Philosophy

### Pre-Aggregated Data

The analytics table stores **pre-computed aggregations** rather than raw data. This enables:

- Fast dashboard loading
- Reduced query costs
- Consistent metrics across views
- Historical trend analysis

### Update Strategy

Analytics data is updated via:
1. **Real-time**: Critical metrics (e.g., daily revenue)
2. **Batch**: Periodic aggregations (e.g., hourly, daily)
3. **On-demand**: Report generation triggers

---

## Entities

### 1. Dashboard Stats

Real-time and daily aggregated statistics for the admin dashboard.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `DASHBOARD` | `DASHBOARD` |
| SK | `STATS#<date>` | `STATS#2024-01-15` |

For current/live stats:
| Key | Pattern | Example |
|-----|---------|---------|
| PK | `DASHBOARD` | `DASHBOARD` |
| SK | `STATS#CURRENT` | `STATS#CURRENT` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| Date | String | Yes | Date for stats (or "CURRENT") |
| Orders | Object | Yes | Order statistics |
| Revenue | Object | Yes | Revenue statistics |
| Customers | Object | Yes | Customer statistics |
| Products | Object | Yes | Product statistics |
| Inventory | Object | Yes | Inventory statistics |
| UpdatedAt | String | Yes | Last update timestamp |

#### Orders Structure

```json
{
  "Total": 150,
  "Pending": 12,
  "Processing": 25,
  "Shipped": 45,
  "Delivered": 60,
  "Cancelled": 8,
  "TodayCount": 15,
  "WeekCount": 85,
  "MonthCount": 320,
  "GrowthPercent": 12.5
}
```

#### Revenue Structure

```json
{
  "Today": 15000000,
  "Yesterday": 12500000,
  "ThisWeek": 85000000,
  "LastWeek": 78000000,
  "ThisMonth": 320000000,
  "LastMonth": 290000000,
  "GrowthPercent": 10.3,
  "Currency": "INR"
}
```

#### Customers Structure

```json
{
  "Total": 5420,
  "Active": 4200,
  "NewToday": 15,
  "NewThisWeek": 85,
  "NewThisMonth": 320,
  "GrowthPercent": 8.2
}
```

#### Products Structure

```json
{
  "Total": 1250,
  "Active": 1100,
  "Draft": 100,
  "Inactive": 50,
  "NewThisWeek": 25
}
```

#### Inventory Structure

```json
{
  "TotalValue": 45000000000,
  "LowStockCount": 45,
  "OutOfStockCount": 12,
  "Currency": "INR"
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get current stats | PK = `DASHBOARD`, SK = `STATS#CURRENT` |
| Get stats for date | PK = `DASHBOARD`, SK = `STATS#2024-01-15` |
| Get stats range | PK = `DASHBOARD`, SK between `STATS#2024-01-01` and `STATS#2024-01-31` |

---

### 2. Sales Analytics

Detailed sales analytics by time period.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `SALES` | `SALES` |
| SK | `<period>#<date>` | `DAILY#2024-01-15` |

#### Period Types

| Period | SK Pattern | Example |
|--------|------------|---------|
| Hourly | `HOURLY#<datetime>` | `HOURLY#2024-01-15T14:00:00Z` |
| Daily | `DAILY#<date>` | `DAILY#2024-01-15` |
| Weekly | `WEEKLY#<year>-W<week>` | `WEEKLY#2024-W03` |
| Monthly | `MONTHLY#<year>-<month>` | `MONTHLY#2024-01` |
| Yearly | `YEARLY#<year>` | `YEARLY#2024` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| Period | String | Yes | Period type |
| Date | String | Yes | Period identifier |
| OrderCount | Number | Yes | Number of orders |
| Revenue | Number | Yes | Total revenue in paise |
| AverageOrderValue | Number | Yes | AOV in paise |
| ItemsSold | Number | Yes | Total items sold |
| UniqueCustomers | Number | Yes | Unique customers |
| NewCustomers | Number | Yes | New customers |
| ReturningCustomers | Number | Yes | Returning customers |
| TopProducts | List[Object] | No | Top selling products |
| TopCategories | List[Object] | No | Top selling categories |
| PaymentMethods | Map | No | Breakdown by payment method |
| Regions | Map | No | Breakdown by region |
| UpdatedAt | String | Yes | Last update timestamp |

#### TopProducts Structure

```json
[
  {
    "ProductID": "prod-001",
    "ProductName": "Red Silk Saree",
    "SKU": "HL-SAR-001",
    "Quantity": 45,
    "Revenue": 6750000
  }
]
```

#### TopCategories Structure

```json
[
  {
    "CategoryID": "cat-001",
    "CategoryName": "Silk Sarees",
    "OrderCount": 120,
    "Revenue": 18000000
  }
]
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get daily sales | PK = `SALES`, SK = `DAILY#2024-01-15` |
| Get daily sales range | PK = `SALES`, SK between `DAILY#2024-01-01` and `DAILY#2024-01-31` |
| Get monthly sales | PK = `SALES`, SK = `MONTHLY#2024-01` |

---

### 3. Customer Analytics

Customer behavior and segmentation analytics.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMERS` | `CUSTOMERS` |
| SK | `<period>#<date>` | `MONTHLY#2024-01` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| Period | String | Yes | Period type |
| Date | String | Yes | Period identifier |
| TotalCustomers | Number | Yes | Total customers |
| ActiveCustomers | Number | Yes | Active customers |
| NewCustomers | Number | Yes | New acquisitions |
| ChurnedCustomers | Number | Yes | Churned customers |
| RetentionRate | Number | Yes | Retention percentage |
| AverageLifetimeValue | Number | Yes | CLV in paise |
| TopCustomers | List[Object] | No | Highest value customers |
| SegmentBreakdown | Map | No | Customer segments |
| AcquisitionChannels | Map | No | Acquisition sources |
| UpdatedAt | String | Yes | Last update timestamp |

#### TopCustomers Structure

```json
[
  {
    "CustomerID": "cust-001",
    "CustomerName": "John Doe",
    "TotalOrders": 25,
    "TotalSpent": 450000,
    "LastOrderDate": "2024-01-10"
  }
]
```

#### SegmentBreakdown Structure

```json
{
  "VIP": {
    "Count": 150,
    "Revenue": 45000000
  },
  "Regular": {
    "Count": 2500,
    "Revenue": 125000000
  },
  "Occasional": {
    "Count": 1800,
    "Revenue": 36000000
  },
  "AtRisk": {
    "Count": 450,
    "Revenue": 0
  }
}
```

---

### 4. Inventory Analytics

Inventory movement and valuation analytics.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `INVENTORY_ANALYTICS` | `INVENTORY_ANALYTICS` |
| SK | `<period>#<date>` | `DAILY#2024-01-15` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| Period | String | Yes | Period type |
| Date | String | Yes | Period identifier |
| TotalProducts | Number | Yes | Total SKUs |
| TotalQuantity | Number | Yes | Total units |
| TotalValue | Number | Yes | Inventory value in paise |
| LowStockProducts | Number | Yes | Below threshold count |
| OutOfStockProducts | Number | Yes | Zero stock count |
| TopMovingProducts | List[Object] | No | Fast-moving items |
| SlowMovingProducts | List[Object] | No | Slow-moving items |
| StockTurnover | Number | No | Turnover ratio |
| DaysOfInventory | Number | No | Average days to sell |
| UpdatedAt | String | Yes | Last update timestamp |

#### TopMovingProducts Structure

```json
[
  {
    "ProductID": "prod-001",
    "ProductName": "Red Silk Saree",
    "SKU": "HL-SAR-001",
    "UnitsSold": 150,
    "TurnoverRate": 4.5
  }
]
```

#### SlowMovingProducts Structure

```json
[
  {
    "ProductID": "prod-050",
    "ProductName": "Green Cotton Stole",
    "SKU": "HL-STL-050",
    "DaysInStock": 120,
    "LastSoldDate": "2023-10-15"
  }
]
```

---

### 5. Category Performance

Category-level performance metrics.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CATEGORY_PERF` | `CATEGORY_PERF` |
| SK | `<period>#<date>#<category_id>` | `MONTHLY#2024-01#cat-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| CategoryID | String | Yes | Category ID |
| CategoryName | String | Yes | Category name |
| Period | String | Yes | Period type |
| Date | String | Yes | Period identifier |
| ProductCount | Number | Yes | Products in category |
| OrderCount | Number | Yes | Orders containing category |
| UnitsSold | Number | Yes | Units sold |
| Revenue | Number | Yes | Revenue in paise |
| AverageOrderValue | Number | Yes | AOV for category |
| ConversionRate | Number | No | View to purchase rate |
| ReturnRate | Number | No | Return percentage |
| UpdatedAt | String | Yes | Last update timestamp |

---

### 6. Artisan Performance

Artisan-level performance metrics.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ARTISAN_PERF` | `ARTISAN_PERF` |
| SK | `<period>#<date>#<artisan_id>` | `MONTHLY#2024-01#art-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ArtisanID | String | Yes | Artisan ID |
| ArtisanName | String | Yes | Artisan name |
| Period | String | Yes | Period type |
| Date | String | Yes | Period identifier |
| ProductCount | Number | Yes | Active products |
| OrderCount | Number | Yes | Orders fulfilled |
| UnitsSold | Number | Yes | Units sold |
| Revenue | Number | Yes | Revenue generated |
| Earnings | Number | Yes | Artisan earnings |
| AverageRating | Number | No | Customer rating |
| ReturnRate | Number | No | Return percentage |
| UpdatedAt | String | Yes | Last update timestamp |

---

## Data Aggregation Jobs

### Real-Time Updates

Updated immediately on events:
- `DASHBOARD#STATS#CURRENT`: Order placed, payment received
- Order status changes

### Hourly Jobs

```
Schedule: Every hour at :00
Updates:
  - SALES#HOURLY#<current_hour>
  - DASHBOARD#STATS#CURRENT (refresh)
```

### Daily Jobs

```
Schedule: 00:30 UTC daily
Updates:
  - SALES#DAILY#<yesterday>
  - CUSTOMERS#DAILY#<yesterday>
  - INVENTORY_ANALYTICS#DAILY#<yesterday>
  - DASHBOARD#STATS#<yesterday>
  - CATEGORY_PERF#DAILY#<yesterday>#*
  - ARTISAN_PERF#DAILY#<yesterday>#*
```

### Weekly Jobs

```
Schedule: Monday 01:00 UTC
Updates:
  - SALES#WEEKLY#<last_week>
  - CUSTOMERS#WEEKLY#<last_week>
```

### Monthly Jobs

```
Schedule: 1st of month, 02:00 UTC
Updates:
  - SALES#MONTHLY#<last_month>
  - CUSTOMERS#MONTHLY#<last_month>
  - INVENTORY_ANALYTICS#MONTHLY#<last_month>
  - CATEGORY_PERF#MONTHLY#<last_month>#*
  - ARTISAN_PERF#MONTHLY#<last_month>#*
```

---

## Common Queries

### Dashboard Data

```
# Get current stats
PK = "DASHBOARD", SK = "STATS#CURRENT"

# Get last 30 days stats
PK = "DASHBOARD", SK between "STATS#2024-01-01" and "STATS#2024-01-30"
```

### Sales Trends

```
# Daily sales for a month
PK = "SALES", SK begins_with "DAILY#2024-01"

# Monthly sales for a year
PK = "SALES", SK begins_with "MONTHLY#2024"
```

### Category Comparison

```
# All category performance for a month
PK = "CATEGORY_PERF", SK begins_with "MONTHLY#2024-01#"
```

---

## Data Retention

| Data Type | Retention | Archive |
|-----------|-----------|---------|
| Hourly | 7 days | Delete |
| Daily | 90 days | S3 Glacier |
| Weekly | 2 years | S3 Glacier |
| Monthly | 7 years | S3 Glacier |
| Yearly | Forever | S3 Glacier |

---

## Integration with Reporting

### Report Generation

Reports query analytics data for fast generation:

```
1. User requests "Monthly Sales Report"
2. System queries SALES#MONTHLY#<month>
3. Pre-aggregated data returned instantly
4. Report formatted and delivered
```

### Custom Reports

For custom date ranges:
1. Query relevant period data
2. Aggregate in application layer
3. Cache results if frequently requested

---

## Future Enhancements

### Planned Analytics

- [ ] Geographic heatmaps
- [ ] Product recommendation data
- [ ] Customer journey analytics
- [ ] Real-time inventory alerts
- [ ] Predictive analytics (ML)

### Potential GSIs

| GSI | Purpose |
|-----|---------|
| GSI2 | Category-based queries |
| GSI3 | Artisan-based queries |
| GSI4 | Time-series optimization |
