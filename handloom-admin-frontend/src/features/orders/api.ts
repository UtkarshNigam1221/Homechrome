import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type {
  CreateOrderRequest,
  CreateRefundRequest,
  Order,
  PreviewRefundRequest,
  ProviderPaymentStatus,
  Refund,
  RefundPreview,
} from './types';

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

  updateTracking: async (
    id: string,
    trackingNumber: string,
    carrier?: string,
    trackingUrl?: string
  ) => {
    await apiClient.patch(ROUTES.ORDERS.TRACKING(id), {
      tracking_number: trackingNumber,
      carrier,
      tracking_url: trackingUrl,
    });
  },

  cancel: async (id: string, reason?: string) => {
    await apiClient.post(ROUTES.ORDERS.CANCEL(id), { reason });
  },

  listRefunds: async (id: string): Promise<Refund[]> => {
    const response = await apiClient.get(ROUTES.ORDERS.REFUNDS(id));
    return normalizeListResponse<Refund>(response.data as Record<string, unknown>, 'refunds').items;
  },

  // Prices a refund without raising one. The server derives the amount either way,
  // so previewing through it is what makes the figure on screen the one that leaves.
  previewRefund: async (id: string, data: PreviewRefundRequest): Promise<RefundPreview> => {
    const response = await apiClient.post<RefundPreview>(ROUTES.ORDERS.REFUND_PREVIEW(id), data);
    return response.data;
  },

  createRefund: async (id: string, data: CreateRefundRequest): Promise<Refund> => {
    const response = await apiClient.post<Refund>(ROUTES.ORDERS.REFUNDS(id), data);
    return response.data;
  },

  // Asks the provider directly. The escape hatch for a webhook that never came.
  recheckRefund: async (id: string, refundId: string): Promise<Refund> => {
    const response = await apiClient.post<Refund>(ROUTES.ORDERS.REFUND_RECHECK(id, refundId));
    return response.data;
  },

  checkPaymentStatus: async (id: string): Promise<ProviderPaymentStatus> => {
    const response = await apiClient.get<ProviderPaymentStatus>(ROUTES.ORDERS.PAYMENT_STATUS(id));
    return response.data;
  },
};
