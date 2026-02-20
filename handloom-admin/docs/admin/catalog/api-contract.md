# Catalog Lambda API Documentation

Product catalog management including Categories (flat with custom attributes), Products, and Inventory.

## Authentication
All endpoints require JWT authentication via the `Authorization: Bearer <token>` header.

---

# Categories

## Base Path
`/admin/categories`

Categories are **flat** (no hierarchy/parent-child relationships). Each category can define its own set of custom attributes that products in that category must provide. Attributes marked as `searchable` are indexed for efficient filtering.

### List Categories

**Endpoint:** `GET /admin/categories`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by status (`ACTIVE`, `INACTIVE`) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "cat_abc123",
      "name": "Sarees",
      "slug": "sarees",
      "description": "Traditional handloom sarees",
      "image_url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/categories/sarees.jpg",
      "own_attributes": [
        {
          "name": "material",
          "label": "Material",
          "type": "SELECT",
          "required": true,
          "searchable": true,
          "display_order": 1,
          "options": [
            { "value": "silk", "label": "Silk", "surcharge": 500 },
            { "value": "cotton", "label": "Cotton", "surcharge": 0 }
          ]
        }
      ],
      "status": "ACTIVE",
      "product_count": 45,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-10T15:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 15,
    "total_pages": 2
  }
}
```

---

### Create Category

**Endpoint:** `POST /admin/categories`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Silk Sarees",
  "description": "Premium silk handloom sarees",
  "image_url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/categories/silk-sarees.jpg",
  "own_attributes": [
    {
      "name": "material",
      "label": "Material",
      "type": "SELECT",
      "required": true,
      "searchable": true,
      "display_order": 1,
      "options": [
        { "value": "silk", "label": "Silk", "surcharge": 500 },
        { "value": "cotton", "label": "Cotton" }
      ]
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "id": "cat_xyz789",
  "name": "Silk Sarees",
  "slug": "silk-sarees",
  "description": "Premium silk handloom sarees",
  "image_url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/categories/silk-sarees.jpg",
  "own_attributes": [...],
  "status": "ACTIVE",
  "product_count": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Category by ID

**Endpoint:** `GET /admin/categories/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": {
    "id": "cat_abc123",
    "name": "Sarees",
    "slug": "sarees",
    "description": "Traditional handloom sarees",
    "image_url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/categories/sarees.jpg",
    "own_attributes": [...],
    "status": "ACTIVE",
    "product_count": 45,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-10T15:00:00Z"
  }
}
```

---

### Update Category

**Endpoint:** `PATCH /admin/categories/{id}`
**Authentication:** Required

**Request Body (partial update):**
```json
{
  "name": "Handloom Sarees",
  "description": "Traditional handwoven sarees",
  "status": "ACTIVE"
}
```

**Response (200 OK):**
Returns the full updated category object.

---

### Delete Category

**Endpoint:** `DELETE /admin/categories/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": {
    "message": "Category deleted successfully"
  }
}
```

**Error Responses:**
- `400 Bad Request` - Category has products assigned to it

---

## Category Attribute Management

Categories support custom attributes that define what product-specific fields are available. Attributes with `searchable: true` are indexed via `ProductAttributeIndex` records for efficient filtering.

### Add Attribute

**Endpoint:** `POST /admin/categories/{id}/attributes`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "color",
  "label": "Color",
  "type": "SELECT",
  "required": true,
  "searchable": true,
  "display_order": 2,
  "options": [
    { "value": "red", "label": "Red" },
    { "value": "blue", "label": "Blue" }
  ]
}
```

**Attribute Types:** `SELECT`, `MULTI_SELECT`, `TEXT`, `NUMBER`, `BOOLEAN`, `DIMENSION`, `DIMENSION_RANGE`

**Response (201 Created):**
```json
{
  "data": {
    "attribute": {
      "name": "color",
      "label": "Color",
      "type": "SELECT",
      "required": true,
      "searchable": true,
      "display_order": 2,
      "options": [...]
    },
    "category": {
      "id": "cat_abc123",
      "own_attributes_count": 3
    }
  }
}
```

---

### Update Attribute

**Endpoint:** `PATCH /admin/categories/{id}/attributes/{attrName}`
**Authentication:** Required

**Request Body:**
```json
{
  "label": "Color / Shade",
  "required": false,
  "searchable": true,
  "display_order": 3,
  "options": [
    { "value": "red", "label": "Red" },
    { "value": "blue", "label": "Blue" },
    { "value": "green", "label": "Green" }
  ]
}
```

**Response (200 OK):**
```json
{
  "data": {
    "attribute": {...},
    "affected_products_count": 12
  }
}
```

---

### Delete Attribute

**Endpoint:** `DELETE /admin/categories/{id}/attributes/{attrName}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": {
    "message": "Attribute removed successfully"
  }
}
```

---

