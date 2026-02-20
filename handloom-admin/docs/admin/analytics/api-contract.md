# Analytics Lambda API Documentation

Dashboard analytics and reporting service.

## Base Path
`/admin/analytics`

## Authentication
All endpoints require authentication.

---

### Get Dashboard Stats

Get summary statistics for the dashboard.

**Endpoint:** `GET /admin/analytics/dashboard`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "revenue": {
    "today": 45000.00,
    "this_week": 320000.00,
    "this_month": 1250000.00,
    "last_month": 1100000.00,
    "growth_percentage": 13.6
  },
  "orders": {
    "today": 12,
    "this_week": 85,
    "this_month": 340,
    "pending": 15,
    "processing": 8,
    "shipped": 12
  },
  "products": {
    "total": 150,
    "active": 142,
    "out_of_stock": 5,
    "low_stock": 12
  },
  "customers": {
    "total": 520,
    "new_this_month": 45,
    "returning_rate": 35.2
  }
}
```

---

### Get Sales Analytics

Get detailed sales analytics.

**Endpoint:** `GET /admin/analytics/sales`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| period | string | month | Time period (day, week, month, year) |
| start_date | string | - | Start date (ISO 8601) |
| end_date | string | - | End date (ISO 8601) |
| group_by | string | day | Grouping (hour, day, week, month) |

**Request:**
```
GET /admin/analytics/sales?period=month&group_by=day
```

**Response (200 OK):**
```json
{
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "summary": {
    "total_revenue": 1250000.00,
    "total_orders": 340,
    "average_order_value": 3676.47,
    "total_items_sold": 520
  },
  "data": [
    {
      "date": "2024-01-01",
      "revenue": 42000.00,
      "orders": 11,
      "items_sold": 18
    },
    {
      "date": "2024-01-02",
      "revenue": 38500.00,
      "orders": 9,
      "items_sold": 14
    }
  ],
  "comparison": {
    "previous_period_revenue": 1100000.00,
    "revenue_change_percentage": 13.6,
    "previous_period_orders": 298,
    "orders_change_percentage": 14.1
  }
}
```

---

### Get Top Products

Get best-selling products.

**Endpoint:** `GET /admin/analytics/top-products`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| period | string | month | Time period |
| limit | int | 10 | Number of products |
| metric | string | revenue | Sort by (revenue, quantity, orders) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "product_id": "prod_abc123",
      "product_name": "Kanchipuram Silk Saree",
      "sku": "SAR-SLK-001",
      "category": "Silk Sarees",
      "total_revenue": 450000.00,
      "quantity_sold": 30,
      "order_count": 28,
      "average_rating": 4.8
    },
    {
      "product_id": "prod_xyz789",
      "product_name": "Banarasi Silk Saree",
      "sku": "SAR-SLK-002",
      "category": "Silk Sarees",
      "total_revenue": 380000.00,
      "quantity_sold": 25,
      "order_count": 24,
      "average_rating": 4.7
    }
  ],
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  }
}
```

---

### Get Top Categories

Get best-performing categories.

**Endpoint:** `GET /admin/analytics/top-categories`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| period | string | month | Time period |
| limit | int | 10 | Number of categories |

**Response (200 OK):**
```json
{
  "data": [
    {
      "category_id": "cat_abc123",
      "category_name": "Silk Sarees",
      "total_revenue": 850000.00,
      "percentage_of_total": 68.0,
      "quantity_sold": 85,
      "order_count": 78,
      "product_count": 45
    },
    {
      "category_id": "cat_xyz789",
      "category_name": "Cotton Sarees",
      "total_revenue": 250000.00,
      "percentage_of_total": 20.0,
      "quantity_sold": 120,
      "order_count": 95,
      "product_count": 60
    }
  ]
}
```

---

### Get Customer Analytics

Get customer behavior analytics.

**Endpoint:** `GET /admin/analytics/customers`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| period | string | month | Time period |

**Response (200 OK):**
```json
{
  "summary": {
    "total_customers": 520,
    "new_customers": 45,
    "returning_customers": 182,
    "customer_retention_rate": 35.0,
    "average_lifetime_value": 12500.00
  },
  "acquisition": {
    "organic": 25,
    "referral": 10,
    "social": 8,
    "direct": 2
  },
  "segments": [
    {
      "name": "High Value",
      "count": 52,
      "percentage": 10.0,
      "average_order_value": 15000.00,
      "total_revenue": 780000.00
    },
    {
      "name": "Regular",
      "count": 208,
      "percentage": 40.0,
      "average_order_value": 5000.00,
      "total_revenue": 520000.00
    },
    {
      "name": "Occasional",
      "count": 260,
      "percentage": 50.0,
      "average_order_value": 2500.00,
      "total_revenue": 325000.00
    }
  ],
  "geographic": [
    {
      "state": "Tamil Nadu",
      "customer_count": 120,
      "revenue": 450000.00
    },
    {
      "state": "Karnataka",
      "customer_count": 85,
      "revenue": 280000.00
    }
  ]
}
```

---

### Get Inventory Analytics

Get inventory performance analytics.

**Endpoint:** `GET /admin/analytics/inventory`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "summary": {
    "total_sku_count": 150,
    "total_stock_value": 5500000.00,
    "average_turnover_days": 45,
    "dead_stock_value": 120000.00
  },
  "stock_status": {
    "in_stock": 133,
    "low_stock": 12,
    "out_of_stock": 5
  },
  "turnover": {
    "fast_moving": [
      {
        "product_id": "prod_abc123",
        "product_name": "Kanchipuram Silk Saree",
        "turnover_days": 12,
        "monthly_sales": 30
      }
    ],
    "slow_moving": [
      {
        "product_id": "prod_slow001",
        "product_name": "Heavy Silk Saree",
        "days_in_stock": 90,
        "quantity": 5,
        "stock_value": 75000.00
      }
    ]
  },
  "reorder_recommendations": [
    {
      "product_id": "prod_abc123",
      "product_name": "Kanchipuram Silk Saree",
      "current_stock": 8,
      "recommended_order": 25,
      "expected_stockout_days": 5
    }
  ]
}
```

---

## TODO

No pending TODO items identified.
