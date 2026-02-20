# Analytics Lambda - High Level Design

## 1. Overview

The Analytics Lambda provides comprehensive business intelligence and reporting capabilities for the Handloom Admin platform. It aggregates data from orders, products, customers, and inventory to deliver actionable insights through dashboard statistics, sales analytics, and performance metrics.

### Key Features
- Real-time dashboard statistics
- Sales analytics with date range filtering
- Top products and categories analysis
- Customer behavior insights
- Inventory health analytics
- Export capabilities for reports

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ANALYTICS LAMBDA ARCHITECTURE                      │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Client     │
                              │  (Browser)   │
                              └──────┬───────┘
                                     │
                                     ▼
                              ┌──────────────┐
                              │  CloudFront  │
                              │     CDN      │
                              └──────┬───────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Gateway                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  /analytics/dashboard    GET     - Dashboard statistics              │    │
│  │  /analytics/sales        GET     - Sales analytics                   │    │
│  │  /analytics/products/top GET     - Top selling products              │    │
│  │  /analytics/categories   GET     - Category performance              │    │
│  │  /analytics/customers    GET     - Customer analytics                │    │
│  │  /analytics/inventory    GET     - Inventory analytics               │    │
│  │  /analytics/export       POST    - Export reports                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Analytics Lambda                                   │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │  Dashboard   │ │    Sales     │ │   Products   │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │  Customer    │ │  Inventory   │ │    Export    │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                    Analytics Service                          │  │     │
│  │  │  - GetDashboardStats()      - GetSalesAnalytics()            │  │     │
│  │  │  - GetTopProducts()         - GetTopCategories()             │  │     │
│  │  │  - GetCustomerAnalytics()   - GetInventoryAnalytics()        │  │     │
│  │  │  - ExportReport()                                             │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Orders     │ │   Products   │ │   Users      │                │     │
│  │  │   Repo       │ │    Repo      │ │   Repo       │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐                                 │     │
│  │  │  Inventory   │ │  Analytics   │                                 │     │
│  │  │    Repo      │ │    Repo      │                                 │     │
│  │  └──────────────┘ └──────────────┘                                 │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                     ┌───────────────┼───────────────┐
                     │               │               │
                     ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ DynamoDB │    │    S3    │    │CloudWatch│
              │  Tables  │    │ (Export) │    │  (Logs)  │
              └──────────┘    └──────────┘    └──────────┘
```

---

## 3. Component Design

### 3.1 Analytics Handler

```go
type AnalyticsHandler struct {
    analyticsService domain.AnalyticsService
    logger           *logger.Logger
}

// Handler Methods
- GetDashboardStats(c *gin.Context)
- GetSalesAnalytics(c *gin.Context)
- GetTopProducts(c *gin.Context)
- GetTopCategories(c *gin.Context)
- GetCustomerAnalytics(c *gin.Context)
- GetInventoryAnalytics(c *gin.Context)
- ExportReport(c *gin.Context)
```

### 3.2 Analytics Service

```go
type AnalyticsService interface {
    GetDashboardStats(ctx context.Context) (*DashboardStats, error)
    GetSalesAnalytics(ctx context.Context, req *SalesAnalyticsRequest) (*SalesAnalytics, error)
    GetTopProducts(ctx context.Context, limit int, sortBy string) ([]ProductAnalytics, error)
    GetTopCategories(ctx context.Context, limit int) ([]CategoryAnalytics, error)
    GetCustomerAnalytics(ctx context.Context) (*CustomerAnalytics, error)
    GetInventoryAnalytics(ctx context.Context) (*InventoryAnalytics, error)
}
```

### 3.3 Analytics Repository

```go
type AnalyticsRepository interface {
    GetOrderStats(ctx context.Context, startDate, endDate time.Time) (*OrderStats, error)
    GetProductStats(ctx context.Context) (*ProductStats, error)
    GetUserStats(ctx context.Context) (*UserStats, error)
    GetSalesByDateRange(ctx context.Context, start, end time.Time) ([]DailySales, error)
    GetTopSellingProducts(ctx context.Context, limit int) ([]ProductSales, error)
    GetCategoryPerformance(ctx context.Context) ([]CategoryPerformance, error)
}
```

---

## 4. Data Model

### 4.1 Dashboard Stats

```go
type DashboardStats struct {
    TotalOrders    int64   `json:"total_orders"`
    TotalRevenue   float64 `json:"total_revenue"`
    TotalProducts  int64   `json:"total_products"`
    TotalUsers     int64   `json:"total_users"`
    ActiveUsers    int64   `json:"active_users"`
    LowStockCount  int64   `json:"low_stock_count"`
    PendingOrders  int64   `json:"pending_orders"`
    OrdersToday    int64   `json:"orders_today"`
    RevenueToday   float64 `json:"revenue_today"`
    GrowthRate     float64 `json:"growth_rate"`
}
```

### 4.2 Sales Analytics

```go
type SalesAnalyticsRequest struct {
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
    GroupBy   string    `json:"group_by"` // day, week, month
}

