# Product Video — Design Spec

**Date:** 2026-05-22
**Status:** Approved
**Scope:** Admin uploads a single MP4 video (+ poster image) per product; storefront product page shows it in the gallery.

---

## Goals

- Admin can upload one video per product from the existing product form.
- Customer-facing product page displays the video in the existing image gallery (first slot, play icon overlay).
- Reuse existing tmp/→assets/ S3 asset infrastructure.

## Non-Goals (YAGNI)

- Multiple videos per product.
- Server-side transcoding, multi-bitrate, or HLS streaming.
- Auto-poster extraction (ffmpeg).
- Video analytics / view tracking.
- Autoplay or muted-loop hero video.

## Constraints

- **Limit:** 1 video per product.
- **Max size:** 50 MB.
- **Format:** MP4 / H.264 only (`video/mp4`).
- **Poster:** admin-uploaded image alongside video (optional but recommended).

---

## Data Model

### Migration: `migrations/004_product_video.sql`

```sql
ALTER TABLE products
  ADD COLUMN video_url        TEXT,
  ADD COLUMN video_poster_url TEXT;
```

Both columns nullable. No new table. Backwards-compatible — existing rows = NULL = no video shown.

### Domain (`internal/domain/product.go`)

Add to `Product`, `CreateProductRequest`, `UpdateProductRequest`:

```go
VideoURL       string `json:"video_url,omitempty"`
VideoPosterURL string `json:"video_poster_url,omitempty"`
```

### Repository (`internal/repository/postgres/product_repository.go`)

- Add columns to existing `SELECT` / `INSERT` / `UPDATE` query builders for products.
- No new loader, no batch fetch — fields live on the product row.
- Cache: existing `prodItemTTL` / `prodListTTL` invalidation already covers writes.

---

## Backend

### Asset Service (`internal/service/asset_service.go`)

Extend `GetUploadURL` enforcement when `Type == VIDEO`:

- Reject if `ContentType != "video/mp4"`.
- Reject if `Size > 50 * 1024 * 1024`.
- Apply `content-length-range` condition in the presigned PUT URL with the 50 MB cap (use presigned POST policy if PUT URL lacks size-bound support; verify during implementation).

### Product Service (`internal/service/product_service.go`)

On `Create` / `Update`:

- Call existing `AssetService.FinalizeIfTemp(ctx, videoURL)` for `VideoURL` and `VideoPosterURL` if non-empty.
- Validate URLs belong to the asset bucket (reject arbitrary external URLs).
- On `Update` where video URL changes: delete old asset via `AssetService.DeleteAsset(oldURL)`.

### API

Reuse existing endpoints:

- `POST /admin/assets/upload-url` (already supports `Type: VIDEO`)
- `DELETE /admin/assets`
- `POST /admin/products` / `PUT /admin/products/{id}` (just adds new fields to payload)

No new routes.

---

## Admin Frontend

### Types (`handloom-admin-frontend/src/features/products/types.ts`)

Add `video_url?: string` and `video_poster_url?: string` to `Product`, `CreateProductRequest`, `UpdateProductRequest`.

### Form schema (`ProductFormModal.tsx`)

```ts
video_url:        z.string().url().optional().or(z.literal('')),
video_poster_url: z.string().url().optional().or(z.literal('')),
```

### New Component: `ProductVideoUpload.tsx`

Located: `handloom-admin-frontend/src/features/products/components/ProductFormModal/ProductVideoUpload.tsx`

**Two slots:** Video (MP4, ≤50MB) + Poster (image, ≤2MB).

**Per-slot flow:**

1. User picks file via `<input type="file">`.
2. Client-side validation:
   - Video: `file.type === 'video/mp4'`, `file.size <= 50*1024*1024`.
   - Poster: `file.type.startsWith('image/')`, `file.size <= 2*1024*1024`.
3. `POST /admin/assets/upload-url` with `{ file_name, content_type, size, type: 'VIDEO' | 'IMAGE' }`.
4. Direct `PUT` to returned presigned URL.
5. Store returned `tmp_url` in form field.
6. On product submit, backend `FinalizeIfTemp` moves to `assets/`.

**Preview:** inline `<video controls>` for video; `<img>` for poster.

