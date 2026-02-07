# Catalog Lambda API Documentation

Product catalog management including Categories, Designs, and Products.

## Authentication
All endpoints require authentication.

---

# Categories

## Base Path
`/admin/categories`

### List Categories

**Endpoint:** `GET /admin/categories`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| parent_id | string | - | Filter by parent category |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "cat_abc123",
      "name": "Sarees",
      "slug": "sarees",
      "description": "Traditional handloom sarees",
      "parent_id": null,
      "image_url": "https://cdn.example.com/categories/sarees.jpg",
      "status": "active",
      "sort_order": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
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
  "slug": "silk-sarees",
  "description": "Premium silk handloom sarees",
  "parent_id": "cat_abc123",
  "image_url": "https://cdn.example.com/categories/silk-sarees.jpg",
  "sort_order": 2
}
```

**Response (201 Created):**
```json
{
  "id": "cat_xyz789",
  "name": "Silk Sarees",
  "slug": "silk-sarees",
  "description": "Premium silk handloom sarees",
  "parent_id": "cat_abc123",
  "image_url": "https://cdn.example.com/categories/silk-sarees.jpg",
  "status": "active",
  "sort_order": 2,
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Category by ID

**Endpoint:** `GET /admin/categories/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "cat_abc123",
  "name": "Sarees",
  "slug": "sarees",
  "description": "Traditional handloom sarees",
  "parent_id": null,
  "image_url": "https://cdn.example.com/categories/sarees.jpg",
  "status": "active",
  "sort_order": 1,
  "product_count": 45,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-10T15:00:00Z"
}
```

---

### Update Category

**Endpoint:** `PUT /admin/categories/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Handloom Sarees",
  "description": "Traditional handwoven sarees",
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "id": "cat_abc123",
  "name": "Handloom Sarees",
  "description": "Traditional handwoven sarees",
  "status": "active",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Delete Category

**Endpoint:** `DELETE /admin/categories/{id}`
**Authentication:** Required

**Response (204 No Content)**

**Error Responses:**
- `400 Bad Request` - Category has products or subcategories

---

### Get Category Tree

**Endpoint:** `GET /admin/categories/tree`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "cat_abc123",
      "name": "Sarees",
      "children": [
        {
          "id": "cat_xyz789",
          "name": "Silk Sarees",
          "children": []
        }
      ]
    }
  ]
}
```

---

# Designs

## Base Path
`/admin/designs`

### List Designs

**Endpoint:** `GET /admin/designs`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| category_id | string | - | Filter by category |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "des_abc123",
      "name": "Peacock Motif",
      "code": "PM-001",
      "description": "Traditional peacock design pattern",
      "category_id": "cat_abc123",
      "image_urls": ["https://cdn.example.com/designs/peacock-1.jpg"],
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
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

### Create Design

**Endpoint:** `POST /admin/designs`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Lotus Border",
  "code": "LB-002",
  "description": "Elegant lotus flower border pattern",
  "category_id": "cat_abc123",
  "image_urls": ["https://cdn.example.com/designs/lotus-1.jpg"]
}
```

**Response (201 Created):**
```json
{
  "id": "des_xyz789",
  "name": "Lotus Border",
  "code": "LB-002",
  "description": "Elegant lotus flower border pattern",
  "category_id": "cat_abc123",
  "image_urls": ["https://cdn.example.com/designs/lotus-1.jpg"],
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Design by ID

**Endpoint:** `GET /admin/designs/{id}`
**Authentication:** Required

---

### Update Design

**Endpoint:** `PUT /admin/designs/{id}`
**Authentication:** Required

---

### Delete Design

**Endpoint:** `DELETE /admin/designs/{id}`
**Authentication:** Required

---

### Get Designs by Category

**Endpoint:** `GET /admin/designs/category/{categoryId}`
**Authentication:** Required

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
| design_id | string | - | Filter by design |
| status | string | - | Filter by status |
| min_price | float | - | Minimum price |
| max_price | float | - | Maximum price |
| search | string | - | Search in name/SKU |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "prod_abc123",
      "sku": "SAR-SLK-001",
      "name": "Kanchipuram Silk Saree",
      "description": "Premium handwoven silk saree with gold zari",
      "category_id": "cat_xyz789",
      "design_id": "des_abc123",
      "base_price": 15000.00,
      "images": [
        {
          "url": "https://cdn.example.com/products/saree-1.jpg",
          "is_primary": true
        }
      ],
      "attributes": {
        "material": "Pure Silk",
        "weight": "650g",
        "length": "6.3m"
      },
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
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

### Create Product

**Endpoint:** `POST /admin/products`
**Authentication:** Required

**Request Body:**
```json
{
  "sku": "SAR-COT-002",
  "name": "Handloom Cotton Saree",
  "description": "Lightweight cotton saree with traditional motifs",
  "category_id": "cat_abc123",
  "design_id": "des_xyz789",
  "base_price": 3500.00,
  "images": [
    {
      "url": "https://cdn.example.com/products/cotton-saree.jpg",
      "is_primary": true
    }
  ],
  "attributes": {
    "material": "Cotton",
    "weight": "400g",
    "length": "6m"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "prod_xyz789",
  "sku": "SAR-COT-002",
  "name": "Handloom Cotton Saree",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Product by ID

**Endpoint:** `GET /admin/products/{id}`
**Authentication:** Required

---

### Update Product

**Endpoint:** `PUT /admin/products/{id}`
**Authentication:** Required

---

### Delete Product

**Endpoint:** `DELETE /admin/products/{id}`
**Authentication:** Required

---

### Get Product Inventory

**Endpoint:** `GET /admin/products/{id}/inventory`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "product_id": "prod_abc123",
  "total_quantity": 25,
  "reserved_quantity": 3,
  "available_quantity": 22,
  "low_stock_threshold": 5,
  "is_low_stock": false,
  "locations": [
    {
      "location_id": "loc_main",
      "quantity": 25
    }
  ]
}
```

---

### Update Product Inventory

**Endpoint:** `PUT /admin/products/{id}/inventory`
**Authentication:** Required

**Request Body:**
```json
{
  "quantity": 30,
  "adjustment_type": "restock",
  "reason": "New stock arrival"
}
```

---

### Search Products

**Endpoint:** `GET /admin/products/search`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Search query |
| limit | int | Max results |

---

## TODO

No pending TODO items identified.
