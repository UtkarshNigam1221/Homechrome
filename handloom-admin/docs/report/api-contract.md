# Report Lambda API Documentation

Report generation service for various business metrics.

## Base Path
`/admin/reports`

## Authentication
All endpoints require authentication.

---

### List Reports

Get paginated list of all generated reports.

**Endpoint:** `GET /admin/reports`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| type | string | - | Filter by report type |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "report_abc123",
      "name": "Monthly Sales Report - January 2024",
      "type": "sales",
      "format": "pdf",
      "status": "completed",
      "file_url": "/reports/sales_jan2024.pdf",
      "file_size": 125000,
      "parameters": {
        "start_date": "2024-01-01",
        "end_date": "2024-01-31"
      },
      "created_by": "user_xyz789",
      "created_at": "2024-02-01T10:00:00Z",
      "completed_at": "2024-02-01T10:01:30Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 50,
    "total_pages": 5
  }
}
```

---

### Generate Report

Generate a custom report.

**Endpoint:** `POST /admin/reports`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Q1 2024 Sales Analysis",
  "type": "sales",
  "format": "pdf",
  "parameters": {
    "start_date": "2024-01-01",
    "end_date": "2024-03-31",
    "group_by": "month",
    "include_charts": true
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_xyz789",
  "name": "Q1 2024 Sales Analysis",
  "type": "sales",
  "status": "pending",
  "message": "Report generation started",
  "created_at": "2024-04-01T10:00:00Z"
}
```

---

### Get My Reports

Get reports created by the authenticated user.

**Endpoint:** `GET /admin/reports/my`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |

---

### Get Report by ID

**Endpoint:** `GET /admin/reports/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "report_abc123",
  "name": "Monthly Sales Report - January 2024",
  "type": "sales",
  "format": "pdf",
  "status": "completed",
  "file_url": "/reports/sales_jan2024.pdf",
  "file_size": 125000,
  "parameters": {
    "start_date": "2024-01-01",
    "end_date": "2024-01-31",
    "group_by": "day"
  },
  "summary": {
    "total_revenue": 1250000.00,
    "total_orders": 340,
    "average_order_value": 3676.47
  },
  "created_by": "user_xyz789",
  "created_at": "2024-02-01T10:00:00Z",
  "completed_at": "2024-02-01T10:01:30Z"
}
```

---

### Delete Report

**Endpoint:** `DELETE /admin/reports/{id}`
**Authentication:** Required

**Response (204 No Content)**

---

### Get Download URL

Get the download URL for a completed report.

**Endpoint:** `GET /admin/reports/{id}/download`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "download_url": "/reports/sales_jan2024.pdf",
  "filename": "sales_jan2024.pdf",
  "expires_at": "2024-02-02T10:00:00Z"
}
```

---

### Generate Sales Report

Generate a sales report.

**Endpoint:** `POST /admin/reports/sales`
**Authentication:** Required

**Request Body:**
```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "format": "pdf",
  "group_by": "day",
  "include_charts": true,
  "filters": {
    "category_ids": ["cat_abc123"],
    "payment_status": "paid"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_sales_001",
  "type": "sales",
  "status": "pending",
  "message": "Sales report generation started"
}
```

---

### Generate Inventory Report

Generate an inventory report.

**Endpoint:** `POST /admin/reports/inventory`
**Authentication:** Required

**Request Body:**
```json
{
  "format": "xlsx",
  "include_low_stock": true,
  "include_out_of_stock": true,
  "filters": {
    "category_ids": ["cat_abc123"]
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_inv_001",
  "type": "inventory",
  "status": "pending",
  "message": "Inventory report generation started"
}
```

---

### Generate Orders Report

Generate an orders report.

**Endpoint:** `POST /admin/reports/orders`
**Authentication:** Required

**Request Body:**
```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "format": "csv",
  "filters": {
    "status": ["confirmed", "shipped", "delivered"],
    "payment_status": "paid"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_ord_001",
  "type": "orders",
  "status": "pending",
  "message": "Orders report generation started"
}
```

---

### Generate Customers Report

Generate a customers report.

**Endpoint:** `POST /admin/reports/customers`
**Authentication:** Required

**Request Body:**
```json
{
  "format": "xlsx",
  "include_order_history": true,
  "filters": {
    "min_orders": 1,
    "registration_date_from": "2024-01-01"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_cust_001",
  "type": "customers",
  "status": "pending",
  "message": "Customers report generation started"
}
```

---

### Generate Products Report

Generate a products report.

**Endpoint:** `POST /admin/reports/products`
**Authentication:** Required

**Request Body:**
```json
{
  "format": "pdf",
  "include_inventory": true,
  "include_sales_data": true,
  "filters": {
    "category_ids": ["cat_abc123"],
    "status": "active"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_prod_001",
  "type": "products",
  "status": "pending",
  "message": "Products report generation started"
}
```

---

### Generate Artisans Report

Generate an artisans report.

**Endpoint:** `POST /admin/reports/artisans`
**Authentication:** Required

**Request Body:**
```json
{
  "format": "xlsx",
  "include_payouts": true,
  "include_products": true,
  "filters": {
    "status": "active",
    "location": "Tamil Nadu"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "report_art_001",
  "type": "artisans",
  "status": "pending",
  "message": "Artisans report generation started"
}
```

---

## Report Types

| Type | Description |
|------|-------------|
| `sales` | Sales and revenue analysis |
| `inventory` | Inventory status and levels |
| `orders` | Order details and status |
| `customers` | Customer data and behavior |
| `products` | Product catalog and performance |
| `artisans` | Artisan details and payouts |

## Report Formats

| Format | Description |
|--------|-------------|
| `pdf` | PDF document with charts |
| `xlsx` | Excel spreadsheet |
| `csv` | CSV file for data analysis |

## Report Status

| Status | Description |
|--------|-------------|
| `pending` | Report queued for generation |
| `processing` | Report being generated |
| `completed` | Report ready for download |
| `failed` | Report generation failed |

---

## TODO

No pending TODO items identified.
