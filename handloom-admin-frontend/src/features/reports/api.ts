import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Report } from './types';

export const reportsApi = {
  list: async (
    params?: PaginationParams & {
      type?: string;
      status?: string;
      start_date?: string;
      end_date?: string;
    }
  ): Promise<ListResponse<Report>> => {
    const response = await apiClient.get('/admin/reports', { params });
    return normalizeListResponse<Report>(response.data as Record<string, unknown>, 'reports');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Report>(`/admin/reports/${id}`);
    return response.data;
  },

  generate: async (type: string, filters?: Record<string, unknown>, format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports', {
      type,
      filters,
      format,
    });
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/reports/${id}`);
  },

  getDownloadUrl: async (id: string) => {
    const response = await apiClient.get(`/admin/reports/${id}/download`);
    return response.data;
  },

  generateSalesReport: async (startDate: string, endDate: string, format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports/sales', {
      start_date: startDate,
      end_date: endDate,
      format,
    });
    return response.data;
  },

  generateInventoryReport: async (format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports/inventory', { format });
    return response.data;
  },

  generateOrdersReport: async (
    startDate: string,
    endDate: string,
    status?: string,
    format = 'CSV'
  ) => {
    const response = await apiClient.post<Report>('/admin/reports/orders', {
      start_date: startDate,
      end_date: endDate,
      status,
      format,
    });
    return response.data;
  },
};
