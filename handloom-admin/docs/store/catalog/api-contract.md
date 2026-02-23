# Store Catalog API Documentation

Public product and category browsing for the B2C storefront. All endpoints are read-only and require no authentication. Only ACTIVE products and categories are returned. Sensitive fields (cost_price) are stripped. Inventory is exposed as a boolean `in_stock` flag.

## Base Path
`/api/v1/store/catalog`

## Endpoints

### List Categories

Retrieve a paginated list of active categories.

**Endpoint:** `GET /api/v1/store/catalog/categories`
**Authentication:** None (public)

**Query Parameters:**

| Parameter    | Type   | Default | Max | Description                        |
|-------------|--------|---------|-----|------------------------------------|
| `limit`     | int    | 20      | 100 | Number of items per page           |
| `cursor`    | string | -       | -   | Pagination cursor from previous response |
| `sort_by`   | string | -       | -   | Field to sort by                   |
| `sort_order`| string | desc    | -   | Sort direction: `asc` or `desc`    |
| `search`    | string | -       | -   | Search by category name            |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "cat-001-uuid-4567-89ab-cdef01234567",
      "name": "Banarasi Sarees",
      "slug": "banarasi-sarees",
      "description": "Handwoven Banarasi silk sarees from Varanasi",
      "image_url": "https://assets.homechrome.lldlab.com/assets/categories/banarasi-sarees.jpg",
      "own_attributes": [
        {
          "name": "material",
          "label": "Material",
          "type": "SELECT",
          "required": true,
          "searchable": true,
          "display_order": 1,
          "options": [
            {"value": "pure_silk", "label": "Pure Silk", "surcharge": 0},
            {"value": "silk_cotton", "label": "Silk Cotton", "surcharge": 0},
            {"value": "organza", "label": "Organza", "surcharge": 50000}
          ]
        }
      ],
      "product_count": 42,
      "status": "ACTIVE",
      "created_at": "2025-08-15T10:00:00Z",
      "updated_at": "2026-02-10T14:30:00Z"
    },
    {
      "id": "cat-002-uuid-4567-89ab-cdef01234567",
      "name": "Pashmina Shawls",
      "slug": "pashmina-shawls",
      "description": "Authentic Kashmiri Pashmina shawls",
      "image_url": "https://assets.homechrome.lldlab.com/assets/categories/pashmina-shawls.jpg",
      "own_attributes": [],
      "product_count": 18,
      "status": "ACTIVE",
      "created_at": "2025-09-01T08:00:00Z",
      "updated_at": "2026-01-20T11:15:00Z"
    }
  ],
  "meta": {
    "limit": 20,
    "next_cursor": "eyJQSyI6IkNBVEVHT1JZI...",
    "has_more": true
  }
}
```

**Error Responses:**
- `500 Internal Server Error` - Database read failure

---

### Get Category

Retrieve a single active category by UUID or slug.

**Endpoint:** `GET /api/v1/store/catalog/categories/{idOrSlug}`
**Authentication:** None (public)

**Path Parameters:**

| Parameter   | Type   | Description                          |
|------------|--------|--------------------------------------|
| `idOrSlug` | string | Category UUID or URL-friendly slug   |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "cat-001-uuid-4567-89ab-cdef01234567",
    "name": "Banarasi Sarees",
    "slug": "banarasi-sarees",
    "description": "Handwoven Banarasi silk sarees from Varanasi",
    "image_url": "https://assets.homechrome.lldlab.com/assets/categories/banarasi-sarees.jpg",
    "own_attributes": [
      {
        "name": "material",
        "label": "Material",
        "type": "SELECT",
        "required": true,
        "searchable": true,
        "display_order": 1,
        "options": [
          {"value": "pure_silk", "label": "Pure Silk", "surcharge": 0},
          {"value": "silk_cotton", "label": "Silk Cotton", "surcharge": 0}
        ]
      },
      {
        "name": "weave_pattern",
        "label": "Weave Pattern",
        "type": "SELECT",
        "required": false,
        "searchable": true,
        "display_order": 2,
        "options": [
          {"value": "jangla", "label": "Jangla"},
          {"value": "tanchoi", "label": "Tanchoi"},
          {"value": "cutwork", "label": "Cutwork"}
        ]
      }
    ],
    "product_count": 42,
    "status": "ACTIVE",
    "created_at": "2025-08-15T10:00:00Z",
    "updated_at": "2026-02-10T14:30:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request` - Category identifier is required
