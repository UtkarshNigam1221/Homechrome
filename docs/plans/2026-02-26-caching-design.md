# Zero-Cost Caching Strategy for Store Catalog APIs

**Date:** 2026-02-26
**Status:** Approved
**Goal:** Reduce PostgreSQL load and API latency for B2C store catalog endpoints with zero additional infrastructure cost.

## Context

The Homechrome B2C storefront serves catalog data (categories, products) via Lambda-backed API endpoints hitting PostgreSQL. Current state:

- **Already cached:** Categories (5min TTL) and individual products (2min TTL) via in-process `go-cache` wrapper
- **Not cached:** Product list queries (the heaviest read path), no HTTP cache headers, no client-side cache configuration
- **Catalog update frequency:** A few times per week
- **Infrastructure constraint:** Zero additional cost — no Redis, no ElastiCache, no new AWS services

## Design

Three-layer caching strategy using only existing infrastructure:

```
Browser / Next.js cache (Cache-Control: max-age=3600)
  | miss
In-process go-cache (1hr TTL, invalidated on writes)
  | miss
PostgreSQL
```

### Layer 1: Extend In-Process Lambda Cache

**Files:** `internal/repository/postgres/cached_category_repo.go`, `cached_product_repo.go`, `internal/wire/providers.go`

Bump existing TTLs and add product list caching:

| Cache target | Current TTL | New TTL |
|-------------|-------------|---------|
| Category list | 5min | 1hr |
| Individual category | 2min | 1hr |
| Individual product | 2min | 1hr |
| Attribute filters | 5min | 1hr |
| Product list queries | NOT CACHED | 1hr |

**Product list cache key strategy:**

Hash the filter parameters into a deterministic cache key:
- Key format: `prod:list:<md5 of canonical filter string>`
- Canonical filter string: sorted serialization of `ListProductsParams` (category_id, search, min_price, max_price, material, color, attribute_filters, sort_by, sort_order, cursor, limit)
- Invalidation: `DeletePrefix("prod:list")` on any product create/update/delete (same pattern as existing category invalidation)

**Limitation:** Cache is per-Lambda-instance. Cold starts hit PostgreSQL, but with 1hr TTLs and steady traffic, warm instances serve the vast majority of requests from memory.

### Layer 2: HTTP Cache Headers on Store Catalog API

**Files:** New middleware in `internal/middleware/`, applied in store catalog router setup

Add a Chi middleware that sets `Cache-Control` headers on public catalog endpoints:

| Endpoint | Header |
|----------|--------|
| `GET /api/v1/store/categories` | `Cache-Control: public, max-age=3600` |
| `GET /api/v1/store/categories/{idOrSlug}` | `Cache-Control: public, max-age=3600` |
| `GET /api/v1/store/products` | `Cache-Control: public, max-age=3600` |
| `GET /api/v1/store/products/search` | `Cache-Control: public, max-age=3600` |
| `GET /api/v1/store/products/{idOrSlug}` | `Cache-Control: public, max-age=3600` |
| `GET /api/v1/store/products/{id}/availability` | `Cache-Control: no-store` |

This gives:
- Browsers cache responses locally (repeat visits and back-button navigation don't hit Lambda)
- Next.js server-side fetch respects these headers automatically
- Future CloudFront distribution would respect these headers with zero extra work

### Layer 3: Next.js Storefront Client-Side Cache

**Files:** React Query configuration in `homechrome-store/`

Set `staleTime` on catalog queries to match API cache TTLs:
- Category queries: `staleTime: 3600000` (1 hour)
- Product list queries: `staleTime: 3600000` (1 hour)
- Product detail queries: `staleTime: 3600000` (1 hour)

This prevents React Query from refetching when users navigate between pages within the storefront during a session.

### Cache Invalidation

**In-process cache:** Already handled by existing `DeletePrefix` calls in `CachedCategoryRepository` and `CachedProductRepository` on create/update/delete. Extend to cover new product list cache keys.

**HTTP cache (browser/Next.js):** No active invalidation. Stale data expires naturally within 1 hour. Acceptable given catalog updates happen a few times per week.

## What This Design Explicitly Does NOT Include

- **No DynamoDB cache layer** — adds complexity (serialization, new key schema, invalidation logic) for marginal benefit given low update frequency and warm Lambda hit rates
- **No ElastiCache/Redis** — violates zero-cost constraint
- **No CloudFront on API** — can be added later; HTTP cache headers are already CloudFront-ready
- **No ISR/static generation** — filter combinations make this impractical; API cache headers provide similar benefit
- **No SNS/SQS-driven cache invalidation** — overengineering for weekly update frequency
- **No cache versioning or `?v=` query params** — future optimization if instant invalidation is ever needed

## Components to Change

1. `internal/repository/postgres/cached_category_repo.go` — bump TTLs to 1hr
2. `internal/repository/postgres/cached_product_repo.go` — bump TTLs to 1hr, add `List()` caching with hashed filter key
3. `internal/wire/providers.go` — update `ProvideCatalogCache()` default TTL and cleanup interval
4. `internal/middleware/cache.go` (new) — Chi middleware for `Cache-Control` headers
5. Store catalog router setup — apply cache middleware to catalog route group
6. `homechrome-store/` React Query config — set `staleTime` for catalog queries
