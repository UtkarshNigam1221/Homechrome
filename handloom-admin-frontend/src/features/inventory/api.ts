import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Inventory } from './types';

export const inventoryApi = {
  getLowStock: async (params?: PaginationParams): Promise<ListResponse<Inventory>> => {
    const response = await apiClient.get('/admin/inventory/low-stock', {
      params,
    });
    return normalizeListResponse<Inventory>(
      response.data as Record<string, unknown>,
      'inventories'
    );
  },

  addStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/add`, { quantity, reason });
  },

  removeStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/remove`, { quantity, reason });
  },

  adjustStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/adjust`, {
      new_quantity: quantity,
      reason,
    });
  },
};