- `404 Not Found` - Category not found or not ACTIVE

---

### List Products

Retrieve a paginated list of active products with optional filters.

**Endpoint:** `GET /api/v1/store/catalog/products`
**Authentication:** None (public)

**Query Parameters:**

| Parameter           | Type   | Default | Max | Description                                       |
|--------------------|--------|---------|-----|---------------------------------------------------|
| `limit`            | int    | 20      | 100 | Number of items per page                          |
| `cursor`           | string | -       | -   | Pagination cursor from previous response          |
| `sort_by`          | string | -       | -   | Field to sort by                                  |
| `sort_order`       | string | desc    | -   | Sort direction: `asc` or `desc`                   |
| `category_id`      | string | -       | -   | Filter by category UUID                           |
| `search`           | string | -       | -   | Search by product name/description                |
| `min_price`        | int    | -       | -   | Minimum selling price in paise                    |
| `max_price`        | int    | -       | -   | Maximum selling price in paise                    |
| `material`         | string | -       | -   | Filter by material                                |
| `color`            | string | -       | -   | Filter by color                                   |
| `attribute_filters`| JSON   | -       | -   | Dynamic attribute filters (URL-encoded JSON)      |

`attribute_filters` example: `?attribute_filters={"weave_type":["jangla","tanchoi"],"border":["zari"]}`

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "prod-001-uuid-4567-89ab-cdef01234567",
      "name": "Royal Blue Banarasi Silk Saree",
      "slug": "royal-blue-banarasi-silk-saree",
      "sku": "BNR-SAR-001",
      "description": "Handwoven pure silk Banarasi saree with gold zari work",
      "category_id": "cat-001-uuid-4567-89ab-cdef01234567",
      "artisan_id": "art-001-uuid",
      "base_price": 1500000,
      "selling_price": 1250000,
      "currency": "INR",
      "dimensions": {
        "length": 5.5,
        "width": 1.15,
        "unit": "METER"
      },
      "weight": 800,
      "allow_custom_dimensions": false,
      "attributes": {
        "weave_pattern": "jangla",
        "border": "zari"
      },
      "material": "Pure Silk",
      "color": "Royal Blue",
      "weave_type": "Banarasi",
      "origin": "Varanasi, UP",
      "craft_type": "Handloom",
      "images": [
        {
          "url": "https://assets.homechrome.lldlab.com/assets/products/bnr-sar-001-1.jpg",
          "alt_text": "Royal Blue Banarasi Silk Saree - Full View",
          "is_primary": true,
          "sort_order": 1
        },
        {
          "url": "https://assets.homechrome.lldlab.com/assets/products/bnr-sar-001-2.jpg",
          "alt_text": "Royal Blue Banarasi Silk Saree - Border Detail",
          "is_primary": false,
          "sort_order": 2
        }
      ],
      "tags": ["banarasi", "silk", "wedding", "festive"],
      "in_stock": true,
      "created_at": "2025-10-20T09:00:00Z",
      "updated_at": "2026-02-18T16:30:00Z"
    }
  ],
  "meta": {
    "limit": 20,
    "next_cursor": "eyJQSyI6IlBST0RVQ1Qi...",
    "has_more": true
  }
}
```

**Notes:**
- `cost_price` is excluded from the response (admin-only field)
- `in_stock` is a boolean derived from `available_qty > 0`
- Prices are in paise (1250000 paise = Rs. 12,500.00)
- Only products with `status=ACTIVE` are returned

**Error Responses:**
- `500 Internal Server Error` - Database read failure

---

### Search Products

Search for products. This is a convenience alias for List Products.

**Endpoint:** `GET /api/v1/store/catalog/products/search`
**Authentication:** None (public)

**Query Parameters:** Same as List Products.

**Response:** Same as List Products.

---

### Get Product

Retrieve a single active product by UUID or slug, with category summary populated.

**Endpoint:** `GET /api/v1/store/catalog/products/{idOrSlug}`
**Authentication:** None (public)

**Path Parameters:**

| Parameter   | Type   | Description                        |
|------------|--------|------------------------------------|
| `idOrSlug` | string | Product UUID or URL-friendly slug  |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "prod-001-uuid-4567-89ab-cdef01234567",
    "name": "Royal Blue Banarasi Silk Saree",
    "slug": "royal-blue-banarasi-silk-saree",
    "sku": "BNR-SAR-001",
    "description": "Handwoven pure silk Banarasi saree with gold zari work. Features intricate jangla weave pattern with heavy zari border. Perfect for weddings and festive occasions.",
    "category_id": "cat-001-uuid-4567-89ab-cdef01234567",
    "artisan_id": "art-001-uuid",
    "base_price": 1500000,
    "selling_price": 1250000,
    "currency": "INR",
    "dimensions": {
      "length": 5.5,
      "width": 1.15,
      "unit": "METER"
    },
    "weight": 800,
    "allow_custom_dimensions": false,
    "pricing_rule_id": "pr-001-uuid",
    "attributes": {
      "weave_pattern": "jangla",
      "border": "zari",
      "pallu_type": "heavy"
    },
    "material": "Pure Silk",
    "color": "Royal Blue",
    "weave_type": "Banarasi",
    "origin": "Varanasi, UP",
    "craft_type": "Handloom",
    "images": [
      {
        "url": "https://assets.homechrome.lldlab.com/assets/products/bnr-sar-001-1.jpg",
        "alt_text": "Royal Blue Banarasi Silk Saree - Full View",
        "is_primary": true,
        "sort_order": 1
      },
      {
        "url": "https://assets.homechrome.lldlab.com/assets/products/bnr-sar-001-2.jpg",
        "alt_text": "Royal Blue Banarasi Silk Saree - Border Detail",
        "is_primary": false,
        "sort_order": 2
      },
      {
        "url": "https://assets.homechrome.lldlab.com/assets/products/bnr-sar-001-3.jpg",
        "alt_text": "Royal Blue Banarasi Silk Saree - Pallu",
        "is_primary": false,
        "sort_order": 3
      }
    ],
    "tags": ["banarasi", "silk", "wedding", "festive"],
    "in_stock": true,
    "category": {
      "id": "cat-001-uuid-4567-89ab-cdef01234567",
      "name": "Banarasi Sarees",
      "slug": "banarasi-sarees"
    },
    "created_at": "2025-10-20T09:00:00Z",
    "updated_at": "2026-02-18T16:30:00Z"
  }
}
```

**Notes:**
- The `category` field is populated with a summary (id, name, slug) from the Category entity
- Slug lookup searches active products and matches on exact slug

**Error Responses:**
- `400 Bad Request` - Product identifier is required
- `404 Not Found` - Product not found or not ACTIVE

---

### Check Product Availability

Check real-time stock availability for a product.

**Endpoint:** `GET /api/v1/store/catalog/products/{id}/availability`
**Authentication:** None (public)

**Path Parameters:**

| Parameter | Type   | Description   |
|----------|--------|---------------|
| `id`     | string | Product UUID  |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "in_stock": true,
    "available_quantity": 7
  }
}
```

Out of stock example:
```json
{
  "success": true,
  "data": {
    "in_stock": false,
    "available_quantity": 0
  }
}
```

**Notes:**
- Fetches live data from the Inventory table (not denormalized product data)
- Falls back to denormalized product `available_qty` if no Inventory record found
- Only works with product UUIDs (not slugs)

**Error Responses:**
- `400 Bad Request` - Product ID is required
- `404 Not Found` - Product not found or not ACTIVE

---