type SalesAnalytics struct {
    TotalSales     float64      `json:"total_sales"`
    TotalOrders    int64        `json:"total_orders"`
    AvgOrderValue  float64      `json:"avg_order_value"`
    GrowthRate     float64      `json:"growth_rate"`
    SalesTrend     []DailySales `json:"sales_trend"`
    ComparisonData *Comparison  `json:"comparison,omitempty"`
}

type DailySales struct {
    Date    string  `json:"date"`
    Sales   float64 `json:"sales"`
    Orders  int64   `json:"orders"`
}
```

### 4.3 Product Analytics

```go
type ProductAnalytics struct {
    ProductID   string  `json:"product_id"`
    ProductName string  `json:"product_name"`
    SKU         string  `json:"sku"`
    Category    string  `json:"category"`
    TotalSold   int64   `json:"total_sold"`
    Revenue     float64 `json:"revenue"`
    OrderCount  int64   `json:"order_count"`
    AvgPrice    float64 `json:"avg_price"`
}
```

### 4.4 Category Analytics

```go
type CategoryAnalytics struct {
    CategoryID   string  `json:"category_id"`
    CategoryName string  `json:"category_name"`
    TotalProducts int64  `json:"total_products"`
    TotalOrders   int64  `json:"total_orders"`
    Revenue       float64 `json:"revenue"`
    Percentage    float64 `json:"percentage"`
    GrowthRate    float64 `json:"growth_rate"`
}
```

### 4.5 Customer Analytics

```go
type CustomerAnalytics struct {
    TotalCustomers  int64            `json:"total_customers"`
    NewCustomers    int64            `json:"new_customers"`
    ActiveCustomers int64            `json:"active_customers"`
    AvgOrderValue   float64          `json:"avg_order_value"`
    RepeatRate      float64          `json:"repeat_rate"`
    LifetimeValue   float64          `json:"lifetime_value"`
    TopCustomers    []TopCustomer    `json:"top_customers"`
}

type TopCustomer struct {
    CustomerID   string  `json:"customer_id"`
    Name         string  `json:"name"`
    Email        string  `json:"email"`
    TotalOrders  int64   `json:"total_orders"`
    TotalSpent   float64 `json:"total_spent"`
    LastOrderAt  string  `json:"last_order_at"`
}
```

### 4.6 Inventory Analytics

```go
type InventoryAnalytics struct {
    TotalProducts   int64              `json:"total_products"`
    InStock         int64              `json:"in_stock"`
    OutOfStock      int64              `json:"out_of_stock"`
    LowStock        int64              `json:"low_stock"`
    Overstocked     int64              `json:"overstocked"`
    TotalStockValue float64            `json:"total_stock_value"`
    TurnoverRate    float64            `json:"turnover_rate"`
    ByCategory      []CategoryStock    `json:"by_category"`
}

type CategoryStock struct {
    CategoryID   string  `json:"category_id"`
    CategoryName string  `json:"category_name"`
    TotalStock   int64   `json:"total_stock"`
    StockValue   float64 `json:"stock_value"`
    Status       string  `json:"status"` // good, low, over
}
```

---

## 5. DynamoDB Schema

### 5.1 Analytics Aggregation Table

```
Table: handloom-analytics

