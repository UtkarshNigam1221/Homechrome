import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Coupon, CreateCouponRequest } from './types';

export const couponsApi = {
  list: async (
    params?: PaginationParams & {
      status?: string;
      type?: string;
      is_active?: boolean;
      search?: string;
    }
  ): Promise<ListResponse<Coupon>> => {
    const response = await apiClient.get('/admin/coupons', { params });
    return normalizeListResponse<Coupon>(response.data as Record<string, unknown>, 'coupons');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Coupon>(`/admin/coupons/${id}`);
    return response.data;
  },

  getByCode: async (code: string) => {
    const response = await apiClient.get<Coupon>(`/admin/coupons/code/${code}`);
    return response.data;
  },

  create: async (data: CreateCouponRequest) => {
    const response = await apiClient.post<Coupon>('/admin/coupons', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCouponRequest>) => {
    const response = await apiClient.patch<Coupon>(`/admin/coupons/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/coupons/${id}`);
  },

  validate: async (
    code: string,
    orderTotal: number,
    customerId?: string,
    productIds?: string[]
  ) => {
    const response = await apiClient.post('/admin/coupons/validate', {
      code,
      order_total: orderTotal,
      customer_id: customerId,
      product_ids: productIds,
    });
    return response.data;
  },
};
