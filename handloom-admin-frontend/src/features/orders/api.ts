import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateOrderRequest, Order } from './types';

export const ordersApi = {
  list: async (
    params?: PaginationParams & {
      status?: string;
      payment_status?: string;
      customer_id?: string;
      start_date?: string;
      end_date?: string;
      search?: string;
    }
  ): Promise<ListResponse<Order>> => {
    const response = await apiClient.get('/admin/orders', { params });
    return normalizeListResponse<Order>(response.data as Record<string, unknown>, 'orders');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Order>(`/admin/orders/${id}`);
    return response.data;
  },

  create: async (data: CreateOrderRequest) => {
    const response = await apiClient.post<Order>('/admin/orders', data);
    return response.data;
  },

  updateStatus: async (id: string, status: string) => {
    await apiClient.patch(`/admin/orders/${id}/status`, { status });
  },

  addNote: async (id: string, note: string, isInternal = true) => {
    await apiClient.post(`/admin/orders/${id}/notes`, { note, is_internal: isInternal });
  },

  updateTracking: async (id: string, trackingNumber: string, carrier?: string) => {
    await apiClient.patch(`/admin/orders/${id}/tracking`, {
      tracking_number: trackingNumber,
      carrier,
    });
  },

  cancel: async (id: string, reason?: string) => {
    await apiClient.post(`/admin/orders/${id}/cancel`, { reason });
  },

  refund: async (id: string, amount: number, reason?: string) => {
    await apiClient.post(`/admin/orders/${id}/refund`, { amount, reason });
  },
};
