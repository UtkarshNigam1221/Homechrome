# Analytics Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for the Analytics Lambda service, illustrating the interactions between components for dashboard statistics, sales analytics, and reporting operations.

---

## 1. Get Dashboard Statistics

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/dashboard              │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetDashboardStats() │                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Query Orders        │
     │                 │                     │ (count, revenue)    │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Orders data         │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Query Products      │
     │                 │                     │ (count, stock)      │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Products data       │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Query Users         │
     │                 │                     │ (count, active)     │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Users data          │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Aggregate stats     │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │ Dashboard stats     │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ {stats}         │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 2. Get Sales Analytics

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/sales                  │                     │
     │ ?start=2024-01-01&end=2024-01-31      │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetSalesAnalytics() │                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Parse date range    │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │                     │ Query Orders by     │
     │                 │                     │ date range (GSI)    │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Orders in range     │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Aggregate by day    │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │                     │ Calculate totals    │
     │                 │                     │ & averages          │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │ Sales analytics     │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ {total_sales,   │                     │                     │
     │  orders, trend} │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 3. Get Top Products

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/products/top           │                     │
     │ ?limit=10&sort=revenue                │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetTopProducts()    │                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Query order items   │
     │                 │                     │ with products       │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Order items data    │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Aggregate by        │
     │                 │                     │ product_id          │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │                     │ Sort by revenue     │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │                     │ Query product       │
     │                 │                     │ details             │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Product details     │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │ Top products list   │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ [{product,      │                     │                     │
     │   revenue}]     │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 4. Get Top Categories

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/categories/top         │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetTopCategories()  │                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Query all orders    │
     │                 │                     │ with items          │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Order data          │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Query products      │
     │                 │                     │ for categories      │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Products with       │
     │                 │                     │ categories          │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Aggregate by        │
     │                 │                     │ category            │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │ Category analytics  │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ [{category,     │                     │                     │
     │   revenue, %]}  │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 5. Get Customer Analytics

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/customers              │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetCustomerAnalytics│                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Query all customers │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Customer list       │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Query orders by     │
     │                 │                     │ customer            │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Order history       │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Calculate metrics   │
     │                 │                     │ - Avg order value   │
     │                 │                     │ - Repeat rate       │
     │                 │                     │ - Lifetime value    │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │ Customer analytics  │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ {total, active, │                     │                     │
     │  top_customers} │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 6. Get Inventory Analytics

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘
     │                 │                     │                     │
     │ GET /analytics/inventory              │                     │
     │────────────────>│                     │                     │
     │                 │                     │                     │
     │                 │ Validate JWT        │                     │
     │                 │──────────┐          │                     │
     │                 │          │          │                     │
     │                 │<─────────┘          │                     │
     │                 │                     │                     │
     │                 │ GetInventoryAnalytics                     │
     │                 │────────────────────>│                     │
     │                 │                     │                     │
     │                 │                     │ Query all products  │
     │                 │                     │ with stock data     │
     │                 │                     │────────────────────>│
     │                 │                     │                     │
     │                 │                     │ Products with stock │
     │                 │                     │<────────────────────│
     │                 │                     │                     │
     │                 │                     │ Categorize stock    │
     │                 │                     │ levels:             │
     │                 │                     │ - In stock          │
     │                 │                     │ - Low stock         │
     │                 │                     │ - Out of stock      │
     │                 │                     │ - Overstocked       │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │                     │ Calculate turnover  │
     │                 │                     │──────────┐          │
     │                 │                     │          │          │
     │                 │                     │<─────────┘          │
     │                 │                     │                     │
     │                 │ Inventory analytics │                     │
     │                 │<────────────────────│                     │
     │                 │                     │                     │
     │ 200 OK          │                     │                     │
     │ {total, in_stock│                     │                     │
     │  low_stock,...} │                     │                     │
     │<────────────────│                     │                     │
     │                 │                     │                     │
```

---

## 7. Error Handling - Invalid Date Range

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │
└────┬────┘     └──────┬──────┘     └────────┬────────┘
     │                 │                     │
     │ GET /analytics/sales                  │
     │ ?start=2024-12-31&end=2024-01-01      │
     │────────────────>│                     │
     │                 │                     │
     │                 │ Validate JWT        │
     │                 │──────────┐          │
     │                 │          │          │
     │                 │<─────────┘          │
     │                 │                     │
     │                 │ GetSalesAnalytics() │
     │                 │────────────────────>│
     │                 │                     │
     │                 │                     │ Validate dates
     │                 │                     │──────────┐
     │                 │                     │          │
     │                 │                     │<─────────┘
     │                 │                     │
     │                 │                     │ End date before
     │                 │                     │ start date
     │                 │                     │
     │                 │ Error: Invalid      │
     │                 │ date range          │
     │                 │<────────────────────│
     │                 │                     │
     │ 400 Bad Request │                     │
     │ {error: "End    │                     │
     │  date must be   │                     │
     │  after start"}  │                     │
     │<────────────────│                     │
     │                 │                     │
```

---

## 8. Export Analytics Report

```
┌─────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────────┐     ┌─────────┐
│ Client  │     │   API GW    │     │ Analytics Svc   │     │  DynamoDB    │     │   S3    │
└────┬────┘     └──────┬──────┘     └────────┬────────┘     └──────┬───────┘     └────┬────┘
     │                 │                     │                     │                  │
     │ POST /analytics/export                │                     │                  │
     │ {type: "sales", format: "csv"}        │                     │                  │
     │────────────────>│                     │                     │                  │
     │                 │                     │                     │                  │
     │                 │ Validate JWT        │                     │                  │
     │                 │──────────┐          │                     │                  │
     │                 │          │          │                     │                  │
     │                 │<─────────┘          │                     │                  │
     │                 │                     │                     │                  │
     │                 │ ExportAnalytics()   │                     │                  │
     │                 │────────────────────>│                     │                  │
     │                 │                     │                     │                  │
     │                 │                     │ Query data          │                  │
     │                 │                     │────────────────────>│                  │
     │                 │                     │                     │                  │
     │                 │                     │ Analytics data      │                  │
     │                 │                     │<────────────────────│                  │
     │                 │                     │                     │                  │
     │                 │                     │ Generate CSV        │                  │
     │                 │                     │──────────┐          │                  │
     │                 │                     │          │          │                  │
     │                 │                     │<─────────┘          │                  │
     │                 │                     │                     │                  │
     │                 │                     │ Upload to S3        │                  │
     │                 │                     │────────────────────────────────────────>│
     │                 │                     │                     │                  │
     │                 │                     │ S3 URL              │                  │
     │                 │                     │<────────────────────────────────────────│
     │                 │                     │                     │                  │
     │                 │ Download URL        │                     │                  │
     │                 │<────────────────────│                     │                  │
     │                 │                     │                     │                  │
     │ 200 OK          │                     │                     │                  │
     │ {download_url}  │                     │                     │                  │
     │<────────────────│                     │                     │                  │
     │                 │                     │                     │                  │
```

