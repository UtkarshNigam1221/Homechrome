import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Notification } from './types';

export const notificationsApi = {
  list: async (
    params?: PaginationParams & { user_id?: string; type?: string; status?: string }
  ): Promise<ListResponse<Notification>> => {
    const response = await apiClient.get('/admin/notifications', {
      params,
    });
    return normalizeListResponse<Notification>(
      response.data as Record<string, unknown>,
      'notifications'
    );
  },

  getMy: async (params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Notification>>('/admin/notifications/my', {
      params,
    });
    return response.data;
  },

  send: async (data: {
    user_id: string;
    type: string;
    title: string;
    message: string;
    data?: Record<string, unknown>;
    priority?: string;
  }) => {
    const response = await apiClient.post<Notification>('/admin/notifications', data);
    return response.data;
  },

  markAsRead: async (id: string) => {
    await apiClient.post(`/admin/notifications/${id}/read`);
  },

  markAllAsRead: async () => {
    await apiClient.post('/admin/notifications/read-all');
  },
};
