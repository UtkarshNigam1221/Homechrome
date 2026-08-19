import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Inventory, ReconciliationReport } from './types';

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

  // Stock held against orders that never shipped and never cancelled. minAge
  // keeps checkouts merely still in flight out of the report.
  getReconciliation: async (params?: { min_age?: string; limit?: number }) => {
    const response = await apiClient.get<ReconciliationReport>(ROUTES.INVENTORY.RECONCILIATION, {
      params,
    });
    return response.data;
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
