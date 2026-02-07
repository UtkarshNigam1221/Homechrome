# Bulk Lambda API Documentation

Bulk import/export operations service.

## Base Path
`/admin/bulk`

## Authentication
All endpoints require authentication.

---

### List Bulk Operations

Get paginated list of all bulk operations.

**Endpoint:** `GET /admin/bulk`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| type | string | - | Filter by type (IMPORT, EXPORT) |
| entity_type | string | - | Filter by entity (PRODUCT, ORDER, INVENTORY) |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "bulk_abc123",
      "type": "EXPORT",
      "entity_type": "PRODUCT",
      "status": "COMPLETED",
      "total_records": 150,
      "processed_count": 150,
      "success_count": 150,
      "failure_count": 0,
      "input_file_url": null,
      "output_file_url": "/exports/products_20240115_103000.csv",
      "created_by": "user_xyz789",
      "created_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:30:15Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 25,
    "total_pages": 3
  }
}
```

---

### Get My Operations

Get bulk operations created by the authenticated user.

**Endpoint:** `GET /admin/bulk/my`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |

---

### Get Operation by ID

**Endpoint:** `GET /admin/bulk/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "bulk_abc123",
  "type": "IMPORT",
  "entity_type": "PRODUCT",
  "status": "COMPLETED",
  "total_records": 50,
  "processed_count": 50,
  "success_count": 48,
  "failure_count": 2,
  "errors": [
    {
      "row": 15,
      "field": "price",
      "message": "Invalid price format",
      "value": "abc"
    },
    {
      "row": 32,
      "field": "category_id",
      "message": "Category not found",
      "value": "cat_invalid"
    }
  ],
  "input_file_url": "https://s3.../imports/products_upload.csv",
  "output_file_url": null,
  "error_file_url": "/exports/errors_bulk_abc123.csv",
  "metadata": {
    "original_filename": "products_upload.csv",
    "file_size": 25600
  },
  "created_by": "user_xyz789",
  "created_at": "2024-01-15T10:30:00Z",
  "started_at": "2024-01-15T10:30:05Z",
  "completed_at": "2024-01-15T10:31:00Z"
}
```

---

### Cancel Operation

Cancel a pending or in-progress bulk operation.

**Endpoint:** `POST /admin/bulk/{id}/cancel`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "bulk_abc123",
  "status": "CANCELLED",
  "message": "Operation cancelled successfully"
}
```

---

### Get Download URL

Get the download URL for completed export.

**Endpoint:** `GET /admin/bulk/{id}/download`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "download_url": "/exports/products_20240115_103000.csv",
  "filename": "products_20240115_103000.csv",
  "expires_at": "2024-01-16T10:30:00Z"
}
```

---

### Get Error File URL

Get the error file URL for failed imports.

**Endpoint:** `GET /admin/bulk/{id}/errors`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "error_file_url": "/exports/errors_bulk_abc123.csv",
  "filename": "errors_bulk_abc123.csv",
  "error_count": 2
}
```

---

### Get Upload URL

Get a pre-signed URL for uploading import files.

**Endpoint:** `POST /admin/bulk/upload-url`
**Authentication:** Required

**Request Body:**
```json
{
  "filename": "products_import.csv",
  "content_type": "text/csv",
  "entity_type": "PRODUCT"
}
```

**Response (200 OK):**
```json
{
  "upload_url": "https://s3.../presigned-upload-url",
  "file_key": "imports/user_xyz789/products_import_20240115.csv",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

---

### Import Products

Start a bulk product import operation.

**Endpoint:** `POST /admin/bulk/products/import`
**Authentication:** Required

**Request Body:**
```json
{
  "file_url": "https://s3.../imports/products_upload.csv",
  "options": {
    "update_existing": true,
    "skip_validation_errors": false
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "bulk_xyz789",
  "type": "IMPORT",
  "entity_type": "PRODUCT",
  "status": "PENDING",
  "message": "Import operation started",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**CSV Format for Products:**
```csv
sku,name,description,category_id,design_id,base_price,status
SAR-SLK-001,Kanchipuram Silk Saree,Premium silk saree,cat_abc123,des_xyz789,15000.00,active
SAR-COT-002,Cotton Handloom Saree,Lightweight cotton saree,cat_abc123,,3500.00,active
```

---

### Update Inventory (Bulk)

Start a bulk inventory update operation.

**Endpoint:** `POST /admin/bulk/inventory/update`
**Authentication:** Required

**Request Body:**
```json
{
  "file_url": "https://s3.../imports/inventory_update.csv",
  "options": {
    "adjustment_type": "absolute"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "bulk_inv123",
  "type": "IMPORT",
  "entity_type": "INVENTORY",
  "status": "PENDING",
  "message": "Inventory update operation started"
}
```

**CSV Format for Inventory:**
```csv
product_id,sku,quantity,adjustment_type,reason
prod_abc123,SAR-SLK-001,50,restock,Monthly restock
prod_xyz789,SAR-COT-002,30,restock,Monthly restock
```

---

### Update Prices (Bulk)

Start a bulk price update operation.

**Endpoint:** `POST /admin/bulk/prices/update`
**Authentication:** Required

**Request Body:**
```json
{
  "file_url": "https://s3.../imports/prices_update.csv"
}
```

**Response (202 Accepted):**
```json
{
  "id": "bulk_price123",
  "type": "IMPORT",
  "entity_type": "PRICING",
  "status": "PENDING",
  "message": "Price update operation started"
}
```

**CSV Format for Prices:**
```csv
product_id,sku,base_price
prod_abc123,SAR-SLK-001,16500.00
prod_xyz789,SAR-COT-002,3800.00
```

---

### Export Data

Start a bulk export operation.

**Endpoint:** `POST /admin/bulk/export`
**Authentication:** Required

**Request Body:**
```json
{
  "entity_type": "PRODUCT",
  "format": "CSV",
  "filters": {
    "category_id": "cat_abc123",
    "status": "active"
  }
}
```

**Response (200 OK):**
```json
{
  "id": "bulk_exp123",
  "type": "EXPORT",
  "entity_type": "PRODUCT",
  "status": "COMPLETED",
  "total_records": 45,
  "output_file_url": "/exports/products_20240115_103000.csv",
  "message": "Export completed successfully",
  "created_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:05Z"
}
```

**Supported Entity Types for Export:**
| Entity Type | Description |
|-------------|-------------|
| `PRODUCT` | Export products catalog |
| `ORDER` | Export orders |
| `CUSTOMER` | Export customer data |
| `INVENTORY` | Export inventory status |

**Supported Formats:**
| Format | Description |
|--------|-------------|
| `CSV` | Comma-separated values |
| `JSON` | JSON format |

---

## Operation Status

| Status | Description |
|--------|-------------|
| `PENDING` | Operation queued, waiting to start |
| `PROCESSING` | Operation in progress |
| `COMPLETED` | Operation finished successfully |
| `FAILED` | Operation failed |
| `CANCELLED` | Operation was cancelled |

---

## TODO

No pending TODO items identified.
