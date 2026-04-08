import type { CreateUserRequest, User } from '@/features/auth/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

export const usersApi = {
  list: async (
    params?: PaginationParams & { role?: string; status?: string; search?: string }
  ): Promise<ListResponse<User>> => {
    const response = await apiClient.get(ROUTES.SETTINGS.USERS.LIST, { params });
    return normalizeListResponse<User>(response.data as Record<string, unknown>, 'users');
  },

  get: async (id: string): Promise<User> => {
    const response = await apiClient.get<User>(ROUTES.SETTINGS.USERS.DETAIL(id));
    return response.data;
  },

  create: async (data: CreateUserRequest): Promise<User> => {
    const response = await apiClient.post<User>(ROUTES.SETTINGS.USERS.LIST, data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateUserRequest>): Promise<User> => {
    const response = await apiClient.patch<User>(ROUTES.SETTINGS.USERS.DETAIL(id), data);
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(ROUTES.SETTINGS.USERS.DETAIL(id));
  },

  updateStatus: async (id: string, status: string): Promise<void> => {
    await apiClient.patch(ROUTES.SETTINGS.USERS.STATUS(id), { status });
  },
};
