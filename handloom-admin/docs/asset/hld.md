# Asset Service - High Level Design

## 1. Overview

The Asset Service provides file upload and storage for the Handloom Admin platform. It uses a **tmp/ → assets/ S3-only flow** — no DynamoDB records are stored for assets. Files are uploaded to a temporary S3 prefix via presigned PUT URL, then moved to the permanent `assets/` prefix on finalize.

### Key Features
- Presigned PUT URL uploads (browser → S3 direct, no Lambda proxy)
- Two-phase flow: upload to `tmp/` on the frontend, finalize to `assets/` on the backend when the entity is saved
- S3 lifecycle auto-deletes `tmp/` objects after 24 hours (prevents orphaned files)
- Client-side image compression (browser-image-compression)
- Public-read bucket policy on `assets/*` for permanent URLs
- Support for IMAGE, VIDEO, and DOCUMENT asset types

### What This Service Does NOT Do
- No DynamoDB records for assets
- No CloudFront CDN (direct S3 serving)
- No thumbnail generation
- No usage tracking or asset references
- No media library / browse UI
- No frontend-initiated finalization (backend only)

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      ASSET SERVICE ARCHITECTURE                              │
└─────────────────────────────────────────────────────────────────────────────┘

                           ┌──────────────┐
                           │   Browser    │
                           │  (Frontend)  │
                           └──────┬───────┘
                                  │
               ┌──────────────────┤
               │                  │
               ▼                  ▼
      1. POST /upload-url   2. PUT file directly
               │                  │
               ▼                  ▼
        ┌──────────────┐   ┌──────────────┐
        │  API Gateway │   │     S3       │
        │  → Lambda    │   │  (tmp/)      │
        └──────┬───────┘   └──────────────┘
               │
               ▼                     3. Frontend stores tmp_key in form
        ┌──────────────┐                ↓
        │ Asset        │     4. User saves entity (product/category)
        │ Service      │                ↓
        │              │     5. Backend receives tmp_key in entity payload
        │ Validates    │                ↓
        │ file type,   │     ┌──────────────────────┐
        │ generates    │     │ Product/Category Svc  │
        │ presigned URL│     │ calls FinalizeIfTemp  │
        └──────────────┘     │ → AssetService copies │
                             │   tmp/ → assets/      │
                             └──────────┬───────────┘
                                        │
                                        ▼
                                 ┌──────────────┐
                                 │     S3       │
                                 │  (assets/)   │
                                 │ Public-read  │
                                 └──────────────┘
```

---

## 3. Component Design

### 3.1 Asset Handler

```go
type AssetHandler struct {
    assetService *service.AssetService
    validation   *middleware.Validation
}

// Routes:
// POST /upload-url  → GetUploadURL
// DELETE /          → DeleteAsset
```

### 3.2 Asset Service (implements domain.AssetFinalizer)

```go
type AssetService struct {
    s3Client *s3client.S3Client
    logger   *logger.Logger
    bucket   string
}

func NewAssetService(logger, s3Client, bucket) *AssetService

// Methods:
func (s *AssetService) GetUploadURL(ctx, req UploadAssetRequest) (*UploadURLResponse, error)
func (s *AssetService) FinalizeUpload(ctx, tmpKey string) (string, error)
func (s *AssetService) FinalizeIfTemp(ctx, value string) (string, error)  // AssetFinalizer interface
func (s *AssetService) DeleteAsset(ctx, assetURL string) error
```

### 3.3 AssetFinalizer Interface

```go
// Used by ProductService and CategoryService to finalize images on entity save
type AssetFinalizer interface {
    FinalizeIfTemp(ctx context.Context, value string) (string, error)
}
```

- If `value` starts with `"tmp/"` → calls `FinalizeUpload()` → returns permanent S3 URL
- Otherwise → returns `value` as-is (already a permanent URL)

### 3.4 S3 Client

```go
type S3Client struct {
    client        *s3.Client
    presignClient *s3.PresignClient
}

func New(ctx, region) (*S3Client, error)