**Remove:** calls `DELETE /admin/assets` with current URL + clears field.

**Placement:** Single collapsible section "Product Video (optional)" below existing image uploader in `ProductFormModal`.

### API client

Reuse existing `assetApi.getUploadUrl` / `assetApi.delete` in `src/api/asset.ts`.

---

## Storefront (B2C)

### Types (`homechrome-store/src/types/index.ts`)

Add to `Product`:

```ts
video_url?: string;
video_poster_url?: string;
```

### `ProductDetailView.tsx`

**Gallery item type:**

```ts
type GalleryItem =
  | { kind: 'video'; url: string; poster?: string }
  | { kind: 'image'; image: ProductImage };

const items: GalleryItem[] = [
  ...(product.video_url
    ? [{ kind: 'video', url: product.video_url, poster: product.video_poster_url }]
    : []),
  ...sortedImages.map((image) => ({ kind: 'image', image })),
];

const [selected, setSelected] = useState<GalleryItem>(items[0]);
```

**Main viewer:**

- `selected.kind === 'video'` →
  ```tsx
  <video
    controls
    playsInline
    preload="metadata"
    poster={selected.poster}
    className="w-full h-full object-contain"
  >
    <source src={selected.url} type="video/mp4" />
  </video>
  ```
- Else → existing `<Image>` block unchanged.

**Thumbnail strip:**

- Video thumb: `<img src={poster}>` (fallback: gray box) + absolute play-icon overlay (`lucide-react` `Play`).
- Image thumb: unchanged.
- Active border style unchanged.

**SEO:** Append `VideoObject` to JSON-LD when `video_url` present:

```json
{
  "@type": "VideoObject",
  "name": "<product name>",
  "thumbnailUrl": "<video_poster_url>",
  "contentUrl": "<video_url>",
  "uploadDate": "<product.updated_at>"
}
```

**Performance:**

- `preload="metadata"` — only fetches metadata until user plays.
- No autoplay.
- `playsInline` — iOS Safari does not hijack to fullscreen.

**Next.js Image config:** poster served from existing assets CloudFront/S3 — already whitelisted in `next.config.ts`. No change.

---

## Edge Cases

1. **No video on existing products** → columns NULL → gallery renders images only. Backwards compatible.
2. **Admin abandons upload mid-form** → tmp/ S3 lifecycle (24 h) auto-cleans.
3. **Admin replaces video** → old URL deleted via `DELETE /admin/assets` before new upload finalized. If finalize fails, form retains new tmp_url; admin retries submit.
4. **Video set but no poster** → storefront thumb shows gray box + play icon. Acceptable.
5. **Mobile Safari iOS** → `playsInline` attr prevents fullscreen hijack.
6. **SSR hydration** → `<video>` renders server-side; no mismatch since gallery state derived from props on first render.

---

## Testing

### Backend

- `internal/service/asset_service_test.go`: extend `GetUploadURL` table — `Type=VIDEO` happy path, oversize reject, non-mp4 reject.
- `internal/service/product_service_test.go`: video URL set → `FinalizeIfTemp` called; video URL changed on update → old asset deleted.
- `internal/repository/postgres/product_repository_test.go` (integration): create/update/get round-trip video fields, NULL handling.

### Admin Frontend

- `ProductVideoUpload.test.tsx`: oversize rejected, non-mp4 rejected, `assetApi.getUploadUrl` called on valid file, remove clears field + calls delete.
- `ProductFormModal.test.tsx`: video fields included in submit payload.

### Storefront

- `ProductDetailView.test.tsx`: with `video_url` → first thumb has play icon, click → `<video>` rendered; without `video_url` → unchanged behavior.

---

## Migration Plan

1. Land `004_product_video.sql` migration (auto-runs via CDK migrator).
2. Deploy backend (catalog + asset Lambdas).
3. Deploy admin frontend.
4. Deploy storefront.

Backwards-compatible at every step. Old admin can keep submitting without video fields; old storefront ignores new fields.

## Out of Scope

- Transcoding / multi-bitrate / HLS.
- Multiple videos per product.
- Auto-poster extraction (ffmpeg).
- Video analytics events.
- Autoplay / muted-loop hero variants.
