# Asset Service - Sequence Diagrams

## Overview
Sequence diagrams for the Asset Service's tmp/ → assets/ S3-only upload flow. There is no DynamoDB involvement — the service only interacts with S3.

---

## 1. Upload File (Frontend: Upload URL → S3 PUT → Store tmp_key)

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────┐
│ Browser  │     │   API GW    │     │  Asset Svc   │     │   S3    │
│(Frontend)│     │  + Lambda   │     │              │     │         │
└────┬────┘     └──────┬──────┘     └──────┬───────┘     └────┬────┘
     │                 │                   │                  │
     │ ─ Step 1: Get presigned URL ─────────────────────────  │
     │                 │                   │                  │
     │ POST /admin/assets/upload-url       │                  │
     │ {file_name, content_type,           │                  │
     │  size, type: "IMAGE"}               │                  │
     │────────────────>│                   │                  │
     │                 │                   │                  │
     │                 │ Validate JWT      │                  │
     │                 │──────────┐        │                  │
     │                 │          │        │                  │
     │                 │<─────────┘        │                  │
     │                 │                   │                  │
     │                 │ GetUploadURL()    │                  │
     │                 │──────────────────>│                  │
     │                 │                   │                  │
     │                 │                   │ Validate content │
     │                 │                   │ type & size      │
     │                 │                   │──────────┐       │
     │                 │                   │          │       │
     │                 │                   │<─────────┘       │
     │                 │                   │                  │
     │                 │                   │ Generate UUID    │
     │                 │                   │ key = tmp/IMAGE/ │
     │                 │                   │ {uuid}.jpg       │
     │                 │                   │──────────┐       │
     │                 │                   │          │       │
     │                 │                   │<─────────┘       │
     │                 │                   │                  │
     │                 │                   │ PresignPutObject │
     │                 │                   │ (15 min expiry)  │
     │                 │                   │─────────────────>│
     │                 │                   │                  │
     │                 │                   │ Presigned URL    │
     │                 │                   │<─────────────────│
     │                 │                   │                  │
     │                 │ {upload_url,      │                  │
     │                 │  tmp_key,         │                  │
     │                 │  expires_at}      │                  │
     │                 │<──────────────────│                  │
     │                 │                   │                  │
     │ 200 OK          │                   │                  │
     │ {upload_url,    │                   │                  │
     │  tmp_key}       │                   │                  │
     │<────────────────│                   │                  │
     │                 │                   │                  │
     │ ─ Step 2: Upload file directly to S3 ─────────────────│
     │                 │                   │                  │
     │ PUT <upload_url>│                   │                  │
     │ Content-Type: image/jpeg            │                  │
     │ Body: <raw file bytes>              │                  │
     │────────────────────────────────────────────────────────>│
     │                 │                   │                  │
     │ 200 OK          │                   │                  │
     │<────────────────────────────────────────────────────────│
     │                 │                   │                  │
     │ ─ Step 3: Store tmp_key, show blob preview ────────── │
     │                 │                   │                  │
     │ Create blob URL │                   │                  │
     │ for local       │                   │                  │
     │ preview.        │                   │                  │
     │ Store tmp_key   │                   │                  │
     │ in form state.  │                   │                  │
     │                 │                   │                  │
     │ (No /finalize   │                   │                  │
     │  call — backend │                   │                  │
     │  finalizes on   │                   │                  │
     │  entity save)   │                   │                  │
     │                 │                   │                  │
```

---

## 2. Delete Asset

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────┐
│ Browser  │     │   API GW    │     │  Asset Svc   │     │   S3    │
│(Frontend)│     │  + Lambda   │     │              │     │         │
└────┬────┘     └──────┬──────┘     └──────┬───────┘     └────┬────┘
     │                 │                   │                  │
     │ DELETE /admin/assets                │                  │
     │ {url: "https://bucket.s3.../       │                  │
     │  assets/IMAGE/2026/02/14/          │                  │
     │  {uuid}.jpg"}                      │                  │
     │────────────────>│                   │                  │
     │                 │                   │                  │
     │                 │ Validate JWT      │                  │
     │                 │──────────┐        │                  │
     │                 │          │        │                  │
     │                 │<─────────┘        │                  │
     │                 │                   │                  │
     │                 │ DeleteAsset()     │                  │
     │                 │──────────────────>│                  │
     │                 │                   │                  │
     │                 │                   │ Parse key from   │
     │                 │                   │ URL, validate    │
     │                 │                   │ assets/ prefix   │
     │                 │                   │──────────┐       │
     │                 │                   │          │       │
     │                 │                   │<─────────┘       │
     │                 │                   │                  │
     │                 │                   │ DeleteObject     │
     │                 │                   │─────────────────>│
     │                 │                   │                  │
     │                 │                   │ Success          │
     │                 │                   │<─────────────────│
     │                 │                   │                  │
     │                 │ Success           │                  │
     │                 │<──────────────────│                  │
     │                 │                   │                  │
     │ 200 OK          │                   │                  │
     │ {message: "..."}│                   │                  │
     │<────────────────│                   │                  │
     │                 │                   │                  │
```

---

