# Asset Service - User Flows

## Overview
This document describes the user flows for the Asset Service. The service uses a **tmp/ → assets/ S3-only flow** — there is no media library or standalone asset management UI. Assets are uploaded inline within entity forms (Product, Category, Artisan).

---

## 1. Upload Image in Product Form

```
    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Open Product    │
│ Create/Edit form│
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Image Upload Area:                                                   │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │                                                                 │ │
│ │    ┌─────────────────────────────────────────────────────┐      │ │
│ │    │                                                     │      │ │
│ │    │       Click to upload or drag and drop              │      │ │
│ │    │                                                     │      │ │
│ │    │       PNG, JPG, GIF up to 5MB                       │      │ │
│ │    │       (max 5 files)                                 │      │ │
│ │    │                                                     │      │ │
│ │    └─────────────────────────────────────────────────────┘      │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┬┘
         │
         ▼
┌─────────────────┐
│ Select file(s)  │
│ or drop files   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Behind the scenes (automatic):                                       │
│                                                                      │
│ 1. Compress image in browser (max 2MB, 2000px)                       │
│ 2. Request presigned URL from backend                                │
│ 3. PUT file directly to S3 tmp/ prefix                               │
│ 4. Create blob URL for preview, store tmp_key in form                │
│    (No finalize — backend does that when entity is saved)            │
│                                                                      │
│ UI shows: spinning loader → "Uploading..."                           │
└────────────────────────────────────────────────────────────────────┬┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Upload Complete:                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │                                                                 │ │
│ │ ┌────────┐  ┌────────┐                                          │ │
│ │ │        │  │        │                                          │ │
│ │ │ [img]  │  │ [img]  │  + Upload more                           │ │
│ │ │    [X] │  │    [X] │                                          │ │
│ │ └────────┘  └────────┘                                          │ │
│ │                                                                 │ │
│ │ Toast: "File uploaded successfully"                              │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┬┘
         │
         ▼
┌─────────────────┐
│ Fill other       │
│ product fields   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Save"    │──────► Backend finalizes tmp/ → assets/, product saved
└────────┬────────┘        with permanent S3 URLs in `images` array
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

**If user closes form without saving:** tmp/ files auto-expire in 24h. No orphaned files in assets/.

---

## 2. Remove Image from Product Form

```
    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ View product    │
│ form with       │
│ existing images │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click [X] on    │
│ image thumbnail │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Behind the scenes:                                                   │
│                                                                      │
│ 1. Remove value from form state immediately                          │
│ 2. If it's a permanent URL: send DELETE /admin/assets (best-effort)  │
│    - If it fails, file stays in S3 (negligible cost)                 │
│    - UI doesn't wait for or show delete result                       │
│ 3. If it's a tmp key: do nothing (S3 lifecycle auto-cleans in 24h)   │
└────────────────────────────────────────────────────────────────────┬┘
         │
         ▼
┌─────────────────┐
│ Image removed   │
│ from preview    │
│ grid            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Save"    │──────► Product saved without the removed URL
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 3. Upload Category Image

```
    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Open Category   │
│ Create/Edit form│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click upload    │
│ area for        │
│ category image  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Same upload flow│
│ as product      │
│ (compress →     │
│  presign →      │
│  PUT → tmp_key) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Single image    │
│ preview shown   │
│ (categories use │
│ single image)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Save category   │──────► Backend finalizes tmp/ → assets/, category saved with `image_url` = S3 URL
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 4. Upload Video for Product

```
    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Open product    │
│ form with video │
│ upload enabled  │
│ (accept includes│
│ video/*)        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select video    │
│ file            │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Behind the scenes:                                                   │
│                                                                      │
│ 1. NO compression (videos pass through as-is)                        │
│ 2. Asset type detected as "VIDEO" from MIME type                     │
│ 3. Same presign → PUT → finalize flow                                │
│ 4. Video preview shown with film icon overlay                        │
└────────────────────────────────────────────────────────────────────┬┘
         │
         ▼
┌─────────────────┐
│ Video preview   │
│ with film icon  │
│ and [X] button  │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## State Diagram — Asset Lifecycle

```
                    ┌─────────────────┐
                    │  File selected  │
                    │  in browser     │
                    └────────┬────────┘
                             │
                             │ Compress (images only)
                             ▼
                    ┌─────────────────┐
                    │  IN TMP/        │
                    │  (presigned PUT │
                    │   to S3 tmp/)   │
                    │  Frontend shows │
                    │  blob preview   │
                    └────────┬────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                    ▼                 ▼
           ┌─────────────┐   ┌─────────────────┐
           │ ENTITY SAVED│   │ EXPIRED         │
           │ (backend    │   │ (user closed    │
           │  finalizes  │   │  form without   │
           │  tmp/ →     │   │  saving — S3    │
           │  assets/)   │   │  lifecycle      │
           └──────┬──────┘   │  deletes after  │
                  │          │  24h)           │
         ┌────────┴────────┐ └─────────────────┘
         │                 │
         ▼                 ▼
  ┌─────────────┐   ┌─────────────┐
  │ IN USE      │   │ DELETED     │
  │ (referenced │   │ (DELETE API │
  │  by entity) │   │  call)      │
  └─────────────┘   └─────────────┘
```

---

## Asset Types

```
  ┌─────────────────────────────────────────────────────────┐
  │                  SUPPORTED ASSET TYPES                   │
  │                                                         │
  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
  │  │    IMAGE    │  │   DOCUMENT  │  │     VIDEO       │  │
  │  │ JPG,PNG,    │  │  PDF, DOCX  │  │   MP4, WebM,    │  │
  │  │ GIF, WebP   │  │  XLS, CSV   │  │   MOV           │  │
  │  │ Max 50MB    │  │  Max 10MB   │  │   Max 100MB     │  │
  │  │ Compressed  │  │             │  │   No compress   │  │
  │  └─────────────┘  └─────────────┘  └─────────────────┘  │
  │                                                         │
  └─────────────────────────────────────────────────────────┘
```