func (c *S3Client) GeneratePresignedPutURL(ctx, bucket, key, contentType, expiry) (string, error)
func (c *S3Client) CopyObject(ctx, bucket, srcKey, dstKey) error
func (c *S3Client) DeleteObject(ctx, bucket, key) error
```

---

## 4. Data Model

### No DynamoDB Records

Assets are stored purely in S3. There is no Asset table or entity in DynamoDB. Entities that use assets (Product, Category, Artisan) store the S3 URL directly as a string field.

### 4.1 Domain Types

```go
type AssetType string
const (
    AssetTypeImage    AssetType = "IMAGE"
    AssetTypeDocument AssetType = "DOCUMENT"
    AssetTypeVideo    AssetType = "VIDEO"
)

type UploadAssetRequest struct {
    FileName    string    `json:"file_name" validate:"required"`
    ContentType string    `json:"content_type" validate:"required"`
    Size        int64     `json:"size" validate:"required"`
    Type        AssetType `json:"type" validate:"required"`
}

type UploadURLResponse struct {
    UploadURL string    `json:"upload_url"`
    TmpKey    string    `json:"tmp_key"`
    TmpURL    string    `json:"tmp_url"`
    ExpiresAt time.Time `json:"expires_at"`
}

type DeleteAssetRequest struct {
    URL string `json:"url" validate:"required"`
}
```

---

## 5. S3 Storage Structure

```
handloom-assets-{env}/
├── tmp/                          ← Temporary uploads (auto-deleted after 24h)
│   ├── IMAGE/
│   │   └── {uuid}.jpg
│   ├── VIDEO/
│   │   └── {uuid}.mp4
│   └── DOCUMENT/
│       └── {uuid}.pdf
│
└── assets/                       ← Permanent storage (public-read via bucket policy)
    ├── IMAGE/
    │   └── 2026/02/14/
    │       └── {uuid}.jpg
    ├── VIDEO/
    │   └── 2026/02/14/
    │       └── {uuid}.mp4
    └── DOCUMENT/
        └── 2026/02/14/
            └── {uuid}.pdf
```

### Key Naming Conventions
- **tmp key:** `tmp/{TYPE}/{uuid}.{ext}` (e.g. `tmp/IMAGE/a1b2c3d4.jpg`)
- **Final key:** `assets/{TYPE}/{YYYY/MM/DD}/{uuid}.{ext}` (e.g. `assets/IMAGE/2026/02/14/a1b2c3d4.jpg`)
- **Public URL:** `https://{bucket}.s3.amazonaws.com/assets/{TYPE}/{date}/{uuid}.{ext}`

---

## 6. S3 Configuration

### 6.1 Bucket Policy (Public Read for assets/)

The bucket has a policy that grants `s3:GetObject` on the `assets/*` prefix to everyone. This makes finalized assets publicly accessible via direct S3 URLs. The `tmp/` prefix is NOT publicly readable.

### 6.2 Lifecycle Rule

```
Prefix: tmp/
Action: Expire after 1 day
```

Any file uploaded to `tmp/` that is not finalized within 24 hours is automatically deleted by S3. This replaces the need for a cleanup Lambda or DynamoDB scans.

### 6.3 CORS Configuration

The bucket has CORS rules allowing `PUT` from the frontend origin so browsers can upload directly via presigned URLs.

---

## 7. File Validation

### 7.1 Content Type Validation

