# Fix 429 Too Many Requests Errors — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate 429 errors from the storefront by fixing frontend request amplification and adding proper retry/backoff handling.

**Architecture:** Three layers of defense: (1) reduce outgoing requests via debounce + deduplication, (2) cancel superseded in-flight requests via AbortController, (3) gracefully handle 429 responses with exponential backoff retry in the axios client. React Query retry config updated to respect 429 backoff.

**Tech Stack:** Axios interceptors, AbortController, setTimeout debounce, React Query retryDelay

---

## Root Cause Summary

1. **Double API calls per filter change** — `CategoryProductsView` and `ProductsView` both set filter state AND push URL on every change. The URL change triggers a sync useEffect that re-sets filters with a new object reference, causing the product-fetch useEffect to fire twice.
2. **No debounce** — each checkbox/toggle fires an immediate API call.
3. **No 429 handling** — `api.ts` only handles 401; 429 responses are rejected immediately.
4. **No request cancellation** — old in-flight requests aren't aborted when new ones start.

---

### Task 1: Add 429 retry with exponential backoff to API client

**Files:**
- Modify: `src/lib/api.ts`

**Step 1: Add 429 retry interceptor**

Add a new error interceptor that detects 429 responses and retries with exponential backoff. Place it before the existing 401 interceptor by adding it as a second error handler in the response interceptor chain.

```typescript
import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: {
    limit: number;
    next_cursor: string;
    has_more: boolean;
  };
}

const MAX_429_RETRIES = 3;
const BASE_BACKOFF_MS = 1000;

// Always use relative URLs so requests go through Next.js rewrites (same-origin).
// NEXT_PUBLIC_API_URL is for server-side fetches only (page.tsx files).
const client: AxiosInstance = axios.create({
  baseURL: '',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Response interceptor to unwrap envelope
client.interceptors.response.use(
  (response) => {
    if (response.data?.data !== undefined) {
      return { ...response, data: response.data.data };
    }
    return response;
  },
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
      _retryCount?: number;
    };

    // --- 429 retry with exponential backoff ---
    if (error.response?.status === 429) {
      const retryCount = originalRequest._retryCount ?? 0;
      if (retryCount < MAX_429_RETRIES) {
        originalRequest._retryCount = retryCount + 1;
        const retryAfter = error.response.headers['retry-after'];
        const delayMs = retryAfter
          ? parseInt(retryAfter, 10) * 1000
          : BASE_BACKOFF_MS * Math.pow(2, retryCount);
        await sleep(delayMs);
        return client(originalRequest);
      }
      // Exhausted retries — fall through to reject
    }

    // --- 401 token refresh ---
    if (error.response?.status === 401 && !originalRequest._retry) {
      const isAuthCheck = originalRequest.url?.includes('/store/me');
      if (isAuthCheck) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;
      try {
        await client.post('/api/v1/store/auth/refresh');
        return client(originalRequest);
      } catch {
        if (
          typeof window !== 'undefined' &&
          !window.location.pathname.startsWith('/login')
        ) {
          window.location.href = '/login';
        }
      }
    }

    return Promise.reject(error);
  },
);

export const api = {
  get: <T>(url: string, params?: Record<string, unknown>) =>
    client.get<T>(url, { params }),
  post: <T>(url: string, data?: unknown) => client.post<T>(url, data),
  patch: <T>(url: string, data?: unknown) => client.patch<T>(url, data),
  put: <T>(url: string, data?: unknown) => client.put<T>(url, data),
  delete: <T>(url: string) => client.delete<T>(url),
};

export default api;
```

**Step 2: Verify build passes**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npx next build`
Expected: Build succeeds with no type errors.

**Step 3: Commit**

```bash
git add src/lib/api.ts
git commit -m "fix: add 429 retry with exponential backoff to API client"
```

---

### Task 2: Fix double-fetch and add debounce in CategoryProductsView

**Files:**
- Modify: `src/app/c/[slug]/CategoryProductsView.tsx`

**Problem:** The filter change handler sets `filters` state AND calls `router.push()`. The URL change triggers the `searchParams` sync useEffect which re-sets `filters` with a new object reference, firing the product-fetch useEffect a second time.

**Fix:**
1. Use a ref (`isProgrammaticNav`) to track when URL changes originate from user filter interaction (skip the sync in that case).
2. Wrap the product-fetch API call in a `setTimeout` debounce (300ms) with cleanup.
3. Use `AbortController` to cancel superseded in-flight requests.

**Step 1: Apply the fix**

Replace the three useEffects (URL sync at ~line 110, product fetch at ~line 116) with this corrected version:

```typescript
// --- replace the URL→filter sync useEffect and the product-fetch useEffect ---

const isProgrammaticNav = useRef(false);
const abortRef = useRef<AbortController | null>(null);

// Sync URL → filter state only on browser back/forward (not programmatic pushes)
useEffect(() => {
  if (isProgrammaticNav.current) {
    isProgrammaticNav.current = false;
    return;
  }
  const parsed = parseFiltersFromParams(searchParams);
  setFilters(parsed);
}, [searchParams]);

