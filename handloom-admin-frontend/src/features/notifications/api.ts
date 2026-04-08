import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { Notification } from './types';

export const notificationsApi = {
  list: async (
    params?: PaginationParams & { user_id?: string; type?: string; status?: string }
  ): Promise<ListResponse<Notification>> => {
    const response = await apiClient.get(ROUTES.NOTIFICATIONS.LIST, {
      params,
    });
    return normalizeListResponse<Notification>(
      response.data as Record<string, unknown>,
      'notifications'
    );
  },

  getMy: async (params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Notification>>(ROUTES.NOTIFICATIONS.MY, {
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
    const response = await apiClient.post<Notification>(ROUTES.NOTIFICATIONS.LIST, data);
    return response.data;
  },

  markAsRead: async (id: string) => {
    await apiClient.post(ROUTES.NOTIFICATIONS.MARK_READ(id));
  },

  markAllAsRead: async () => {
    await apiClient.post(ROUTES.NOTIFICATIONS.MARK_ALL_READ);
  },
};
