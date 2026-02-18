import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Product } from '@/features/products/types';

import type { Artisan, CreateArtisanRequest } from './types';

export const artisansApi = {
  list: async (
    params?: PaginationParams & {
      craft_type?: string;
      location?: string;
      status?: string;
      search?: string;
    }
  ): Promise<ListResponse<Artisan>> => {
    const response = await apiClient.get('/admin/artisans', { params });
    return normalizeListResponse<Artisan>(response.data as Record<string, unknown>, 'artisans');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Artisan>(`/admin/artisans/${id}`);
    return response.data;
  },

  create: async (data: CreateArtisanRequest) => {
    const response = await apiClient.post<Artisan>('/admin/artisans', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateArtisanRequest>) => {
    const response = await apiClient.patch<Artisan>(`/admin/artisans/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/artisans/${id}`);
  },

  updateStatus: async (id: string, status: string) => {
    await apiClient.patch(`/admin/artisans/${id}/status`, { status });
  },

  getProducts: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Product>>(`/admin/artisans/${id}/products`, {
      params,
    });
    return response.data;
  },
};
