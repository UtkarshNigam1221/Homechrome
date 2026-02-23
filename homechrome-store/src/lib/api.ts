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

// Always use relative URLs so requests go through Next.js rewrites (same-origin).
// NEXT_PUBLIC_API_URL is for server-side fetches only (page.tsx files).
const client: AxiosInstance = axios.create({
  baseURL: '',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

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
    };

    if (error.response?.status === 401 && !originalRequest._retry) {
      // Skip refresh for auth-check calls — just let them fail quietly
      const isAuthCheck = originalRequest.url?.includes('/store/me');
      if (isAuthCheck) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;
      try {
        await client.post('/api/v1/store/auth/refresh');
        return client(originalRequest);
      } catch {
        // Refresh failed — redirect to login only if not already there
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