| Asset Type | Allowed Content Types |
|------------|----------------------|
| IMAGE | `image/*` (any image MIME type) |
| VIDEO | `video/*` (any video MIME type) |
| DOCUMENT | `application/pdf`, `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`, `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `text/csv` |

### 7.2 Size Limits

| Asset Type | Max Size |
|------------|----------|
| IMAGE | 50 MB |
| VIDEO | 100 MB |
| DOCUMENT | 10 MB |

### 7.3 Client-Side Compression (Frontend)

Images are compressed in the browser before upload using `browser-image-compression`:
- Max output size: 2 MB
- Max dimensions: 2000 x 2000 px
- Quality: 0.8
- Reduces image sizes by 50-80%
- Videos pass through uncompressed

---

## 8. How Entities Use Assets

Entity services (ProductService, CategoryService) depend on the `AssetFinalizer` interface. When an entity is saved, the service finalizes any `tmp/` keys to permanent `assets/` URLs before persisting. Entities store plain S3 URL strings.

### Entity Image Fields

| Entity | Field | Type | Example |
|--------|-------|------|---------|
| Product | `images` | `[]ProductImage` | `[{url: "https://...s3.../assets/IMAGE/...", alt_text: "...", is_primary: true, sort_order: 0}]` |
| Category | `image_url` | `string` | `"https://...s3.../assets/IMAGE/..."` |
| Artisan | `profile_image` | `string` | `"https://...s3.../assets/IMAGE/..."` |
| Order Item | `product_image` | `string` | Copied from product at order time |

### ProductImage Struct

```go
type ProductImage struct {
    URL       string `json:"url"`
    AltText   string `json:"alt_text,omitempty"`
    IsPrimary bool   `json:"is_primary,omitempty"`
    SortOrder int    `json:"sort_order,omitempty"`
}
```

### Flow: How an Image Gets Saved on a Product

1. User opens Product form, clicks "Upload Image"
2. Frontend `ImageUpload` component calls `POST /admin/assets/upload-url` with `{file_name, content_type, size, type: "IMAGE"}`
3. Backend returns `{upload_url, tmp_key}` — presigned S3 PUT URL pointing to `tmp/IMAGE/{uuid}.jpg`
4. Frontend PUTs the (compressed) file directly to S3 using the presigned URL
5. Frontend creates a blob URL for local preview and stores the `tmp_key` in form state (e.g. `images[0].url = "tmp/IMAGE/{uuid}.jpg"`)
6. When user clicks "Save Product", the product POST/PATCH includes the `images` array with `tmp_key` values (for new uploads) or permanent URLs (for existing images)
7. ProductService calls `AssetFinalizer.FinalizeIfTemp()` on each image URL — tmp keys are moved to `assets/`, permanent URLs pass through unchanged
8. Product is saved to DynamoDB with permanent S3 URLs

### Flow: How an Image Gets Deleted

1. User clicks the X button on an image in the Product form
2. Frontend removes the URL from the form state
3. Frontend sends `DELETE /admin/assets` with `{url: "https://..."}` (best-effort, fire-and-forget)
4. If the delete fails, the file stays in S3 (minimal cost impact)
5. When user saves the product, the removed URL is no longer in the `images` array

---

## 9. Error Handling

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `Invalid content type for asset type` | 400 | MIME type doesn't match asset type |
| `File size exceeds maximum allowed` | 400 | File too large for asset type |
| `Invalid tmp key` | 400 | Key doesn't start with `tmp/` |
| `Invalid tmp key format` | 400 | Key doesn't match `tmp/{TYPE}/{file}` pattern |
| `Invalid asset URL` | 400 | URL doesn't match bucket |
| `Can only delete files in assets/ prefix` | 400 | Attempted to delete outside `assets/` |
| `Failed to generate presigned URL` | 500 | S3 presign call failed |
| `Failed to copy asset to final location` | 500 | S3 copy failed |
| `Failed to delete asset` | 500 | S3 delete failed |

---

## 10. Security

### 10.1 Access Control

All asset endpoints require JWT authentication. Any authenticated admin/operator can upload, finalize, and delete assets.

### 10.2 S3 Security

- **Writes:** Only the Lambda execution role can write to the bucket (IAM policy)
- **Reads:** `assets/*` is public-read via bucket policy; `tmp/*` is NOT publicly readable
- **Presigned URLs:** Expire after 15 minutes; scoped to a specific key and content type
- **Delete protection:** Only URLs matching the bucket and `assets/` prefix can be deleted

---

## 11. Cost Estimate (S3 Only, No CloudFront)

| Component | Pricing | 10GB stored, 50GB transfer/month |
|-----------|---------|----------------------------------|
| S3 Storage | $0.023/GB/month | ~$0.23 |
| S3 PUT requests | $0.005/1,000 | ~$0.01 |
| S3 GET requests | $0.0004/1,000 | ~$0.01 |
| S3 Data transfer out | $0.09/GB | ~$4.50 |
| **Total** | | **~$4.75/month** |

AWS Free Tier (first 12 months): 5GB storage, 20K GET, 2K PUT, 100GB transfer — likely **$0/month** for small usage.

With client-side compression: ~50% storage and transfer savings.

---

## 12. Dependencies

```
Asset Lambda
    │
    ├── S3 (storage + presigned URLs)
    │   ├── tmp/ prefix (temporary uploads)
    │   └── assets/ prefix (permanent, public-read)
    │
    └── S3 Lifecycle (auto-cleanup of tmp/)
```

### Go Dependencies
- `github.com/aws/aws-sdk-go-v2/service/s3` — S3 client and presign
- `github.com/google/uuid` — unique file names

### Frontend Dependencies
- `browser-image-compression` — client-side image compression before upload
