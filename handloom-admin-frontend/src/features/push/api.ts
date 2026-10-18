import apiClient from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { PaginationParams, PaginationResponse } from '@/shared/types/common';

import type {
  BroadcastRequest,
  PushBroadcast,
  PushSubscriber,
  PushSubscriptionStatus,
} from './types';

export interface SubscriberPage {
  subscriptions: PushSubscriber[];
  pagination?: PaginationResponse;
}

export const pushApi = {
  listSubscribers: async (
    params?: PaginationParams & { status?: PushSubscriptionStatus }
  ): Promise<SubscriberPage> => {
    const response = await apiClient.get<SubscriberPage>(ROUTES.PUSH.SUBSCRIBERS, { params });
    return response.data;
  },

  listBroadcasts: async (): Promise<{ broadcasts: PushBroadcast[] }> => {
    const response = await apiClient.get<{ broadcasts: PushBroadcast[] }>(ROUTES.PUSH.BROADCASTS);
    return response.data;
  },

  broadcast: async (data: BroadcastRequest): Promise<PushBroadcast> => {
    const response = await apiClient.post<PushBroadcast>(ROUTES.PUSH.BROADCAST, data);
    return response.data;
  },
};
