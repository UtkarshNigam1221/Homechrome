import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Coupon, CreateCouponRequest, UpdateCouponRequest } from './types';

export const couponsApi = {
  list: async (
    params?: PaginationParams & {
      status?: string;
      type?: string;
      is_active?: boolean;
      search?: string;
    }
  ): Promise<ListResponse<Coupon>> => {
    const response = await apiClient.get(ROUTES.COUPONS.LIST, { params });
    return normalizeListResponse<Coupon>(response.data as Record<string, unknown>, 'coupons');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Coupon>(ROUTES.COUPONS.DETAIL(id));
    return response.data;
  },

  getByCode: async (code: string) => {
    const response = await apiClient.get<Coupon>(ROUTES.COUPONS.BY_CODE(code));
    return response.data;
  },

  create: async (data: CreateCouponRequest) => {
    const response = await apiClient.post<Coupon>(ROUTES.COUPONS.LIST, data);
    return response.data;
  },

  update: async (id: string, data: UpdateCouponRequest) => {
    const response = await apiClient.patch<Coupon>(ROUTES.COUPONS.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.COUPONS.DETAIL(id));
  },
};
