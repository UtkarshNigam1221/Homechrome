import apiClient from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { PaginationParams } from '@/shared/types/common';

import type {
  BroadcastRequest,
  BroadcastResult,
  PushBroadcast,
  PushSubscriber,
  PushSubscriptionStatus,
} from './types';

export interface SubscriberPage {
  subscriptions: PushSubscriber[];
  pagination?: { has_more: boolean; next_cursor?: string };
}

export const pushApi = {
  listSubscribers: async (
    params?: PaginationParams & { status?: PushSubscriptionStatus }
  ): Promise<SubscriberPage> => {
    const response = await apiClient.get<SubscriberPage>(ROUTES.PUSH.SUBSCRIBERS, { params });
    return response.data;
  },

  listBroadcasts: async (limit = 20): Promise<{ broadcasts: PushBroadcast[] }> => {
    const response = await apiClient.get<{ broadcasts: PushBroadcast[] }>(ROUTES.PUSH.BROADCASTS, {
      params: { limit },
    });
    return response.data;
  },

  broadcast: async (data: BroadcastRequest): Promise<BroadcastResult> => {
    const response = await apiClient.post<BroadcastResult>(ROUTES.PUSH.BROADCAST, data);
    return response.data;
  },
};
