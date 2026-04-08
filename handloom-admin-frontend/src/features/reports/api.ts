import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
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
    const response = await apiClient.get(ROUTES.REPORTS.LIST, { params });
    return normalizeListResponse<Report>(response.data as Record<string, unknown>, 'reports');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Report>(ROUTES.REPORTS.DETAIL(id));
    return response.data;
  },

  generate: async (type: string, filters?: Record<string, unknown>, format = 'CSV') => {
    const response = await apiClient.post<Report>(ROUTES.REPORTS.LIST, {
      type,
      filters,
      format,
    });
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.REPORTS.DETAIL(id));
  },

  getDownloadUrl: async (id: string) => {
    const response = await apiClient.get(ROUTES.REPORTS.DOWNLOAD(id));
    return response.data;
  },

  generateSalesReport: async (startDate: string, endDate: string, format = 'CSV') => {
    const response = await apiClient.post<Report>(ROUTES.REPORTS.SALES, {
      start_date: startDate,
      end_date: endDate,
      format,
    });
    return response.data;
  },

  generateInventoryReport: async (format = 'CSV') => {
    const response = await apiClient.post<Report>(ROUTES.REPORTS.INVENTORY, { format });
    return response.data;
  },

  generateOrdersReport: async (
    startDate: string,
    endDate: string,
    status?: string,
    format = 'CSV'
  ) => {
    const response = await apiClient.post<Report>(ROUTES.REPORTS.ORDERS, {
      start_date: startDate,
      end_date: endDate,
      status,
      format,
    });
    return response.data;
  },
};
