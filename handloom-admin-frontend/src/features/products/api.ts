import type { Inventory, InventoryTransaction } from '@/features/inventory/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { PaginationParams } from '@/shared/types/common';

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
    const response = await apiClient.get(ROUTES.PRODUCTS.LIST, {
      params: queryParams,
    });
    return normalizeListResponse<Product>(response.data as Record<string, unknown>, 'products');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Product>(ROUTES.PRODUCTS.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateProductRequest) => {
    const response = await apiClient.post<Product>(ROUTES.PRODUCTS.LIST, data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateProductRequest>) => {
    const response = await apiClient.patch<Product>(ROUTES.PRODUCTS.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.PRODUCTS.DETAIL(id));
  },

  getInventory: async (id: string) => {
    const response = await apiClient.get<Inventory>(ROUTES.PRODUCTS.INVENTORY(id));
    return response.data;
  },

  addStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(ROUTES.PRODUCTS.INVENTORY_ADD(id), {
      quantity,
      reason,
    });
    return response.data;
  },

  removeStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(ROUTES.PRODUCTS.INVENTORY_REMOVE(id), {
      quantity,
      reason,
    });
    return response.data;
  },

  adjustStock: async (id: string, newQuantity: number, reason?: string) => {
    const response = await apiClient.post(ROUTES.PRODUCTS.INVENTORY_ADJUST(id), {
      new_quantity: newQuantity,
      reason,
    });
    return response.data;
  },

  getInventoryTransactions: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get(ROUTES.PRODUCTS.INVENTORY_TRANSACTIONS(id), { params });
    // The endpoint returns `transactions`, not the `items` ListResponse assumes.
    return normalizeListResponse<InventoryTransaction>(
      response.data as Record<string, unknown>,
      'transactions'
    );
  },

  getFilterOptions: async (categoryId: string): Promise<Record<string, string[]>> => {
    const response = await apiClient.get<Record<string, string[]>>(
      ROUTES.PRODUCTS.FILTER_OPTIONS(categoryId)
    );
    return response.data;
  },

  reorder: async (categoryId: string, productIds: string[]) => {
    const response = await apiClient.put(ROUTES.PRODUCTS.REORDER(categoryId), {
      product_ids: productIds,
    });
    return response.data;
  },
};
