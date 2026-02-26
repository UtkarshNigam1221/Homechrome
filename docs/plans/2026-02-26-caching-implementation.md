# Zero-Cost Caching Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add three-layer caching (in-process, HTTP headers, Next.js ISR) to store catalog APIs with zero new infrastructure.

**Architecture:** Extend existing `go-cache` wrapper to cover product lists with 1hr TTLs, add a Chi middleware for `Cache-Control` headers on catalog routes, and bump Next.js server-side `revalidate` to match.

**Tech Stack:** Go 1.24, Chi router, `github.com/patrickmn/go-cache`, Next.js 16 server components

**Design doc:** `docs/plans/2026-02-26-caching-design.md`

---

### Task 1: Bump Category Cache TTLs to 1 Hour

**Files:**
- Modify: `handloom-admin/internal/repository/postgres/cached_category_repo.go:12-16`

**Step 1: Update TTL constants**

Change the TTL constants from 5min/2min to 1hr:

```go
const (
	catListKey   = "cat:list"
	catListTTL   = 1 * time.Hour
	catItemTTL   = 1 * time.Hour
	catKeyPrefix = "cat:"
)
```

**Step 2: Verify it compiles**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
cd handloom-admin
git add internal/repository/postgres/cached_category_repo.go
git commit -m "feat(cache): bump category cache TTLs to 1 hour"
```

---

### Task 2: Bump Product Cache TTLs to 1 Hour

**Files:**
- Modify: `handloom-admin/internal/repository/postgres/cached_product_repo.go:12-15`

**Step 1: Update TTL constants**

```go
const (
	prodItemTTL = 1 * time.Hour
	prodAttrTTL = 1 * time.Hour
	prodPrefix  = "prod:"
)
```

**Step 2: Verify it compiles**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
cd handloom-admin
git add internal/repository/postgres/cached_product_repo.go
git commit -m "feat(cache): bump product cache TTLs to 1 hour"
```

---

### Task 3: Update Cache Provider Default TTL

**Files:**
- Modify: `handloom-admin/internal/wire/providers.go:53-55`

**Step 1: Update ProvideCatalogCache**

The default TTL and cleanup interval should match the new 1hr cache TTLs:

```go
func ProvideCatalogCache() *cache.Cache {
	return cache.New(1*time.Hour, 2*time.Hour)
}
```

**Step 2: Verify it compiles**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
cd handloom-admin
git add internal/wire/providers.go
git commit -m "feat(cache): update catalog cache default TTL to 1 hour"
```

---

### Task 4: Add Product List Caching

This is the main feature — caching product list queries that are currently uncached.

**Files:**
- Modify: `handloom-admin/internal/repository/postgres/cached_product_repo.go`

**Step 1: Add imports and cache key helper**

Add `"crypto/md5"`, `"encoding/hex"`, `"encoding/json"`, and `"sort"` to the import block. Add a helper function to generate a deterministic cache key from `ListProductsRequest`:

```go
const (
	prodItemTTL   = 1 * time.Hour
	prodAttrTTL   = 1 * time.Hour
	prodPrefix    = "prod:"
	prodListTTL   = 1 * time.Hour
	prodListPrefix = "prod:list:"
)