### Get Attributes

**Endpoint:** `GET /admin/categories/{id}/attributes`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": {
    "own_attributes": [
      {
        "name": "material",
        "label": "Material",
        "type": "SELECT",
        "required": true,
        "searchable": true,
        "display_order": 1,
        "options": [...]
      }
    ],
    "total_count": 3
  }
}
```

---

# Products

## Base Path
`/admin/products`

### List Products

**Endpoint:** `GET /admin/products`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| category_id | string | - | Filter by category |
| status | string | - | Filter by status (`ACTIVE`, `INACTIVE`, `DRAFT`) |
| min_price | int | - | Min base price (in paise) |
| max_price | int | - | Max base price (in paise) |
| in_stock | bool | - | Filter in-stock products |
| low_stock | bool | - | Filter low-stock products |
| material | string | - | Filter by material |
| color | string | - | Filter by color |
| search | string | - | Search in name/SKU |
| attribute_filters | JSON string | - | Dynamic attribute filters (see below) |

**`attribute_filters` format:**
A JSON-encoded map of attribute names to arrays of values. When a category has searchable attributes, this enables filtering by those attributes using the `ProductAttributeIndex` GSI.
```
attribute_filters={"material":["silk","cotton"],"color":["red"]}
```

**Response (200 OK):**
```json
{
  "products": [
    {
      "id": "prod_abc123",
      "sku": "SAR-SLK-001",
      "name": "Kanchipuram Silk Saree",
      "slug": "kanchipuram-silk-saree",
      "description": "Premium handwoven silk saree with gold zari",
      "category_id": "cat_xyz789",
      "base_price": 1500000,
      "selling_price": 1800000,
      "cost_price": 1200000,
      "currency": "INR",
      "dimensions": {
        "length": 6.3,
        "width": 1.2,
        "unit": "meter"
      },
      "weight": 650,
      "attributes": {
        "material": "silk",
        "color": "red",
        "weave_type": "jacquard"
      },
      "material": "silk",
      "color": "red",
      "weave_type": "jacquard",
      "images": [
        {
          "url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/products/saree-1.jpg",
          "is_primary": true,
          "sort_order": 0
        }
      ],
      "tags": ["premium", "wedding"],
      "quantity": 25,
      "reserved_qty": 3,
      "available_qty": 22,
      "low_stock_threshold": 5,
      "status": "ACTIVE",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-05T12:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 100,
    "total_pages": 10
  }
}
```

---

### Get Attribute Filter Options

**Endpoint:** `GET /admin/products/filter-options/{categoryId}`
**Authentication:** Required

Returns the distinct values for each searchable attribute in a category. These are pre-computed and stored in a `CategoryAttributeValues` record, updated atomically whenever a product is created or updated.

**Response (200 OK):**
```json
{
  "data": {
    "material": ["cotton", "silk"],
    "color": ["blue", "green", "red"],
    "weave_type": ["dobby", "jacquard", "plain"]
  }
}
```

---

### Create Product

**Endpoint:** `POST /admin/products`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Handloom Cotton Saree",
  "sku": "SAR-COT-002",
  "category_id": "cat_abc123",
  "description": "Lightweight cotton saree with traditional motifs",
  "base_price": 350000,
  "selling_price": 450000,
  "cost_price": 280000,
  "dimensions": {
    "length": 6.0,
    "width": 1.1,
    "unit": "meter"
  },
  "weight": 400,
  "attributes": {
    "material": "cotton",
    "color": "blue",
    "weave_type": "plain"
  },
  "material": "cotton",
  "color": "blue",
  "weave_type": "plain",
  "images": [
    {
      "url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/products/cotton-saree.jpg",
      "is_primary": true,
      "sort_order": 0
    }
  ],
  "tags": ["everyday", "casual"],
  "initial_stock": 50,
  "low_stock_threshold": 10
}
```

**Side effects on create:**
1. `ProductAttributeIndex` records are created for each searchable attribute value (enables GSI-based filtering)
2. `CategoryAttributeValues` record is updated atomically to include the new attribute values (pre-computed filter options)
3. Inventory record is created with `initial_stock`
4. Category `product_count` is incremented

**Response (201 Created):**
Returns the created product object.

---

### Get Product by ID

**Endpoint:** `GET /admin/products/{id}`
**Authentication:** Required

**Response (200 OK):**
Returns the product with related entities:
```json
{
  "id": "prod_abc123",
  "name": "Kanchipuram Silk Saree",
  ...
  "category": {
    "id": "cat_xyz789",
    "name": "Silk Sarees",
    "slug": "silk-sarees"
  },
  "inventory": {
    "product_id": "prod_abc123",
    "quantity": 25,
    "reserved_qty": 3,
    "available_qty": 22,
    "low_stock_threshold": 5
  }
}
```

---

### Update Product

**Endpoint:** `PATCH /admin/products/{id}`
**Authentication:** Required

**Side effects on update:**
1. Old `ProductAttributeIndex` records are removed, new ones are created (if searchable attributes changed)
2. `CategoryAttributeValues` record is updated with any new attribute values

**Response (200 OK):**
Returns the updated product object.

---

### Delete Product

**Endpoint:** `DELETE /admin/products/{id}`
**Authentication:** Required

**Side effects on delete:**
1. `ProductAttributeIndex` records for the product are deleted
2. Inventory record and all inventory transactions are deleted (`DeleteByProductID`)
3. Category `product_count` is decremented

**Response (200 OK):**
```json
{
  "message": "Product deleted successfully"
}
```

---

## Product Inventory

Inventory is managed through sub-routes under each product.

### Get Inventory

**Endpoint:** `GET /admin/products/{id}/inventory`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "inv_abc123",
  "product_id": "prod_abc123",
  "product_sku": "SAR-SLK-001",
  "product_name": "Kanchipuram Silk Saree",
  "quantity": 25,
  "reserved_qty": 3,
  "available_qty": 22,
  "low_stock_threshold": 5,
  "reorder_point": 10,
  "last_restock_at": "2024-01-10T15:00:00Z"
}
```

---

### Add Stock

**Endpoint:** `POST /admin/products/{id}/inventory/add`
**Authentication:** Required

**Request Body:**
```json
{
  "quantity": 30,
  "reason": "New stock arrival"
}
```

**Response (200 OK):**
```json
{
  "product_id": "prod_abc123",
  "previous_quantity": 25,
  "change_quantity": 30,
  "new_quantity": 55,
  "available_qty": 52,
  "transaction_id": "txn_xyz789"
}
```

---

### Remove Stock

**Endpoint:** `POST /admin/products/{id}/inventory/remove`
**Authentication:** Required

**Request Body:**
```json
{
  "quantity": 5,
  "reason": "Damaged goods"
}
```

---

### Adjust Stock

**Endpoint:** `POST /admin/products/{id}/inventory/adjust`
**Authentication:** Required

**Request Body:**
```json
{
  "new_quantity": 20,
  "reason": "Physical count correction"
}
```

---

### Get Inventory Transactions

**Endpoint:** `GET /admin/products/{id}/inventory/transactions`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |

---

## Low Stock

### Get Low Stock Products

**Endpoint:** `GET /admin/inventory/low-stock`
**Authentication:** Required

Returns products where available quantity is at or below the low stock threshold.

---

## Data Models

### Category
| Field | Type | Description |
|-------|------|-------------|
| id | string | UUID |
| name | string | Category name |
| slug | string | URL-friendly slug (auto-generated) |
| description | string | Optional description |
| image_url | string | Category image URL |
| own_attributes | CategoryAttribute[] | Custom attributes for this category |
| status | string | `ACTIVE` or `INACTIVE` |
| product_count | int | Denormalized count of products |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Last update time |

### CategoryAttribute
| Field | Type | Description |
|-------|------|-------------|
| name | string | Attribute identifier (e.g. `material`) |
| label | string | Display label (e.g. `Material`) |
| type | string | One of: `SELECT`, `MULTI_SELECT`, `TEXT`, `NUMBER`, `BOOLEAN`, `DIMENSION`, `DIMENSION_RANGE` |
| required | bool | Whether products must provide this attribute |
| searchable | bool | If true, attribute values are indexed for filtering |
| display_order | int | Display order in forms |
| options | AttributeOption[] | For `SELECT`/`MULTI_SELECT` types |

### AttributeOption
| Field | Type | Description |
|-------|------|-------------|
| value | string | Option value |
| label | string | Display label |
| surcharge | int | Optional price surcharge in paise |

### Product
| Field | Type | Description |
|-------|------|-------------|
| id | string | UUID |
| sku | string | Stock keeping unit (unique) |
| name | string | Product name |
| slug | string | URL-friendly slug |
| description | string | Optional description |
| category_id | string | Category this product belongs to |
| base_price | int64 | Base price in paise |
| selling_price | int64 | Selling price in paise |
| cost_price | int64 | Cost price in paise |
| currency | string | Currency code (e.g. `INR`) |
| dimensions | Dimensions | Length, width, height, unit |
| weight | int | Weight in grams |
| attributes | map[string]any | Flexible key-value attributes matching category definition |
| material | string | Common indexed attribute |
| color | string | Common indexed attribute |
| weave_type | string | Common indexed attribute |
| images | ProductImage[] | Product images |
| tags | string[] | Tags for organization |
| quantity | int | Total inventory quantity (denormalized) |
| reserved_qty | int | Reserved quantity |
| available_qty | int | Available quantity |
| low_stock_threshold | int | Low stock alert threshold |
| status | string | `ACTIVE`, `INACTIVE`, or `DRAFT` |
