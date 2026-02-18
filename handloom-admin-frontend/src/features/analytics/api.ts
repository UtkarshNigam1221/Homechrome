import apiClient from '@/shared/api/client';

import type { DashboardStats, SalesAnalytics, TopCategory, TopProduct } from './types';

export const analyticsApi = {
  getDashboard: async (): Promise<DashboardStats> => {
    const response = await apiClient.get<DashboardStats>('/admin/analytics/dashboard');
    return response.data;
  },

  getSales: async (params?: {
    period?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<SalesAnalytics> => {
    const response = await apiClient.get<SalesAnalytics>('/admin/analytics/sales', { params });
    return response.data;
  },

  getTopProducts: async (params?: {
    limit?: number;
    start_date?: string;
    end_date?: string;
  }): Promise<TopProduct[]> => {
    const response = await apiClient.get('/admin/analytics/top-products', { params });
    const data = response.data;
    if (Array.isArray(data)) return data;
    const obj = data as Record<string, unknown>;
    return (obj.items || obj.products || []) as TopProduct[];
  },

  getTopCategories: async (params?: {
    limit?: number;
    start_date?: string;
    end_date?: string;
  }): Promise<TopCategory[]> => {
    const response = await apiClient.get('/admin/analytics/top-categories', { params });
    const data = response.data;
    if (Array.isArray(data)) return data;
    const obj = data as Record<string, unknown>;
    return (obj.items || obj.categories || []) as TopCategory[];
  },

  getCustomerAnalytics: async (params?: { start_date?: string; end_date?: string }) => {
    const response = await apiClient.get('/admin/analytics/customers', { params });
    return response.data;
  },

  getInventoryAnalytics: async () => {
    const response = await apiClient.get('/admin/analytics/inventory');
    return response.data;
  },
};
