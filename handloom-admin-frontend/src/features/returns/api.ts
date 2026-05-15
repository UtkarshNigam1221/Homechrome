import apiClient from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';

import type { CreateReturnRequest, ReturnRequest } from './types';

export const returnsApi = {
  createReturn: async (orderId: string, body: CreateReturnRequest): Promise<ReturnRequest> => {
    const response = await apiClient.post<ReturnRequest>(ROUTES.ORDERS.RETURNS(orderId), body);
    return response.data;
  },

  listForOrder: async (orderId: string): Promise<ReturnRequest[]> => {
    const response = await apiClient.get<ReturnRequest[]>(ROUTES.ORDERS.RETURNS(orderId));
    return response.data ?? [];
  },

  cancel: async (returnId: string): Promise<{ status: string }> => {
    const response = await apiClient.patch<{ status: string }>(ROUTES.RETURNS.CANCEL(returnId));
    return response.data;
  },

  processRefund: async (returnId: string, amountPaise: number): Promise<{ status: string }> => {
    const response = await apiClient.post<{ status: string }>(ROUTES.RETURNS.REFUND(returnId), {
      amount_paise: amountPaise,
    });
    return response.data;
  },
};
