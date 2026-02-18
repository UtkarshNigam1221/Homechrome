import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { BulkOperation } from './types';

export const bulkApi = {
  list: async (
    params?: PaginationParams & { type?: string; entity_type?: string; status?: string }
  ): Promise<ListResponse<BulkOperation>> => {
    const response = await apiClient.get('/admin/bulk', { params });
    return normalizeListResponse<BulkOperation>(
      response.data as Record<string, unknown>,
      'operations'
    );
  },

  get: async (id: string) => {
    const response = await apiClient.get<BulkOperation>(`/admin/bulk/${id}`);
    return response.data;
  },

  importProducts: async (fileUrl: string, mapping?: Record<string, string>) => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/products/import', {
      file_url: fileUrl,
      mapping,
    });
    return response.data;
  },

  updateInventory: async (
    fileUrl?: string,
    updates?: { product_id: string; quantity: number }[]
  ) => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/inventory/update', {
      file_url: fileUrl,
      updates,
    });
    return response.data;
  },

  exportData: async (entityType: string, filters?: Record<string, unknown>, format = 'CSV') => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/export', {
      entity_type: entityType,
      filters,
      format,
    });
    return response.data;
  },

  cancel: async (id: string) => {
    await apiClient.post(`/admin/bulk/${id}/cancel`);
  },

  getUploadUrl: async (entityType: string, filename: string) => {
    const response = await apiClient.post('/admin/bulk/upload-url', {
      entity_type: entityType,
      filename,
    });
    return response.data;
  },

  getDownloadUrl: async (id: string) => {
    const response = await apiClient.get(`/admin/bulk/${id}/download`);
    return response.data;
  },
};
