# Storefront Request Flows

How SSR, caching, and image optimization work end-to-end in the homechrome-store deployment.

## Infrastructure Overview

```
Browser
  │
  ▼
CloudFront (CDN, ap-south-1 edge)
  │
  ├─ /_next/static/*    → S3 (immutable hashed assets, 1yr cache)
  ├─ /_next/image        → Image Optimization Lambda (256MB, Sharp)
  ├─ /favicon.ico, etc.  → S3 (public static files)
  └─ /* (everything else) → Server Lambda (128MB, SSR + ISR)
                              │
                              ├─ reads/writes → S3 (_cache/ prefix)
                              └─ proxies /api/* → Backend API
```

---

## Flow 1: First Visit to a Product Page (`/p/silk-saree`)

**Cold start — no cached HTML exists yet.**

```
Browser → CloudFront → Server Lambda → Next.js renders → S3 cache write → Response
```

1. Browser requests `https://homechrome.in/p/silk-saree`
2. CloudFront matches the default behavior (`*`), checks its edge cache — **miss**
3. CloudFront forwards to Server Lambda via Function URL
   - Injects `x-forwarded-host: homechrome.in` via CloudFront Function
   - Forwards `accept`, `rsc`, `next-router-*` headers (part of cache key)
   - Forwards cookies to Lambda (but cookies are NOT in the cache key)
4. Server Lambda receives the request:
   - Checks S3 cache (`_cache/` prefix) for pre-rendered HTML — **miss**
   - Invokes Next.js server-side rendering
   - Fetches product data from backend API (`/api/v1/store/catalog/products/silk-saree`)
   - Renders full HTML with product details, images, JSON-LD structured data
   - Writes rendered HTML to S3 cache with metadata (revalidate = 3600s)
   - Streams response back with `Cache-Control` header
5. CloudFront receives the response:
   - Stores in edge cache (respects `Cache-Control` from Lambda)
   - Applies security headers (HSTS, X-Frame-Options, etc.)
   - Compresses with brotli/gzip
6. Browser receives complete HTML — page renders immediately

**Total time:** ~500-1500ms (cold Lambda + SSR + API call)

---

## Flow 2: Subsequent Visit to Same Product Page

**Warm path — HTML cached at CloudFront edge.**

```
Browser → CloudFront (edge cache hit) → Response
```

1. Browser requests `/p/silk-saree`
2. CloudFront matches default behavior, checks edge cache — **hit**
3. Returns cached HTML directly from edge

**Total time:** ~10-50ms (CDN edge latency only)

Lambda is never invoked. This continues until the cache expires (controlled by the `Cache-Control` header from the original Lambda response).

---

## Flow 3: ISR Revalidation (Cache Expired)

**After the revalidate interval (3600s for product pages), the cache is stale.**

```
Browser → CloudFront (stale) → Server Lambda → Re-renders → Updates S3 + edge cache
```

1. Browser requests `/p/silk-saree` after 1 hour
2. CloudFront's cached copy has expired (TTL from `Cache-Control` elapsed)
3. CloudFront forwards to Server Lambda
4. Lambda checks S3 cache — finds stale entry, re-renders:
   - Fetches fresh product data from backend API
   - Renders updated HTML
   - Writes new HTML to S3 cache (resets the 3600s timer)
   - Returns fresh response
5. CloudFront caches the new response at the edge
6. Browser gets updated content

**Revalidate intervals by page:**

| Route | Interval | Why |
|-------|----------|-----|
| `/` (home) | 1 hour | Featured products change occasionally |
| `/c/[slug]` (category) | 1 hour | Products within categories are fairly stable |
| `/p/[slug]` (product) | 1 hour | Price/stock changes are infrequent |
| `/products` (search) | 60 seconds | Search results should reflect recent additions |
| `/sitemap.ts` | 1 hour | SEO sitemap regeneration |

---

## Flow 4: Static Asset Request (`/_next/static/chunks/app-abc123.js`)

**Immutable content-hashed files — cached for 1 year.**

```
Browser → CloudFront (edge hit or S3 origin) → Response
```

1. Browser requests `/_next/static/chunks/app-abc123.js`
2. CloudFront matches the `/_next/static/*` behavior
3. If cached at edge → returns immediately (~10ms)
4. If not cached → fetches from S3 bucket (`_assets/_next/static/...`)
5. Cached for 1 year (AWS `CACHING_OPTIMIZED` policy)

These files have content hashes in their filenames. When code changes, Next.js generates new filenames, so old cached files are never served for new deployments.

**No Lambda involved.** Pure S3 → CloudFront.

---

## Flow 5: Image Optimization (`/_next/image?url=...&w=640&q=75`)

**Next.js `<Image>` component requests optimized images.**

```
Browser → CloudFront → Image Lambda → Reads original from S3 → Resizes/converts → Response
```

1. A page renders `<Image src="/products/silk-saree.jpg" width={640} />` which generates:
   ```
   /_next/image?url=%2Fproducts%2Fsilk-saree.jpg&w=640&q=75
   ```
2. CloudFront matches the `/_next/image` behavior, checks edge cache
3. **Cache miss** (first request for this size/quality combo):
   - Forwards to Image Optimization Lambda via Function URL
   - Lambda reads the original image from S3 (`_assets/` prefix)
   - Sharp (native image library) resizes to width=640, quality=75
   - Converts to AVIF or WebP based on browser `Accept` header
   - Returns optimized image with appropriate `Cache-Control`
4. **Cache hit** (subsequent requests):
   - CloudFront serves directly from edge cache
   - Same image at same dimensions = same cache key

