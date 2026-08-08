import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';

import { ROUTES } from '@/lib/routes';
import { buildVisitorHeader, VISITOR_HEADER } from '@/lib/visitor-context';
import { Product } from '@/types';

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

// The backend rotates refresh tokens: /auth/refresh mints a new pair and
// revokes the one it was handed. Two requests 401-ing together (a page load
// fires /me + /cart + /orders at once, all with the same expired access
// token) would each post their own refresh; the loser presents an
// already-revoked token, and the handler answers 401 *and* clears both auth
// cookies — deleting the session the winner just established. So all callers
// share one in-flight refresh.
let refreshInFlight: Promise<unknown> | null = null;

function refreshOnce(): Promise<unknown> {
  refreshInFlight ??= client
    .post(ROUTES.AUTH.REFRESH)
    .finally(() => {
      refreshInFlight = null;
    });
  return refreshInFlight;
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
        await refreshOnce();
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

/** One page of a catalog listing plus the cursor that fetches the next one. */
export interface ProductsPage {
  products: Product[];
  nextCursor?: string;
}

/**
 * Fetches one page of products from any listing endpoint (`/catalog/products`
 * or `/catalog/search` — identical envelope). Raw fetch, not the shared axios
 * client, because the response interceptor unwraps {success, data, meta} → data
 * and would strip the `meta.next_cursor` that infinite scroll pages on.
 *
 * `url` may be relative (client, via Next rewrites) or absolute (SSR).
 */
export async function fetchProductsPage(url: string): Promise<ProductsPage> {
  const res = await fetch(url, { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    throw new Error(`products fetch failed: ${res.status}`);
  }
  const json = (await res.json()) as ApiResponse<Product[]>;
  return { products: json.data ?? [], nextCursor: json.meta?.next_cursor || undefined };
}

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

/** Serializes SearchParams into a /catalog/search URL. */
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
