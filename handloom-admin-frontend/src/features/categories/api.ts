import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Category, CategoryAttribute, CreateCategoryRequest } from './types';

export const categoriesApi = {
  list: async (
    params?: PaginationParams & { status?: string; search?: string }
  ): Promise<ListResponse<Category>> => {
    const response = await apiClient.get(ROUTES.CATEGORIES.LIST, { params });
    return normalizeListResponse<Category>(response.data as Record<string, unknown>, 'categories');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Category>(ROUTES.CATEGORIES.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateCategoryRequest) => {
    const response = await apiClient.post<Category>(ROUTES.CATEGORIES.LIST, data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCategoryRequest>) => {
    const response = await apiClient.patch<Category>(ROUTES.CATEGORIES.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.CATEGORIES.DETAIL(id));
  },

  addAttribute: async (id: string, attribute: CategoryAttribute) => {
    const response = await apiClient.post<{ attribute: CategoryAttribute }>(
      ROUTES.CATEGORIES.ATTRIBUTES(id),
      attribute
    );
    return response.data;
  },

  updateAttribute: async (id: string, attrName: string, attribute: Partial<CategoryAttribute>) => {
    const response = await apiClient.patch(
      ROUTES.CATEGORIES.ATTRIBUTE_DETAIL(id, attrName),
      attribute
    );
    return response.data;
  },

  deleteAttribute: async (id: string, attrName: string) => {
    await apiClient.delete(ROUTES.CATEGORIES.ATTRIBUTE_DETAIL(id, attrName));
  },

  getAttributes: async (
    id: string
  ): Promise<{ own_attributes: CategoryAttribute[]; total_count: number }> => {
    const response = await apiClient.get<{
      own_attributes: CategoryAttribute[];
      total_count: number;
    }>(ROUTES.CATEGORIES.ATTRIBUTES(id));
    return response.data;
  },
};
