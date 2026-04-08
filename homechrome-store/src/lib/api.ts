import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';

import { ROUTES } from '@/lib/routes';

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
