import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CalculatePriceRequest, CalculatePriceResponse, PricingRule } from './types';

export const pricingApi = {
  listRules: async (
    params?: PaginationParams & {
      scope_type?: string;
      category_id?: string;
      pricing_type?: string;
      is_active?: boolean;
    }
  ): Promise<ListResponse<PricingRule>> => {
    const response = await apiClient.get('/admin/pricing/rules', {
      params,
    });
    return normalizeListResponse<PricingRule>(response.data as Record<string, unknown>, 'rules');
  },

  getRule: async (id: string) => {
    const response = await apiClient.get<PricingRule>(`/admin/pricing/rules/${id}`);
    return response.data;
  },

  createRule: async (data: Partial<PricingRule>) => {
    const response = await apiClient.post<PricingRule>('/admin/pricing/rules', data);
    return response.data;
  },

  updateRule: async (id: string, data: Partial<PricingRule>) => {
    const response = await apiClient.patch<PricingRule>(`/admin/pricing/rules/${id}`, data);
    return response.data;
  },

  deleteRule: async (id: string) => {
    await apiClient.delete(`/admin/pricing/rules/${id}`);
  },

  getCategoryRules: async (categoryId: string) => {
    const response = await apiClient.get<PricingRule[]>(
      `/admin/pricing/rules/category/${categoryId}`
    );
    return response.data;
  },

  calculatePrice: async (data: CalculatePriceRequest): Promise<CalculatePriceResponse> => {
    const response = await apiClient.post<CalculatePriceResponse>(
      '/api/v1/pricing/calculate',
      data
    );
    return response.data;
  },

  getDimensionOptions: async (categoryId: string) => {
    const response = await apiClient.get(`/api/v1/pricing/dimension-options/${categoryId}`);
    return response.data;
  },
};
