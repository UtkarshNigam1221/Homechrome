import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';

import { ROUTES } from '@/lib/routes';
import { SearchResponse } from '@/lib/semantic-search';
import { buildVisitorHeader, VISITOR_HEADER } from '@/lib/visitor-context';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: {
    limit: number;
    next_cursor: string;
    has_more: boolean;
  };
}

// Always use relative URLs so requests go through Next.js rewrites (same-origin).
// NEXT_PUBLIC_API_URL is for server-side fetches only (page.tsx files).
const client: AxiosInstance = axios.create({
  baseURL: '',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

// Request interceptor — attach the single visitor-attribution header. The
// Next.js middleware on the server side merges in CloudFront viewer-geo
// bits before forwarding to backend; this side only knows browser-side
// fields (device + sticky UTM tuple). Skipped on SSR since there's no
// window context to read.
client.interceptors.request.use((config) => {
  if (typeof window === 'undefined') return config;
  const visitor = buildVisitorHeader();
  if (visitor) config.headers.set(VISITOR_HEADER, visitor);
  return config;
});

// 429 retry configuration
const MAX_429_RETRIES = 3;
const BASE_BACKOFF_MS = 1000;

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

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

    // 429 Too Many Requests — retry with exponential backoff
    if (error.response?.status === 429) {
      const retryCount = originalRequest._retryCount ?? 0;
      if (retryCount < MAX_429_RETRIES) {
        originalRequest._retryCount = retryCount + 1;

        const retryAfterHeader = error.response.headers['retry-after'];
        const parsed = retryAfterHeader ? Number(retryAfterHeader) : NaN;
        const baseDelay =
          Number.isFinite(parsed) && parsed > 0
            ? parsed * 1000
            : BASE_BACKOFF_MS * Math.pow(2, retryCount);
        const delayMs = baseDelay + Math.random() * baseDelay * 0.1;

        await sleep(delayMs);
        return client(originalRequest);
      }
    }

    // Skip refresh for the refresh endpoint itself to prevent infinite loop
    const isRefreshRequest = originalRequest.url?.includes('/auth/refresh');

    if (error.response?.status === 401 && !originalRequest._retry && !isRefreshRequest) {
      originalRequest._retry = true;
      try {
        await client.post(ROUTES.AUTH.REFRESH);
        return client(originalRequest);
      } catch {
        // Refresh failed — redirect to login unless on auth-check, login, or confirmation page
        const isAuthCheck = originalRequest.url?.includes('/store/me');
        const isConfirmation = typeof window !== 'undefined' && window.location.pathname.startsWith('/checkout/confirmation');
        if (
          !isAuthCheck &&
          !isConfirmation &&
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

const api = {
  get: <T>(url: string, params?: Record<string, unknown>) =>
    client.get<T>(url, { params }),
  post: <T>(url: string, data?: unknown) => client.post<T>(url, data),
  patch: <T>(url: string, data?: unknown) => client.patch<T>(url, data),
  put: <T>(url: string, data?: unknown) => client.put<T>(url, data),
  delete: <T>(url: string) => client.delete<T>(url),
};

export default api;

export interface SearchParams {
  q?: string;
  limit?: number;
  cursor?: string;
  category_id?: string;
  min_price?: number;
  max_price?: number;
  in_stock?: boolean;
  material?: string;
  color?: string;
  /** Attribute filters: { material: ['silk'], color: ['red'] } → af_material=silk&af_color=red */
  attribute_filters?: Record<string, string[]>;
}

/**
 * Searches products via the embedder Lambda mounted under the existing
 * REST API at GET /api/v1/store/catalog/search. Hybrid semantic + tsvector +
 * trigram scoring backed by pgvector + HNSW. Accepts the same filters as the
 * legacy /products endpoint (category_id, min/max_price, in_stock, material,
 * color, af_*) so the storefront's useFilteredProducts hook can hit one
 * endpoint for both search-with-filters and filter-only listings.
 *
 * Uses raw fetch (not the shared axios client) because the shared response
 * interceptor unwraps {success, data, meta} → data only, which would strip
 * the SearchResponse meta block. Also keeps a single 5xx retry to absorb
 * embedder cold starts (~5–7 s).
 */
export async function searchProducts(params: SearchParams = {}): Promise<SearchResponse> {
  const url = buildSearchURL(params);
  const init: RequestInit = { method: 'GET', headers: { Accept: 'application/json' } };

  let res = await fetch(url, init);
  if (!res.ok && res.status >= 500) {
    await new Promise((r) => setTimeout(r, 1000));
    res = await fetch(url, init);
  }
  if (!res.ok) {
    throw new Error(`search failed: ${res.status}`);
  }
  return (await res.json()) as SearchResponse;
}

/** buildSearchURL serializes SearchParams to a query string. Exported for SSR
 *  callers that need to call the embedder via absolute URL (e.g. page.tsx). */
export function buildSearchURL(params: SearchParams): string {
  const qs = new URLSearchParams();
  if (params.q) qs.set('q', params.q);
  if (params.limit !== undefined) qs.set('limit', String(params.limit));
  if (params.cursor) qs.set('cursor', params.cursor);
  if (params.category_id) qs.set('category_id', params.category_id);
  if (params.min_price !== undefined) qs.set('min_price', String(params.min_price));
  if (params.max_price !== undefined) qs.set('max_price', String(params.max_price));
  if (params.in_stock) qs.set('in_stock', 'true');
  if (params.material) qs.set('material', params.material);
  if (params.color) qs.set('color', params.color);
  for (const [name, values] of Object.entries(params.attribute_filters ?? {})) {
    if (values.length > 0) qs.set(`af_${name}`, values.join(','));
  }
  const s = qs.toString();
  return s ? `${ROUTES.CATALOG.SEARCH}?${s}` : ROUTES.CATALOG.SEARCH;
}
