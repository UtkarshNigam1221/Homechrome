import type { CreateShipmentResponse, ShipmentPriority } from '@/features/shipping';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateOrderRequest, Order, ProviderPaymentStatus } from './types';

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
    const response = await apiClient.get(ROUTES.ORDERS.LIST, { params });
    return normalizeListResponse<Order>(response.data as Record<string, unknown>, 'orders');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Order>(ROUTES.ORDERS.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateOrderRequest) => {
    const response = await apiClient.post<Order>(ROUTES.ORDERS.LIST, data);
    return response.data;
  },

  updateStatus: async (id: string, status: string) => {
    await apiClient.patch(ROUTES.ORDERS.STATUS(id), { status });
  },

  addNote: async (id: string, note: string, isInternal = true) => {
    await apiClient.post(ROUTES.ORDERS.NOTES(id), { note, is_internal: isInternal });
  },

  updateTracking: async (id: string, trackingNumber: string, carrier?: string) => {
    await apiClient.patch(ROUTES.ORDERS.TRACKING(id), {
      tracking_number: trackingNumber,
      carrier,
    });
  },

  cancel: async (id: string, reason?: string) => {
    await apiClient.post(ROUTES.ORDERS.CANCEL(id), { reason });
  },

  refund: async (id: string, amount: number, reason?: string) => {
    await apiClient.post(ROUTES.ORDERS.REFUND(id), { amount, reason });
  },

  checkPaymentStatus: async (id: string): Promise<ProviderPaymentStatus> => {
    const response = await apiClient.get<ProviderPaymentStatus>(ROUTES.ORDERS.PAYMENT_STATUS(id));
    return response.data;
  },

  createShipment: async (
    id: string,
    priority: ShipmentPriority = 'NORMAL'
  ): Promise<CreateShipmentResponse> => {
    const response = await apiClient.post<CreateShipmentResponse>(
      ROUTES.ORDERS.SHIPMENTS(id),
      null,
      { params: { priority } }
    );
    return response.data;
  },
};
