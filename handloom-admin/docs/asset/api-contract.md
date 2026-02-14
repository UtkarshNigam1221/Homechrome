# Asset Service - API Documentation

File upload and storage service using presigned S3 URLs.

## Base Path
`/admin/assets`

## Authentication
All endpoints require a valid JWT token in the `Authorization` header.

---

### Get Upload URL

Request a presigned PUT URL to upload a file to the temporary S3 prefix (`tmp/`).

**Endpoint:** `POST /admin/assets/upload-url`

**Request Body:**
```json
{
  "file_name": "product-hero.jpg",
  "content_type": "image/jpeg",
  "size": 2048000,
  "type": "IMAGE"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file_name` | string | Yes | Original file name (used for extension extraction) |
| `content_type` | string | Yes | MIME type (e.g. `image/jpeg`, `video/mp4`) |
| `size` | integer | Yes | File size in bytes |
| `type` | string | Yes | Asset type: `IMAGE`, `VIDEO`, or `DOCUMENT` |

**Response (200 OK):**
```json
{
  "upload_url": "https://handloom-assets-dev.s3.amazonaws.com/tmp/IMAGE/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
  "tmp_key": "tmp/IMAGE/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg",
  "tmp_url": "https://handloom-assets-dev.s3.amazonaws.com/tmp/IMAGE/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg",
  "expires_at": "2026-02-14T12:15:00Z"
}
```

| Field | Description |
|-------|-------------|
| `upload_url` | Presigned S3 PUT URL (expires in 15 minutes) |
| `tmp_key` | S3 key of the temporary file (used for finalize) |
| `tmp_url` | Direct S3 URL (NOT publicly readable — for reference only) |
| `expires_at` | When the presigned URL expires |

**After receiving this response, the client must:**
```
PUT <upload_url>
Content-Type: image/jpeg
Body: <raw file bytes>
```

**Error Responses:**
- `400` — Invalid content type for asset type
- `400` — File size exceeds maximum allowed

---

### Delete Asset

Delete a file from the `assets/` prefix by its public URL.

**Endpoint:** `DELETE /admin/assets`

**Request Body:**
```json
{
  "url": "https://handloom-assets-dev.s3.amazonaws.com/assets/IMAGE/2026/02/14/a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | The full public S3 URL of the asset to delete |

**Response (200 OK):**
```json
{
  "message": "Asset deleted successfully"
}
```

**Error Responses:**
- `400` — `url` is required
- `400` — Invalid asset URL (doesn't match bucket)
- `400` — Can only delete files in `assets/` prefix
- `500` — Failed to delete asset

---

## Asset Types

| Type | Description | Allowed Content Types | Max Size |
|------|-------------|----------------------|----------|
| `IMAGE` | Image files | `image/*` | 50 MB |
| `VIDEO` | Video files | `video/*` | 100 MB |
| `DOCUMENT` | Documents | `application/pdf`, `application/msword`, `application/vnd.openxmlformats-officedocument.*`, `text/csv` | 10 MB |

---

## Complete Upload Flow

```
1. Client                → POST /admin/assets/upload-url {file_name, content_type, size, type}
2. Server                → 200 {upload_url, tmp_key, expires_at}
3. Client                → PUT <upload_url> with raw file body (direct to S3)
4. S3                    → 200 OK
5. Client stores tmp_key in form state, uses blob URL for preview
6. Client saves entity   → POST/PUT /admin/products/{id} {images: [{url: "tmp/IMAGE/..."}]}
7. Backend (ProductService/CategoryService) calls FinalizeIfTemp for each image
8. AssetService copies tmp/ → assets/, returns permanent URL
9. Entity saved to DynamoDB with permanent S3 URLs
```

**Key benefit:** If the user uploads but never saves, the tmp/ file auto-expires in 24h — no orphaned files in assets/.

---

## Frontend Integration

The `ImageUpload` component handles this flow automatically:

```typescript
// 1. Get presigned URL
const { upload_url, tmp_key } = await assetsApi.getUploadUrl(
  file.name, assetType, file.type, file.size
);

// 2. PUT file directly to S3
await fetch(upload_url, {
  method: 'PUT',
  body: file,
  headers: { 'Content-Type': file.type },
});

// 3. Create blob URL for preview, store tmp_key in form state
const blobUrl = URL.createObjectURL(file);
// tmp_key is sent to the backend when the entity is saved
// Backend finalizes tmp/ → assets/ during entity create/update
```

For deletion (best-effort, fire-and-forget — only for permanent URLs):
```typescript
if (url.startsWith('http')) {
  assetsApi.delete(url).catch(() => {});
}
// tmp keys are not deleted — S3 lifecycle auto-cleans in 24h
```