// prodListKey builds a deterministic cache key from a ListProductsRequest
// by serializing the filter fields to JSON and hashing them.
func prodListKey(req domain.ListProductsRequest) string {
	// Build a canonical representation of all filter params.
	// We include everything that affects the result set.
	canonical := struct {
		Limit      int                 `json:"l"`
		Cursor     string              `json:"c,omitempty"`
		SortBy     string              `json:"sb,omitempty"`
		SortDir    string              `json:"sd,omitempty"`
		CategoryID *string             `json:"cat,omitempty"`
		Status     *domain.ProductStatus `json:"st,omitempty"`
		Search     string              `json:"q,omitempty"`
		MinPrice   *int64              `json:"mn,omitempty"`
		MaxPrice   *int64              `json:"mx,omitempty"`
		InStock    *bool               `json:"is,omitempty"`
		LowStock   *bool               `json:"ls,omitempty"`
		Material   *string             `json:"ma,omitempty"`
		Color      *string             `json:"co,omitempty"`
		AttrFilter map[string][]string `json:"af,omitempty"`
	}{
		Limit:      req.Limit,
		Cursor:     req.Cursor,
		SortBy:     req.SortBy,
		SortDir:    req.SortDir,
		CategoryID: req.CategoryID,
		Status:     req.Status,
		Search:     req.Search,
		MinPrice:   req.MinPrice,
		MaxPrice:   req.MaxPrice,
		InStock:    req.InStock,
		LowStock:   req.LowStock,
		Material:   req.Material,
		Color:      req.Color,
		AttrFilter: req.AttributeFilters,
	}

	// Sort attribute filter values for deterministic hashing
	if canonical.AttrFilter != nil {
		for k := range canonical.AttrFilter {
			sort.Strings(canonical.AttrFilter[k])
		}
	}

	data, _ := json.Marshal(canonical)
	hash := md5.Sum(data)
	return prodListPrefix + hex.EncodeToString(hash[:])
}
```

**Step 2: Implement cached List method**

Replace the pass-through `List` method:

```go
func (r *CachedProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	key := prodListKey(req)
	if v, ok := r.cache.Get(key); ok {
		return v.(*domain.ListProductsResponse), nil
	}
	resp, err := r.inner.List(ctx, req)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, resp, prodListTTL)
	return resp, nil
}
```

**Step 3: Add list cache invalidation**

Update `invalidateForCategory` to also clear product list entries:

```go
func (r *CachedProductRepository) invalidateForCategory(categoryID string) {
	r.cache.DeletePrefix(prodCatPrefix(categoryID))
	r.cache.Delete(prodAttrKey(categoryID))
	r.cache.DeletePrefix(prodListPrefix)
}
```

**Step 4: Verify it compiles**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 5: Run existing tests**

Run: `cd handloom-admin && go test ./internal/repository/postgres/... -v -count=1`
Expected: All existing tests pass (there may not be tests for the cached wrapper specifically, but we verify no regressions)

**Step 6: Commit**

```bash
cd handloom-admin
git add internal/repository/postgres/cached_product_repo.go
git commit -m "feat(cache): add product list query caching with hashed filter key"
```

---

### Task 5: Add Cache-Control HTTP Middleware

**Files:**
- Create: `handloom-admin/internal/middleware/cache_control.go`

**Step 1: Write the middleware**

```go
package middleware

import (
	"net/http"
	"strings"
)

