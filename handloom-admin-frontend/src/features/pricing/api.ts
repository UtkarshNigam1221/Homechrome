import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CalculatePriceRequest, CalculatePriceResponse, PricingRule } from './types';

export const pricingApi = {
  listRules: async (
    params?: PaginationParams & {
      scope_type?: string;
      category_id?: string;
      pricing_type?: string;
      is_active?: boolean;
      search?: string;
    }
  ): Promise<ListResponse<PricingRule>> => {
    const response = await apiClient.get(ROUTES.PRICING.RULES, {
      params,
    });
    return normalizeListResponse<PricingRule>(response.data as Record<string, unknown>, 'rules');
  },

  getRule: async (id: string) => {
    const response = await apiClient.get<PricingRule>(ROUTES.PRICING.RULE_DETAIL(id));
    return response.data;
  },

  createRule: async (data: Partial<PricingRule>) => {
    const response = await apiClient.post<PricingRule>(ROUTES.PRICING.RULES, data);
    return response.data;
  },

  updateRule: async (id: string, data: Partial<PricingRule>) => {
    const response = await apiClient.patch<PricingRule>(ROUTES.PRICING.RULE_DETAIL(id), data);
    return response.data;
  },

  deleteRule: async (id: string) => {
    await apiClient.delete(ROUTES.PRICING.RULE_DETAIL(id));
  },

  getCategoryRules: async (categoryId: string) => {
    const response = await apiClient.get<PricingRule[]>(
      ROUTES.PRICING.RULES_BY_CATEGORY(categoryId)
    );
    return response.data;
  },

  calculatePrice: async (data: CalculatePriceRequest): Promise<CalculatePriceResponse> => {
    const response = await apiClient.post<CalculatePriceResponse>(ROUTES.PRICING.CALCULATE, data);
    return response.data;
  },

  getDimensionOptions: async (categoryId: string) => {
    const response = await apiClient.get(ROUTES.PRICING.DIMENSION_OPTIONS(categoryId));
    return response.data;
  },
};
