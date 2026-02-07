# Asset Lambda API Documentation

Media and file asset management service.

## Base Path
`/admin/assets`

## Authentication
All endpoints require authentication.

---

### List Assets

Get paginated list of all assets.

**Endpoint:** `GET /admin/assets`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| type | string | - | Filter by asset type (image, video, document) |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "asset_abc123",
      "filename": "saree_product_01.jpg",
      "original_filename": "IMG_2024.jpg",
      "type": "image",
      "mime_type": "image/jpeg",
      "size": 256000,
      "url": "https://cdn.example.com/assets/saree_product_01.jpg",
      "thumbnail_url": "https://cdn.example.com/assets/thumb/saree_product_01.jpg",
      "dimensions": {
        "width": 1920,
        "height": 1080
      },
      "alt_text": "Kanchipuram Silk Saree - Red",
      "tags": ["product", "saree", "silk"],
      "usage_count": 3,
      "status": "active",
      "created_by": "user_xyz789",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 250,
    "total_pages": 25
  }
}
```

---

### Get Upload URL

Get a pre-signed URL for uploading an asset.

**Endpoint:** `POST /admin/assets/upload-url`
**Authentication:** Required

**Request Body:**
```json
{
  "filename": "product_image.jpg",
  "content_type": "image/jpeg",
  "size": 256000
}
```

**Response (200 OK):**
```json
{
  "upload_url": "https://s3.../presigned-upload-url",
  "asset_id": "asset_xyz789",
  "file_key": "uploads/user_xyz789/product_image_20240115.jpg",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

---

### Get Asset by ID

**Endpoint:** `GET /admin/assets/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "asset_abc123",
  "filename": "saree_product_01.jpg",
  "original_filename": "IMG_2024.jpg",
  "type": "image",
  "mime_type": "image/jpeg",
  "size": 256000,
  "url": "https://cdn.example.com/assets/saree_product_01.jpg",
  "thumbnail_url": "https://cdn.example.com/assets/thumb/saree_product_01.jpg",
  "dimensions": {
    "width": 1920,
    "height": 1080
  },
  "alt_text": "Kanchipuram Silk Saree - Red",
  "description": "Product image for SKU SAR-SLK-001",
  "tags": ["product", "saree", "silk"],
  "metadata": {
    "camera": "Canon EOS R5",
    "iso": 400
  },
  "usage": [
    {
      "entity_type": "product",
      "entity_id": "prod_abc123",
      "field": "primary_image"
    }
  ],
  "usage_count": 1,
  "status": "active",
  "created_by": "user_xyz789",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### Update Asset

**Endpoint:** `PUT /admin/assets/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "alt_text": "Updated alt text for the image",
  "description": "Updated description",
  "tags": ["product", "saree", "silk", "featured"]
}
```

**Response (200 OK):**
```json
{
  "id": "asset_abc123",
  "alt_text": "Updated alt text for the image",
  "tags": ["product", "saree", "silk", "featured"],
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Delete Asset

**Endpoint:** `DELETE /admin/assets/{id}`
**Authentication:** Required

**Response (204 No Content)**

**Error Responses:**
- `400 Bad Request` - Asset is in use

---

### Confirm Upload

Confirm that an asset upload is complete.

**Endpoint:** `POST /admin/assets/{id}/confirm`
**Authentication:** Required

**Request Body:**
```json
{
  "alt_text": "Product image description",
  "tags": ["product", "saree"]
}
```

**Response (200 OK):**
```json
{
  "id": "asset_xyz789",
  "status": "active",
  "url": "https://cdn.example.com/assets/product_image_20240115.jpg",
  "thumbnail_url": "https://cdn.example.com/assets/thumb/product_image_20240115.jpg"
}
```

---

### Get Download URL

Get a signed download URL for an asset.

**Endpoint:** `GET /admin/assets/{id}/download`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "download_url": "https://s3.../presigned-download-url",
  "filename": "saree_product_01.jpg",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

---

### Add Usage

Track where an asset is being used.

**Endpoint:** `POST /admin/assets/{id}/usage`
**Authentication:** Required

**Request Body:**
```json
{
  "entity_type": "product",
  "entity_id": "prod_abc123",
  "field": "gallery_images"
}
```

**Response (200 OK):**
```json
{
  "id": "asset_abc123",
  "usage_count": 2,
  "message": "Usage added successfully"
}
```

---

### Remove Usage

Remove usage tracking for an asset.

**Endpoint:** `DELETE /admin/assets/{id}/usage`
**Authentication:** Required

**Request Body:**
```json
{
  "entity_type": "product",
  "entity_id": "prod_abc123",
  "field": "gallery_images"
}
```

**Response (200 OK):**
```json
{
  "id": "asset_abc123",
  "usage_count": 1,
  "message": "Usage removed successfully"
}
```

---

### Get Assets by Type

**Endpoint:** `GET /admin/assets/type/{type}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| type | string | Asset type (image, video, document) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "asset_abc123",
      "filename": "saree_product_01.jpg",
      "type": "image",
      "url": "https://cdn.example.com/assets/saree_product_01.jpg",
      "thumbnail_url": "https://cdn.example.com/assets/thumb/saree_product_01.jpg"
    }
  ],
  "pagination": {...}
}
```

---

### Search Assets

**Endpoint:** `GET /admin/assets/search`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Search query (filename, tags, alt_text) |
| type | string | Filter by type |
| tags | string | Comma-separated tags |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "asset_abc123",
      "filename": "saree_product_01.jpg",
      "type": "image",
      "url": "https://cdn.example.com/assets/saree_product_01.jpg",
      "tags": ["product", "saree"]
    }
  ]
}
```

---

## Asset Types

| Type | Description | Supported Formats |
|------|-------------|-------------------|
| `image` | Image files | jpg, jpeg, png, gif, webp |
| `video` | Video files | mp4, webm, mov |
| `document` | Documents | pdf, doc, docx, xls, xlsx |

## Asset Status

| Status | Description |
|--------|-------------|
| `pending` | Upload initiated but not confirmed |
| `active` | Asset is available for use |
| `archived` | Asset is archived (not deleted) |

---

## TODO

No pending TODO items identified.