Primary Key:
- PK: ANALYTICS#<type>
- SK: <date>#<dimension>

Attributes:
- metric_type: string     (sales, orders, products, customers)
- date: string            (YYYY-MM-DD)
- dimension: string       (product_id, category_id, customer_id)
- count: number
- revenue: number
- quantity: number
- updated_at: string

GSI1: metric_type-date-index
- PK: metric_type
- SK: date
```

### 5.2 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get sales by date range | PK = ANALYTICS#SALES, SK between dates | Main |
| Get top products | metric_type = PRODUCT, ordered by revenue | GSI1 |
| Get category stats | PK = ANALYTICS#CATEGORY | Main |
| Get daily aggregates | metric_type = DAILY, SK = date | GSI1 |

---

## 6. API Endpoints

### 6.1 Get Dashboard Statistics

```
GET /analytics/dashboard

Response:
{
    "success": true,
    "data": {
        "total_orders": 1234,
        "total_revenue": 4567890.00,
        "total_products": 456,
        "total_users": 789,
        "active_users": 234,
        "low_stock_count": 12,
        "pending_orders": 45,
        "orders_today": 23,
        "revenue_today": 45678.00,
        "growth_rate": 12.5
    }
}
```

### 6.2 Get Sales Analytics

```
GET /analytics/sales?start_date=2024-01-01&end_date=2024-01-31&group_by=day

Response:
{
    "success": true,
    "data": {
        "total_sales": 1234567.00,
        "total_orders": 456,
        "avg_order_value": 2705.63,
        "growth_rate": 8.5,
        "sales_trend": [
            {"date": "2024-01-01", "sales": 45678.00, "orders": 15},
            {"date": "2024-01-02", "sales": 52341.00, "orders": 18}
        ]
    }
}
```

### 6.3 Get Top Products

```
GET /analytics/products/top?limit=10&sort_by=revenue

Response:
{
    "success": true,
    "data": [
        {
            "product_id": "prod_123",
            "product_name": "Silk Saree Blue",
            "sku": "SKU-001",
            "category": "Sarees",
            "total_sold": 145,
            "revenue": 435000.00,
            "order_count": 140,
            "avg_price": 3000.00
        }
    ]
}
```

### 6.4 Get Top Categories

```
GET /analytics/categories/top?limit=5

Response:
{
    "success": true,
    "data": [
        {
            "category_id": "cat_001",
            "category_name": "Sarees",
            "total_products": 45,
            "total_orders": 234,
            "revenue": 555000.00,
            "percentage": 45.0,
            "growth_rate": 12.5
        }
    ]
}
```

### 6.5 Get Customer Analytics

```
GET /analytics/customers

Response:
{
    "success": true,
    "data": {
        "total_customers": 789,
        "new_customers": 45,
        "active_customers": 234,
        "avg_order_value": 2705.00,
        "repeat_rate": 32.5,
        "lifetime_value": 8500.00,
        "top_customers": [
            {
                "customer_id": "cust_123",
                "name": "Priya Sharma",
                "email": "priya@example.com",
                "total_orders": 12,
                "total_spent": 45000.00,
                "last_order_at": "2024-01-15T10:30:00Z"
            }
        ]
    }
}
```

### 6.6 Get Inventory Analytics

```
GET /analytics/inventory

Response:
{
    "success": true,
    "data": {
        "total_products": 456,
        "in_stock": 432,
        "out_of_stock": 24,
        "low_stock": 45,
        "overstocked": 12,
        "total_stock_value": 2345678.00,
        "turnover_rate": 2.3,
        "by_category": [
            {
                "category_id": "cat_001",
                "category_name": "Sarees",
                "total_stock": 1234,
                "stock_value": 1234000.00,
                "status": "good"
            }
        ]
    }
}
```

---

## 7. Error Handling

### 7.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_DATE_RANGE | End date before start date | 400 |
| DATE_RANGE_TOO_LARGE | Exceeds max range (1 year) | 400 |
| INVALID_LIMIT | Limit out of bounds | 400 |
| INVALID_SORT_BY | Unknown sort field | 400 |
| DATA_NOT_AVAILABLE | No data for period | 404 |
| AGGREGATION_ERROR | Failed to aggregate data | 500 |

### 7.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "INVALID_DATE_RANGE",
        "message": "End date must be after start date"
    }
}
```