## 3. Product Save with Uploaded Images (End-to-End)

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌─────────┐
│ Browser  │     │   API GW    │     │  Asset Svc   │     │ Product Svc  │     │   S3    │
└────┬────┘     └──────┬──────┘     └──────┬───────┘     └──────┬───────┘     └────┬────┘
     │                 │                   │                    │                  │
     │  ── Upload Phase (per file) ──      │                    │                  │
     │                 │                   │                    │                  │
     │ POST /upload-url│                   │                    │                  │
     │────────────────>│──────────────────>│                    │                  │
     │                 │                   │ presign            │                  │
     │                 │                   │─────────────────────────────────────>│
     │ {upload_url,    │                   │                    │                  │
     │  tmp_key}       │                   │                    │                  │
     │<────────────────│<──────────────────│                    │                  │
     │                 │                   │                    │                  │
     │ PUT file → S3   │                   │                    │                  │
     │────────────────────────────────────────────────────────────────────────────>│
     │ 200             │                   │                    │                  │
     │<────────────────────────────────────────────────────────────────────────────│
     │                 │                   │                    │                  │
     │ Store tmp_key   │                   │                    │                  │
     │ in form state,  │                   │                    │                  │
     │ blob URL for    │                   │                    │                  │
     │ preview         │                   │                    │                  │
     │                 │                   │                    │                  │
     │  ── Save Phase (backend finalizes) ─│                    │                  │
     │                 │                   │                    │                  │
     │ POST /admin/products                │                    │                  │
     │ {name: "...",   │                   │                    │                  │
     │  images: [{url: │                   │                    │                  │
     │  "tmp/IMAGE/    │                   │                    │                  │
     │   {uuid}.jpg"}]}│                   │                    │                  │
     │────────────────>│                   │                    │                  │
     │                 │                   │                    │                  │
     │                 │ CreateProduct()   │                    │                  │
     │                 │──────────────────────────────────────>│                  │
     │                 │                   │                    │                  │
     │                 │                   │  FinalizeIfTemp()  │                  │
     │                 │                   │<───────────────────│                  │
     │                 │                   │                    │                  │
     │                 │                   │ CopyObject         │                  │
     │                 │                   │ tmp/ → assets/     │                  │
     │                 │                   │─────────────────────────────────────>│
     │                 │                   │                    │                  │
     │                 │                   │ permanent URL      │                  │
     │                 │                   │───────────────────>│                  │
     │                 │                   │                    │                  │
     │                 │                   │ Save to DynamoDB   │                  │
     │                 │                   │ (permanent URLs)   │                  │
     │                 │                   │                    │                  │
     │                 │ Product created   │                    │                  │
     │                 │<──────────────────────────────────────│                  │
     │                 │                   │                    │                  │
     │ 201 Created     │                   │                    │                  │
     │ {product}       │                   │                    │                  │
     │<────────────────│                   │                    │                  │
     │                 │                   │                    │                  │
```

Note: The Product Service depends on `AssetFinalizer` (implemented by `AssetService`). It finalizes tmp/ keys to permanent URLs atomically during entity save. If the user never saves, no orphaned files are created in assets/.

---

## 4. Unfinalized Upload (Auto-Cleanup)

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────┐
│ Browser  │     │   API GW    │     │  Asset Svc   │     │   S3    │
└────┬────┘     └──────┬──────┘     └──────┬───────┘     └────┬────┘
     │                 │                   │                  │
     │ POST /upload-url│                   │                  │
     │────────────────>│──────────────────>│                  │
     │                 │                   │ presign           │
     │                 │                   │─────────────────>│
     │ {upload_url,    │                   │                  │
     │  tmp_key}       │                   │                  │
     │<────────────────│<──────────────────│                  │
     │                 │                   │                  │
     │ PUT file → S3   │                   │                  │
     │────────────────────────────────────────────────────────>│
     │ 200             │                   │                  │
     │<────────────────────────────────────────────────────────│
     │                 │                   │                  │
     │ ❌ User closes  │                   │                  │
     │ browser / never │                   │                  │
     │ calls finalize  │                   │                  │
     │                 │                   │                  │
     │    ... 24 hours pass ...            │                  │
     │                 │                   │                  │
     │                 │                   │    S3 Lifecycle   │
     │                 │                   │    Rule deletes   │
     │                 │                   │    tmp/ object    │
     │                 │                   │                  │
     │                 │                   │                  │
```

No Lambda, no DynamoDB scan needed — S3 lifecycle handles cleanup automatically.

---

## 5. Error: Invalid File Type

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐
│ Browser  │     │   API GW    │     │  Asset Svc   │
└────┬────┘     └──────┬──────┘     └──────┬───────┘
     │                 │                   │
     │ POST /upload-url│                   │
     │ {file_name: "script.exe",           │
     │  content_type: "application/exe",   │
     │  size: 5000, type: "IMAGE"}         │
     │────────────────>│                   │
     │                 │                   │
     │                 │ GetUploadURL()    │
     │                 │──────────────────>│
     │                 │                   │
     │                 │                   │ isValidContentType
     │                 │                   │ → false
     │                 │                   │──────────┐
     │                 │                   │          │
     │                 │                   │<─────────┘
     │                 │                   │
     │                 │ Error: Invalid    │
     │                 │ content type      │
     │                 │<──────────────────│
     │                 │                   │
     │ 400 Bad Request │                   │
     │ {error:         │                   │
     │  "Invalid       │                   │
     │   content type  │                   │
     │   for asset     │                   │
     │   type"}        │                   │
     │<────────────────│                   │
     │                 │                   │
```
