import type { Inventory, InventoryTransaction } from '@/features/inventory/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateProductRequest, Product } from './types';

export const productsApi = {
  list: async (
    params?: PaginationParams & {
      category_id?: string;
      status?: string;
      min_price?: number;
      max_price?: number;
      in_stock?: boolean;
      low_stock?: boolean;
      search?: string;
      attribute_filters?: Record<string, string[]>;
    }
  ) => {
    // Handle attribute_filters specially - needs to be serialized as JSON
    const { attribute_filters, ...restParams } = params || {};
    const queryParams: Record<string, unknown> = { ...restParams };
    if (attribute_filters && Object.keys(attribute_filters).length > 0) {
      queryParams.attribute_filters = JSON.stringify(attribute_filters);
    }
    const response = await apiClient.get('/admin/products', {
      params: queryParams,
    });
    return normalizeListResponse<Product>(response.data as Record<string, unknown>, 'products');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Product>(`/admin/products/${id}`);
    return response.data;
  },

  create: async (data: CreateProductRequest) => {
    const response = await apiClient.post<Product>('/admin/products', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateProductRequest>) => {
    const response = await apiClient.patch<Product>(`/admin/products/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/products/${id}`);
  },

  getInventory: async (id: string) => {
    const response = await apiClient.get<Inventory>(`/admin/products/${id}/inventory`);
    return response.data;
  },

  addStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/add`, {
      quantity,
      reason,
    });
    return response.data;
  },

  removeStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/remove`, {
      quantity,
      reason,
    });
    return response.data;
  },

  adjustStock: async (id: string, newQuantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/adjust`, {
      new_quantity: newQuantity,
      reason,
    });
    return response.data;
  },

  getInventoryTransactions: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<InventoryTransaction>>(
      `/admin/products/${id}/inventory/transactions`,
      { params }
    );
    return response.data;
  },

  getFilterOptions: async (categoryId: string): Promise<Record<string, string[]>> => {
    const response = await apiClient.get<Record<string, string[]>>(
      `/admin/products/filter-options/${categoryId}`
    );
    return response.data;
  },
};
