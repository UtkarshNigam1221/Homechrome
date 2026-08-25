import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

import type { CreateUTMLinkRequest, UTMLink } from './types';

export const utmLinksApi = {
  list: async (params?: PaginationParams & { search?: string }): Promise<ListResponse<UTMLink>> => {
    const response = await apiClient.get(ROUTES.UTM_LINKS.LIST, { params });
    return normalizeListResponse<UTMLink>(response.data as Record<string, unknown>, 'links');
  },

  get: async (id: string) => {
    const response = await apiClient.get<UTMLink>(ROUTES.UTM_LINKS.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateUTMLinkRequest) => {
    const response = await apiClient.post<UTMLink>(ROUTES.UTM_LINKS.LIST, data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateUTMLinkRequest>) => {
    const response = await apiClient.patch<UTMLink>(ROUTES.UTM_LINKS.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(ROUTES.UTM_LINKS.DETAIL(id));
  },
};