// CacheControl returns a middleware that sets Cache-Control headers on responses.
// Pass "no-store" to disable caching, or "public, max-age=3600" for 1hr caching.
func CacheControl(value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only set cache headers on GET requests
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", value)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CatalogCacheControl returns a middleware that applies cache headers to catalog routes.
// The availability endpoint gets "no-store" (needs real-time data),
// all other catalog GET routes get the specified cache value.
func CatalogCacheControl(cacheValue string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				if strings.HasSuffix(r.URL.Path, "/availability") {
					w.Header().Set("Cache-Control", "no-store")
				} else {
					w.Header().Set("Cache-Control", cacheValue)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

**Step 2: Verify it compiles**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
cd handloom-admin
git add internal/middleware/cache_control.go
git commit -m "feat(cache): add Cache-Control HTTP middleware"
```

---

### Task 6: Apply Cache Middleware to Store Catalog Routes

**Files:**
- Modify: `handloom-admin/internal/handler/store/catalog_handler.go:43-54`

**Step 1: Add cache headers in the Routes() method**

The `CatalogHandler.Routes()` method builds the Chi router. Add the middleware there since it already has access to the route definitions. Import the middleware package and apply it:

Update the `Routes()` method in `catalog_handler.go`:

```go
import (
	// ... existing imports ...
	"github.com/handloom/admin/internal/middleware"
)
```

```go
func (h *CatalogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.CatalogCacheControl("public, max-age=3600"))

	r.Get("/categories", h.ListCategories)
	r.Get("/categories/{idOrSlug}", h.GetCategory)
	r.Get("/products", h.ListProducts)
	r.Get("/products/search", h.SearchProducts)
	r.Get("/products/{idOrSlug}", h.GetProduct)
	r.Get("/products/{id}/availability", h.CheckAvailability)

	return r
}
```

**Step 2: Check for import cycle**

The `store` package importing `middleware` must not create a cycle. Verify:

Run: `cd handloom-admin && go build ./internal/handler/store/...`
Expected: No import cycle errors. The `middleware` package does not import `handler/store`, so this is safe.

**Step 3: Verify full build**

Run: `cd handloom-admin && go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
cd handloom-admin
git add internal/handler/store/catalog_handler.go
git commit -m "feat(cache): apply Cache-Control headers to store catalog routes"
```

---

### Task 7: Update Next.js Server-Side Revalidation to 1 Hour

**Files:**
- Modify: `homechrome-store/src/app/categories/page.tsx:16`
- Modify: `homechrome-store/src/app/c/[slug]/page.tsx:17,31`
- Modify: `homechrome-store/src/app/p/[slug]/page.tsx:17`

**Step 1: Update categories page revalidation**

In `homechrome-store/src/app/categories/page.tsx`, change `revalidate: 300` to `revalidate: 3600`:

```typescript
const res = await fetch(`${API_BASE}/api/v1/store/catalog/categories`, {
  next: { revalidate: 3600 },
});
```

**Step 2: Update category detail + products page revalidation**

In `homechrome-store/src/app/c/[slug]/page.tsx`, change both `revalidate: 300` to `revalidate: 3600`:

```typescript
// getCategory function
const res = await fetch(`${API_BASE}/api/v1/store/catalog/categories/${slug}`, {
  next: { revalidate: 3600 },
});
```

```typescript
// getCategoryProducts function
const res = await fetch(
  `${API_BASE}/api/v1/store/catalog/products?category_id=${categoryId}`,
  { next: { revalidate: 3600 } },
);
```

**Step 3: Update product detail page revalidation**

In `homechrome-store/src/app/p/[slug]/page.tsx`, change `revalidate: 120` to `revalidate: 3600`:

```typescript
const res = await fetch(`${API_BASE}/api/v1/store/catalog/products/${slug}`, {
  next: { revalidate: 3600 },
});
```

**Step 4: Verify build**

Run: `cd homechrome-store && npm run build`
Expected: Build succeeds

**Step 5: Commit**

```bash
cd homechrome-store
git add src/app/categories/page.tsx src/app/c/\[slug\]/page.tsx src/app/p/\[slug\]/page.tsx
git commit -m "feat(cache): increase Next.js ISR revalidation to 1 hour"
```

---

### Task 8: Bump React Query Default staleTime

The storefront uses React Query for client-side data (cart, auth). The default `staleTime` of 60s is fine for those, but we should document that catalog data is served via server components and doesn't use React Query. No change needed here since catalog fetching uses Next.js server `fetch()`, not React Query hooks.

**Skip this task** — the storefront already uses the right pattern (server components with ISR). React Query's `staleTime` only affects client-side hooks (cart, auth), which should remain at 60s.

---

### Task 9: Run Full Test Suite and Verify

**Files:** None (verification only)

**Step 1: Run backend tests**

Run: `cd handloom-admin && make test`
Expected: All tests pass

**Step 2: Run backend lint**

Run: `cd handloom-admin && golangci-lint run`
Expected: No new warnings

**Step 3: Run frontend checks**

Run: `cd homechrome-store && npm run lint`
Expected: No lint errors

**Step 4: Manual smoke test (optional)**

Start the local backend and storefront:
```bash
# Terminal 1
cd handloom-admin && make run

# Terminal 2
cd homechrome-store && npm run dev
```

Verify:
- Visit `http://localhost:3000/categories` — check response headers include `Cache-Control: public, max-age=3600`
- Visit `http://localhost:3000/categories` again — should be instant (cached)
- Check product listing page headers similarly
- Check availability endpoint has `Cache-Control: no-store`
