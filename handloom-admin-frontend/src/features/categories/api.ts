import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Category, CategoryAttribute, CreateCategoryRequest } from './types';

export const categoriesApi = {
  list: async (
    params?: PaginationParams & { status?: string }
  ): Promise<ListResponse<Category>> => {
    const response = await apiClient.get('/admin/categories', { params });
    return normalizeListResponse<Category>(response.data as Record<string, unknown>, 'categories');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Category>(`/admin/categories/${id}`);
    return response.data;
  },

  create: async (data: CreateCategoryRequest) => {
    const response = await apiClient.post<Category>('/admin/categories', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCategoryRequest>) => {
    const response = await apiClient.patch<Category>(`/admin/categories/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/categories/${id}`);
  },

  addAttribute: async (id: string, attribute: CategoryAttribute) => {
    const response = await apiClient.post<{ attribute: CategoryAttribute }>(
      `/admin/categories/${id}/attributes`,
      attribute
    );
    return response.data;
  },

  updateAttribute: async (id: string, attrName: string, attribute: Partial<CategoryAttribute>) => {
    const response = await apiClient.patch(
      `/admin/categories/${id}/attributes/${attrName}`,
      attribute
    );
    return response.data;
  },

  deleteAttribute: async (id: string, attrName: string) => {
    await apiClient.delete(`/admin/categories/${id}/attributes/${attrName}`);
  },

  getAttributes: async (
    id: string
  ): Promise<{ own_attributes: CategoryAttribute[]; total_count: number }> => {
    const response = await apiClient.get<{
      own_attributes: CategoryAttribute[];
      total_count: number;
    }>(`/admin/categories/${id}/attributes`);
    return response.data;
  },
};