---

## 8. Caching Strategy

### 8.1 Cache Configuration

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CACHING STRATEGY                                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Dashboard Stats │     │ Sales Analytics │     │ Top Products    │
│   TTL: 5 min    │     │   TTL: 15 min   │     │   TTL: 30 min   │
└─────────────────┘     └─────────────────┘     └─────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Top Categories  │     │ Customer Stats  │     │ Inventory Stats │
│   TTL: 30 min   │     │   TTL: 1 hour   │     │   TTL: 15 min   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 8.2 Cache Invalidation

- Dashboard stats: Invalidated on new order
- Sales analytics: Invalidated on order completion
- Product analytics: Invalidated on order item changes
- Inventory analytics: Invalidated on stock changes

---

## 9. Performance Optimization

### 9.1 Data Aggregation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AGGREGATION PIPELINE                                  │
└─────────────────────────────────────────────────────────────────────────────┘

  Real-time Events              Aggregation Lambda              Pre-computed
  ┌──────────────┐              ┌──────────────┐               ┌──────────────┐
  │ Order Created│─────────────>│ Aggregate    │──────────────>│ Daily Stats  │
  └──────────────┘              │ by Day       │               └──────────────┘
  ┌──────────────┐              │              │               ┌──────────────┐
  │Order Complete│─────────────>│ Aggregate    │──────────────>│ Weekly Stats │
  └──────────────┘              │ by Week      │               └──────────────┘
  ┌──────────────┐              │              │               ┌──────────────┐
  │ Stock Change │─────────────>│ Aggregate    │──────────────>│Monthly Stats │
  └──────────────┘              │ by Month     │               └──────────────┘
```

### 9.2 Query Optimization

- Use sparse indexes for analytics queries
- Pre-compute frequently accessed metrics
- Batch queries for multiple metrics
- Use parallel queries for independent data

---

## 10. Security

### 10.1 Access Control

| Role | Dashboard | Sales | Products | Customers | Export |
|------|-----------|-------|----------|-----------|--------|
| Admin | Full | Full | Full | Full | Full |
| Manager | View | View | View | View | Export |
| Staff | View | Limited | Limited | - | - |

### 10.2 Data Privacy

- Customer PII masked in exports
- Aggregated data only (no individual transactions)
- Audit logging for all analytics access
- Rate limiting on export endpoints

---

## 11. Monitoring

### 11.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Query Latency | Time to compute analytics | < 2s |
| Cache Hit Rate | % of cached responses | > 80% |
| Aggregation Lag | Delay in pre-computation | < 5 min |
| Export Time | Time to generate export | < 30s |

### 11.2 Alerts

- High query latency (> 5s)
- Cache miss rate spike
- Failed aggregation jobs
- Large export requests

---

## 12. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                         Analytics Lambda
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │ Order Data  │    │ Product Data│    │ User Data   │
   │ (DynamoDB)  │    │ (DynamoDB)  │    │ (DynamoDB)  │
   └─────────────┘    └─────────────┘    └─────────────┘
          │                   │                   │
          └───────────────────┼───────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ S3 (Exports)    │
                    └─────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ CloudWatch Logs │
                    └─────────────────┘
```

### Internal Dependencies
- Order Service: Sales and order data
- Product Service: Product and category data
- User Service: Customer data
- Inventory Service: Stock data

### External Dependencies
- AWS DynamoDB: Data storage
- AWS S3: Export file storage
- AWS CloudWatch: Logging and monitoring

---

## 13. Scalability Considerations

### 13.1 Data Volume Handling

- Partition analytics data by time period
- Use sparse GSIs for filtered queries
- Implement data archival for old analytics
- Support incremental aggregation

### 13.2 Performance Targets

| Metric | Target |
|--------|--------|
| Dashboard load time | < 1s |
| Sales analytics (30 days) | < 2s |
| Top products query | < 500ms |
| Export (100k records) | < 60s |