// Re-fetch products when filters change (debounced, with abort)
useEffect(() => {
  if (isInitialMount.current) {
    isInitialMount.current = false;
    return;
  }

  const controller = new AbortController();
  abortRef.current?.abort();
  abortRef.current = controller;

  const timer = setTimeout(() => {
    const params = new URLSearchParams();
    params.set('category_id', category.id);
    if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
    if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
    if (filters.inStockOnly) params.set('in_stock', 'true');
    if (Object.keys(filters.attributeFilters).length > 0) {
      params.set('attribute_filters', JSON.stringify(filters.attributeFilters));
    }

    setLoading(true);
    api
      .get<Product[]>(`/api/v1/store/catalog/products?${params.toString()}`)
      .then((res) => {
        if (!controller.signal.aborted) {
          setProducts(Array.isArray(res.data) ? res.data : []);
        }
      })
      .catch(() => {
        // keep existing products on error
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });
  }, 300);

  return () => {
    clearTimeout(timer);
    controller.abort();
  };
}, [category.id, filters]);
```

Also update `handleFiltersChange` to set the programmatic nav flag:

```typescript
const handleFiltersChange = useCallback(
  (newFilters: FilterValues) => {
    isProgrammaticNav.current = true;
    setFilters(newFilters);
    const urlParams = filtersToParams(newFilters);
    const qs = urlParams.toString();
    router.push(`/c/${category.slug}${qs ? `?${qs}` : ''}`, { scroll: false });
  },
  [router, category.slug],
);
```

**Step 2: Verify build passes**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npx next build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add src/app/c/[slug]/CategoryProductsView.tsx
git commit -m "fix: debounce filter API calls and prevent double-fetch in category view"
```

---

### Task 3: Fix double-fetch and add debounce in ProductsView

**Files:**
- Modify: `src/app/products/ProductsView.tsx`

**Same pattern as Task 2.** Apply identical fix to `ProductsView`.

**Step 1: Apply the fix**

Add refs at the top of the component (after `isInitialMount`):

```typescript
const isProgrammaticNav = useRef(false);
const abortRef = useRef<AbortController | null>(null);
```

Replace the `searchParams` sync useEffect (~line 69-73):

```typescript
// Sync input with URL changes (only on browser back/forward)
useEffect(() => {
  if (isProgrammaticNav.current) {
    isProgrammaticNav.current = false;
    return;
  }
  const urlSearch = searchParams.get('search') || '';
  setSearchQuery(urlSearch);
  setFilters(parseFiltersFromParams(searchParams));
}, [searchParams]);
```

Replace the product-fetch useEffect (~line 76-99):

```typescript
// Re-fetch products when filters change (debounced, with abort)
useEffect(() => {
  if (isInitialMount.current) {
    isInitialMount.current = false;
    return;
  }

  const controller = new AbortController();
  abortRef.current?.abort();
  abortRef.current = controller;

  const timer = setTimeout(() => {
    const params = new URLSearchParams();
    const search = searchParams.get('search');
    if (search) params.set('search', search);
    if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
    if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
    if (filters.inStockOnly) params.set('in_stock', 'true');

    setLoading(true);
    api
      .get<Product[]>(`/api/v1/store/catalog/products?${params.toString()}`)
      .then((res) => {
        if (!controller.signal.aborted) {
          setProducts(Array.isArray(res.data) ? res.data : []);
        }
      })
      .catch(() => {
        // keep existing products on error
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });
  }, 300);

  return () => {
    clearTimeout(timer);
    controller.abort();
  };
}, [filters, searchParams]);
```

Update `handleFiltersChange` to set the programmatic nav flag:

```typescript
const handleFiltersChange = useCallback(
  (newFilters: FilterValues) => {
    isProgrammaticNav.current = true;
    setFilters(newFilters);
    const params = new URLSearchParams();
    const search = searchParams.get('search');
    if (search) params.set('search', search);
    if (newFilters.minPrice !== null) params.set('min_price', String(newFilters.minPrice));
    if (newFilters.maxPrice !== null) params.set('max_price', String(newFilters.maxPrice));
    if (newFilters.inStockOnly) params.set('in_stock', 'true');
    const qs = params.toString();
    router.push(`/products${qs ? `?${qs}` : ''}`, { scroll: false });
  },
  [router, searchParams],
);
```

**Step 2: Verify build passes**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npx next build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add src/app/products/ProductsView.tsx
git commit -m "fix: debounce filter API calls and prevent double-fetch in products view"
```

---

### Task 4: Configure React Query to back off on 429

**Files:**
- Modify: `src/app/providers.tsx`

**Step 1: Add retryDelay with exponential backoff**

Update the `QueryClient` config to use exponential backoff and skip retrying 429s (the axios interceptor handles those):

```typescript
const [queryClient] = useState(
  () =>
    new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 60 * 1000,
          retry: (failureCount, error) => {
            // Don't retry 429s at React Query level — axios interceptor handles backoff
            if (error instanceof Error && 'status' in error && (error as any).status === 429) {
              return false;
            }
            return failureCount < 1;
          },
        },
      },
    }),
);
```

**Step 2: Verify build passes**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npx next build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add src/app/providers.tsx
git commit -m "fix: skip React Query retry on 429 (axios handles backoff)"
```