**Why 256MB for this Lambda?** Sharp (the image processing library) needs ~200MB minimum to load and process images efficiently. 128MB would cause out-of-memory errors on larger images.

**What triggers image optimization:**

```tsx
// ProductCard — responsive sizes based on viewport
<Image
  src={product.primaryImage.url}
  fill
  sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
/>

// ProductDetail — priority loading for LCP
<Image
  src={selectedImage.url}
  fill
  sizes="(max-width: 1024px) 100vw, 50vw"
  priority  // preloaded, no lazy loading
/>
```

The `sizes` prop tells the browser which width to request. A 640px-wide phone requesting a 50vw image asks for a 320px-wide image — the Lambda resizes accordingly.

---

## Flow 6: Client-Side Navigation (React Server Components)

**After initial page load, navigation uses RSC streaming — no full page reload.**

```
Browser → CloudFront → Server Lambda → RSC payload (JSON-like) → React hydration
```

1. User clicks a link (e.g., from category page to product page)
2. Next.js client router intercepts the click
3. Browser sends a fetch request with special headers:
   - `rsc: 1` (requesting React Server Component payload, not HTML)
   - `next-router-state-tree: ...` (current router state)
   - `next-router-prefetch: 1` (if prefetching on hover)
4. CloudFront includes these headers in the cache key (configured in `serverCachePolicy`)
5. This means RSC payloads are cached separately from full HTML responses
6. Server Lambda returns a streaming RSC payload (not HTML)
7. React on the client patches the DOM — no full page reload

**Why `rsc` is in the cache key:** A request with `rsc: 1` returns a JSON-like RSC payload. A request without it returns full HTML. Same URL, different responses — so the header must be part of the cache key to avoid serving the wrong format.

---

## Flow 7: API Calls from Client (Auth, Cart, Checkout)

**Client-side API calls are proxied through Next.js rewrites.**

```
Browser JS → /api/v1/store/cart → Next.js rewrite → Backend API → Response
```

1. Client code calls `/api/v1/store/cart/items` (relative URL via axios)
2. In production: Next.js rewrites `/api/*` → `https://api.homechrome.in/api/*`
3. In local dev: Vite proxies to `http://localhost:8081/api/*`
4. HttpOnly cookies are forwarded automatically (`withCredentials: true`)
5. Backend returns `{success: true, data: {...}}`
6. Axios response interceptor unwraps: `response.data.data` → `response.data`

**Auth flow specifics:**
- Login via phone OTP: `POST /api/v1/store/auth/send-otp` → `POST /api/v1/store/auth/verify-otp`
- JWT stored in HttpOnly cookie (set by backend, not accessible to JS)
- On 401: interceptor queues requests, calls `/api/v1/store/auth/refresh`, replays queue
- Cookies forwarded to Server Lambda but NOT in CloudFront cache key — so authenticated and unauthenticated users see the same cached HTML; personalization happens client-side

---

## Flow 8: Deployment & Cache Invalidation

**What happens when you run `npm run cdk:deploy:dev`:**

```
next build → open-next build → cdk deploy → S3 upload → CloudFront invalidation
```

1. `next build` — compiles pages, generates static HTML, creates ISR seeds
2. `npx open-next build` — transforms `.next/` into Lambda-compatible artifacts:
   - `.open-next/server-functions/default/` → Server Lambda code
   - `.open-next/image-optimization-function/` → Image Lambda code
   - `.open-next/assets/` → Static files for S3
   - `.open-next/cache/` → ISR cache seeds for S3
3. CDK deploy:
   - Uploads `.open-next/assets/` → S3 `_assets/` prefix (prune: true — old files deleted)
   - Uploads `.open-next/cache/` → S3 `_cache/` prefix (prune: false — preserves runtime ISR cache)
   - Updates Lambda function code
   - Invalidates CloudFront cache (`/*`) — all edge caches cleared
4. First request after deploy hits Lambda (cache miss), re-renders with new code

**Why `_cache/` is not pruned:** During runtime, the Server Lambda writes updated ISR cache entries to S3. Pruning would delete these runtime-generated cache entries, forcing all pages to re-render on the next deploy.

---

## Cache Key Composition

What makes two requests "different" from CloudFront's perspective:

| Factor | In Cache Key? | Example |
|--------|:---:|---------|
| URL path | Yes | `/p/silk-saree` vs `/p/cotton-kurta` |
| Query strings | Yes (all) | `?page=2` vs `?page=3` |
| `accept` header | Yes | AVIF-capable vs JPEG-only browser |
| `rsc` header | Yes | Full HTML vs RSC payload |
| `next-router-prefetch` | Yes | Prefetch vs full navigation |
| `next-router-state-tree` | Yes | Different router states |
| `next-url` | Yes | Internal Next.js routing |
| `x-prerender-revalidate` | Yes | ISR revalidation requests |
| Cookies | No | Same cache for logged-in and anonymous users |
| `Authorization` header | No | Not forwarded |

---

## Cost Model

At low traffic (~10K requests/month):

| Component | Cost Driver | Estimate |
|-----------|------------|----------|
| Server Lambda | 128MB × invocations | ~$0.20 |
| Image Lambda | 256MB × invocations (less frequent, cached) | ~$0.10 |
| CloudFront | Data transfer + requests | ~$1.00 |
| S3 | Storage + requests | ~$0.05 |
| **Total** | | **~$1-3/mo** |

Most requests are served from CloudFront edge cache — Lambda only runs on cache misses and ISR revalidations.
