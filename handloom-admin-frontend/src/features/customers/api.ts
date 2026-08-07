import type { Order } from '@/features/orders/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { Address, ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateCustomerRequest, Customer } from './types';

export const customersApi = {
  list: async (
    params?: PaginationParams & { status?: string; search?: string }
  ): Promise<ListResponse<Customer>> => {
    const response = await apiClient.get(ROUTES.CUSTOMERS.LIST, { params });
    return normalizeListResponse<Customer>(response.data as Record<string, unknown>, 'customers');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Customer>(ROUTES.CUSTOMERS.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateCustomerRequest) => {
    const response = await apiClient.post<Customer>(ROUTES.CUSTOMERS.LIST, data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCustomerRequest>) => {
    const response = await apiClient.put<Customer>(ROUTES.CUSTOMERS.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.CUSTOMERS.DETAIL(id));
  },

  getOrders: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Order>>(ROUTES.CUSTOMERS.ORDERS(id), {
      params,
    });
    return response.data;
  },

  // The three address endpoints each return the full updated customer.
  addAddress: async (id: string, address: Address) => {
    const response = await apiClient.post<Customer>(ROUTES.CUSTOMERS.ADDRESSES(id), address);
    return response.data;
  },

  updateAddress: async (id: string, addressId: string, address: Address) => {
    const response = await apiClient.put<Customer>(
      ROUTES.CUSTOMERS.ADDRESS(id, addressId),
      address
    );
    return response.data;
  },

  removeAddress: async (id: string, addressId: string) => {
    const response = await apiClient.delete<Customer>(ROUTES.CUSTOMERS.ADDRESS(id, addressId));
    return response.data;
  },

  search: async (query: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Customer>>(ROUTES.CUSTOMERS.SEARCH, {
      params: { q: query, ...params },
    });
    return response.data;
  },
};
