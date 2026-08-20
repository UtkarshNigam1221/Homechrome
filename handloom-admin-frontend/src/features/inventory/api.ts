import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Inventory } from './types';

export const inventoryApi = {
  getLowStock: async (params?: PaginationParams): Promise<ListResponse<Inventory>> => {
    const response = await apiClient.get(ROUTES.INVENTORY.LOW_STOCK, {
      params,
    });
    return normalizeListResponse<Inventory>(
      response.data as Record<string, unknown>,
      'inventories'
    );
  },

  addStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(ROUTES.PRODUCTS.INVENTORY_ADD(productId), { quantity, reason });
  },

  removeStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(ROUTES.PRODUCTS.INVENTORY_REMOVE(productId), { quantity, reason });
  },

  adjustStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(ROUTES.PRODUCTS.INVENTORY_ADJUST(productId), {
      new_quantity: quantity,
      reason,
    });
  },
};
