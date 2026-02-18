import type { Order } from '@/features/orders/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateCustomerRequest, Customer } from './types';

export const customersApi = {
  list: async (
    params?: PaginationParams & { status?: string; search?: string }
  ): Promise<ListResponse<Customer>> => {
    const response = await apiClient.get('/admin/customers', { params });
    return normalizeListResponse<Customer>(response.data as Record<string, unknown>, 'customers');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Customer>(`/admin/customers/${id}`);
    return response.data;
  },

  create: async (data: CreateCustomerRequest) => {
    const response = await apiClient.post<Customer>('/admin/customers', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCustomerRequest>) => {
    const response = await apiClient.put<Customer>(`/admin/customers/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/customers/${id}`);
  },

  getOrders: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Order>>(`/admin/customers/${id}/orders`, {
      params,
    });
    return response.data;
  },

  search: async (query: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Customer>>('/admin/customers/search', {
      params: { q: query, ...params },
    });
    return response.data;
  },
};
