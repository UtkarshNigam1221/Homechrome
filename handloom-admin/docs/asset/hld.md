# Asset Lambda - High Level Design

## 1. Overview

The Asset Lambda provides media and file management capabilities for the Handloom Admin platform. It handles file uploads, storage, thumbnail generation, and CDN delivery for product images, documents, and other media assets.

### Key Features
- Pre-signed URL uploads for direct S3 access
- Automatic thumbnail generation
- CDN delivery via CloudFront
- Folder-based organization
- Metadata and tagging support
- Usage tracking and references

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ASSET LAMBDA ARCHITECTURE                           │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Client     │
                              │  (Browser)   │
                              └──────┬───────┘
                                     │
            ┌────────────────────────┼────────────────────────┐
            │                        │                        │
            ▼                        ▼                        ▼
     ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
     │  CloudFront  │        │   API GW     │        │    S3        │
     │  (CDN)       │        │              │        │ (Direct)     │
     └──────────────┘        └──────┬───────┘        └──────────────┘
            │                       │                        ▲
            │                       ▼                        │
            │        ┌─────────────────────────────────┐     │
            │        │        Asset Lambda             │     │
            │        │  ┌───────────────────────────┐  │     │
            │        │  │       Handler Layer       │  │     │
            │        │  │  Upload │ List │ Delete   │  │     │
            │        │  └───────────────────────────┘  │     │
            │        │              │                  │     │
            │        │              ▼                  │     │
            │        │  ┌───────────────────────────┐  │     │
            │        │  │       Service Layer       │  │     │
            │        │  │  - GetUploadURL()         │  │     │
            │        │  │  - ConfirmUpload()        │  │     │
            │        │  │  - GetAsset()             │  │     │
            │        │  │  - ListAssets()           │  │     │
            │        │  │  - DeleteAsset()          │  │     │
            │        │  │  - GenerateThumbnails()   │  │     │
            │        │  └───────────────────────────┘  │     │
            │        │              │                  │     │
            │        │              ▼                  │     │
            │        │  ┌───────────────────────────┐  │     │
            │        │  │     Repository Layer      │  │     │
            │        │  └───────────────────────────┘  │     │
            │        └─────────────────────────────────┘     │
            │                       │                        │
            │        ┌──────────────┼──────────────┐         │
            │        │              │              │         │
            │        ▼              ▼              ▼         │
            │ ┌──────────┐  ┌──────────┐  ┌──────────┐       │
            └─│    S3    │  │ DynamoDB │  │  Lambda  │───────┘
              │ (Storage)│  │(Metadata)│  │(Thumbnail│
              └──────────┘  └──────────┘  └──────────┘
```

---

## 3. Component Design

### 3.1 Asset Handler

```go
type AssetHandler struct {
    assetService domain.AssetService
    logger       *logger.Logger
}

// Handler Methods
- GetUploadURL(c *gin.Context)
- ConfirmUpload(c *gin.Context)
- GetAsset(c *gin.Context)
- ListAssets(c *gin.Context)
- UpdateAsset(c *gin.Context)
- DeleteAsset(c *gin.Context)
- GetAssetURL(c *gin.Context)
```

### 3.2 Asset Service

```go
type AssetService interface {
    // Upload
    GetUploadURL(ctx context.Context, req *UploadRequest) (*UploadResponse, error)
    ConfirmUpload(ctx context.Context, key string, metadata *AssetMetadata) (*Asset, error)

    // CRUD
    GetAsset(ctx context.Context, id string) (*Asset, error)
    ListAssets(ctx context.Context, filter *AssetFilter) (*AssetList, error)
    UpdateAsset(ctx context.Context, id string, update *AssetUpdate) (*Asset, error)
    DeleteAsset(ctx context.Context, id string) error

    // URLs
    GetAssetURL(ctx context.Context, id string, size string) (string, error)
    GetCDNURL(ctx context.Context, key string) (string, error)

    // Thumbnails
    GenerateThumbnails(ctx context.Context, assetID string) error
}
```

### 3.3 Asset Repository

```go
type AssetRepository interface {
    Create(ctx context.Context, asset *Asset) error
    GetByID(ctx context.Context, id string) (*Asset, error)
    GetByKey(ctx context.Context, key string) (*Asset, error)
    Update(ctx context.Context, asset *Asset) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter *AssetFilter) ([]*Asset, error)
    GetUsageRefs(ctx context.Context, assetID string) ([]*AssetRef, error)
}
```

---

## 4. Data Model

### 4.1 Asset Entity

```go
type Asset struct {
    ID           string            `json:"id" dynamodbav:"id"`
    Key          string            `json:"key" dynamodbav:"key"`
    Filename     string            `json:"filename" dynamodbav:"filename"`
    Title        string            `json:"title,omitempty" dynamodbav:"title,omitempty"`
    AltText      string            `json:"alt_text,omitempty" dynamodbav:"alt_text,omitempty"`
    ContentType  string            `json:"content_type" dynamodbav:"content_type"`
    Size         int64             `json:"size" dynamodbav:"size"`
    Width        int               `json:"width,omitempty" dynamodbav:"width,omitempty"`
    Height       int               `json:"height,omitempty" dynamodbav:"height,omitempty"`
    Folder       string            `json:"folder" dynamodbav:"folder"`
    Tags         []string          `json:"tags,omitempty" dynamodbav:"tags,omitempty"`
    Thumbnails   map[string]string `json:"thumbnails,omitempty" dynamodbav:"thumbnails,omitempty"`
    URL          string            `json:"url" dynamodbav:"url"`
    CDNURL       string            `json:"cdn_url" dynamodbav:"cdn_url"`
    Status       AssetStatus       `json:"status" dynamodbav:"status"`
    UsageCount   int               `json:"usage_count" dynamodbav:"usage_count"`
    CreatedAt    time.Time         `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy    string            `json:"created_by" dynamodbav:"created_by"`
}
```

### 4.2 Asset Status

```go
type AssetStatus string

const (
    AssetStatusUploading  AssetStatus = "UPLOADING"
    AssetStatusProcessing AssetStatus = "PROCESSING"
    AssetStatusActive     AssetStatus = "ACTIVE"
    AssetStatusArchived   AssetStatus = "ARCHIVED"
    AssetStatusDeleted    AssetStatus = "DELETED"
)
```

### 4.3 Upload Request/Response

```go
type UploadRequest struct {
    Filename    string `json:"filename" binding:"required"`
    ContentType string `json:"content_type" binding:"required"`
    Size        int64  `json:"size" binding:"required"`
    Folder      string `json:"folder,omitempty"`
}

type UploadResponse struct {
    UploadURL string `json:"upload_url"`
    AssetKey  string `json:"asset_key"`
    ExpiresAt int64  `json:"expires_at"`
}
```

### 4.4 Asset Reference

```go
type AssetRef struct {
    ID         string `json:"id" dynamodbav:"id"`
    AssetID    string `json:"asset_id" dynamodbav:"asset_id"`
    EntityType string `json:"entity_type" dynamodbav:"entity_type"` // product, category, artisan
    EntityID   string `json:"entity_id" dynamodbav:"entity_id"`
    Field      string `json:"field" dynamodbav:"field"` // main_image, gallery, banner
    CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
}
```

---

## 5. DynamoDB Schema

### 5.1 Assets Table

```
Table: handloom-assets

Primary Key:
- PK: ASSET#<asset_id>
- SK: ASSET#<asset_id>

Attributes:
- id: string
- key: string (S3 key)
- filename: string
- title: string
- alt_text: string
- content_type: string
- size: number
- width: number
- height: number
- folder: string
- tags: list
- thumbnails: map
- url: string
- cdn_url: string
- status: string
- usage_count: number
- created_at: string
- updated_at: string
- created_by: string

GSI1: folder-type-index
- PK: folder
- SK: content_type#created_at

GSI2: key-index
- PK: key
- SK: ASSET

GSI3: status-index
- PK: status
- SK: created_at
```

### 5.2 Asset References Table

```
Table: handloom-asset-refs

Primary Key:
- PK: ASSET#<asset_id>
- SK: REF#<entity_type>#<entity_id>

GSI1: entity-assets-index
- PK: ENTITY#<entity_type>#<entity_id>
- SK: ASSET#<asset_id>
```

### 5.3 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get asset by ID | PK = ASSET#{id} | Main |
| Get asset by key | PK = {key} | GSI2 |
| List by folder | PK = {folder} | GSI1 |
| List by status | PK = {status} | GSI3 |
| Get asset refs | PK = ASSET#{id} | Refs Table |
| Get entity assets | PK = ENTITY#{type}#{id} | Refs GSI1 |

---

## 6. API Endpoints

### 6.1 Get Upload URL

```
POST /assets/upload-url

Request:
{
    "filename": "product_image.jpg",
    "content_type": "image/jpeg",
    "size": 2500000,
    "folder": "products"
}

Response:
{
    "success": true,
    "data": {
        "upload_url": "https://s3.amazonaws.com/bucket/...",
        "asset_key": "products/abc123_product_image.jpg",
        "expires_at": 1705766400
    }
}
```

### 6.2 Confirm Upload

```
POST /assets/confirm

Request:
{
    "asset_key": "products/abc123_product_image.jpg",
    "title": "Blue Silk Saree",
    "alt_text": "Beautiful blue silk saree with traditional design",
    "tags": ["silk", "blue", "saree"]
}

Response:
{
    "success": true,
    "data": {
        "id": "asset_456",
        "key": "products/abc123_product_image.jpg",
        "url": "https://bucket.s3.amazonaws.com/...",
        "cdn_url": "https://cdn.example.com/...",
        "status": "PROCESSING"
    }
}
```

### 6.3 List Assets

```
GET /assets?folder=products&type=image&limit=20

Response:
{
    "success": true,
    "data": {
        "assets": [
            {
                "id": "asset_456",
                "filename": "product_image.jpg",
                "title": "Blue Silk Saree",
                "content_type": "image/jpeg",
                "size": 2500000,
                "thumbnails": {
                    "small": "https://cdn.../thumb_small.jpg",
                    "medium": "https://cdn.../thumb_medium.jpg"
                },
                "cdn_url": "https://cdn.example.com/...",
                "created_at": "2024-01-20T10:00:00Z"
            }
        ],
        "total": 456,
        "folders": [
            {"name": "products", "count": 456},
            {"name": "artisans", "count": 89}
        ]
    }
}
```

### 6.4 Get Asset URL

```
GET /assets/{id}/url?size=thumbnail

Response:
{
    "success": true,
    "data": {
        "url": "https://cdn.example.com/thumb_small.jpg",
        "expires_at": 1705766400
    }
}
```

---

## 7. S3 Storage Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          S3 BUCKET STRUCTURE                                 │
└─────────────────────────────────────────────────────────────────────────────┘

handloom-assets/
├── products/
│   ├── abc123_silk_saree.jpg
│   ├── def456_cotton_kurta.jpg
│   └── ...
├── artisans/
│   ├── art001_photo.jpg
│   └── ...
├── categories/
│   ├── cat_sarees_banner.jpg
│   └── ...
├── thumbnails/
│   ├── small/
│   │   ├── abc123_silk_saree.jpg
│   │   └── ...
│   ├── medium/
│   │   └── ...
│   └── large/
│       └── ...
└── temp/
    └── uploads/
```

---

## 8. Thumbnail Generation

### 8.1 Thumbnail Sizes

| Size | Dimensions | Use Case |
|------|------------|----------|
| Small | 150x150 | List views, thumbnails |
| Medium | 400x400 | Product cards |
| Large | 800x800 | Product detail |
| Original | Preserved | Full size download |

### 8.2 Processing Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      THUMBNAIL PROCESSING FLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

  Upload Complete       S3 Event           Lambda Trigger      Generate
  ┌──────────┐         ┌──────────┐        ┌──────────┐       ┌──────────┐
  │ Original │────────>│ S3 Event │───────>│ Thumbnail│──────>│ Upload   │
  │ Uploaded │         │ Trigger  │        │ Lambda   │       │ Thumbs   │
  └──────────┘         └──────────┘        └──────────┘       └──────────┘
                                                                    │
                                                                    ▼
                                                              ┌──────────┐
                                                              │ Update   │
                                                              │ Asset DB │
                                                              └──────────┘
```

---

## 9. Error Handling

### 9.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_FILE_TYPE | Unsupported file type | 400 |
| FILE_TOO_LARGE | File exceeds size limit | 400 |
| ASSET_NOT_FOUND | Asset does not exist | 404 |
| UPLOAD_FAILED | S3 upload failed | 500 |
| PROCESSING_FAILED | Thumbnail generation failed | 500 |
| ASSET_IN_USE | Cannot delete, asset is referenced | 400 |

### 9.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "INVALID_FILE_TYPE",
        "message": "File type 'application/exe' is not supported. Allowed types: image/jpeg, image/png, image/webp"
    }
}
```

---

## 10. Security

### 10.1 Access Control

| Role | Upload | View | Update | Delete |
|------|--------|------|--------|--------|
| Admin | Yes | All | Yes | Yes |
| Manager | Yes | All | Yes | Own |
| Staff | Yes | All | Own | No |

### 10.2 File Validation

- Max file size: 5 MB (images), 10 MB (documents)
- Allowed image types: JPEG, PNG, WebP, GIF
- Allowed document types: PDF, DOCX
- Virus scanning on upload
- Content-type verification

### 10.3 URL Security

- Pre-signed URLs expire after 15 minutes
- CDN URLs use signed cookies/tokens
- Private bucket with CloudFront OAI

---

## 11. Performance Optimization

### 11.1 CDN Configuration

- CloudFront distribution for all assets
- Edge caching with 1-year TTL
- Automatic WebP conversion
- Gzip/Brotli compression

### 11.2 Upload Optimization

- Direct-to-S3 uploads (bypass Lambda)
- Multipart upload for large files
- Client-side image resizing (optional)

---

## 12. Monitoring

### 12.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Upload Success Rate | % of successful uploads | > 99% |
| Processing Time | Thumbnail generation time | < 5s |
| CDN Hit Ratio | % of cached requests | > 95% |
| Storage Usage | Total storage used | Monitor |

### 12.2 Alerts

- Upload failure spike
- Processing queue backlog
- Storage approaching limit
- CDN cache invalidation failures

---

## 13. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                           Asset Lambda
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
           ▼                   ▼                   ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │  DynamoDB   │    │     S3      │    │ CloudFront  │
    │ (Metadata)  │    │  (Storage)  │    │   (CDN)     │
    └─────────────┘    └─────────────┘    └─────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │   Lambda    │
                       │ (Thumbnail) │
                       └─────────────┘
```

### External Dependencies
- AWS S3: File storage
- AWS CloudFront: CDN delivery
- AWS Lambda: Thumbnail generation
- AWS DynamoDB: Metadata storage
- Sharp/ImageMagick: Image processing

