import type { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import axios from 'axios';

import type { ListResponse } from '@/shared/types/common';
import { useAuthStore } from '@/shared/stores/authStore';

// In dev, Vite proxy handles /admin/* → API Gateway (same-origin, cookies work).
// In production builds, VITE_API_URL points to the real API.
const API_BASE_URL = import.meta.env.DEV ? '' : import.meta.env.VITE_API_URL || '';

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
  timeout: 30000,
});

// Token refresh mutex to prevent concurrent refresh attempts
let isRefreshing = false;
let failedQueue: Array<{
  resolve: () => void;
  reject: (error: unknown) => void;
}> = [];

function processQueue(error: unknown) {
  failedQueue.forEach((pending) => {
    if (error) {
      pending.reject(error);
    } else {
      pending.resolve();
    }
  });
  failedQueue = [];
}

// Response interceptor: unwrap API envelope and handle token refresh
apiClient.interceptors.response.use(
  (response) => {
    // Auto-unwrap standard API envelope:
    //   Single: { success, data: T }           → T
    //   List:   { success, data: T[], meta: M } → { items: T[], pagination: M }
    const d = response.data;
    if (d && typeof d === 'object' && !Array.isArray(d) && 'success' in d && 'data' in d) {
      if (Array.isArray(d.data) && 'meta' in d) {
        response.data = { items: d.data, pagination: d.meta };
      } else {
        response.data = d.data;
      }
    }
    return response;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status === 401 && !originalRequest._retry) {
      // If already refreshing, queue this request
      if (isRefreshing) {
        return new Promise<void>((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then(() => {
          return apiClient(originalRequest);
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // Call refresh endpoint — cookie is sent automatically
        await axios.post(`${API_BASE_URL}/admin/auth/refresh`, null, {
          withCredentials: true,
          headers: { 'Content-Type': 'application/json' },
        });

        processQueue(null);
        return apiClient(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError);
        useAuthStore.getState().logout();
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;

// Helper function to handle API errors
export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as
      | { error?: { message?: string }; message?: string }
      | undefined;
    return data?.error?.message || data?.message || error.message || 'An unexpected error occurred';
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'An unexpected error occurred';
}

// Normalize backend list responses that may use different keys (e.g. 'products', 'orders')
// into a consistent { items, pagination } shape.
export function normalizeListResponse<T>(
  data: Record<string, unknown>,
  key: string
): ListResponse<T> {
  const items = (data[key] || data.items || data.data || []) as T[];
  const pagination = (data.pagination as ListResponse<T>['pagination']) || {
    limit: 10,
    has_more: false,
  };
  return { items, pagination };
}
