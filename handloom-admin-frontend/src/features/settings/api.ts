import type { CreateUserRequest, User } from '@/features/auth/types';
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';

export const usersApi = {
  list: async (
    params?: PaginationParams & { role?: string; status?: string; search?: string }
  ): Promise<ListResponse<User>> => {
    const response = await apiClient.get('/admin/users', { params });
    return normalizeListResponse<User>(response.data as Record<string, unknown>, 'users');
  },

  get: async (id: string): Promise<User> => {
    const response = await apiClient.get<User>(`/admin/users/${id}`);
    return response.data;
  },

  create: async (data: CreateUserRequest): Promise<User> => {
    const response = await apiClient.post<User>('/admin/users', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateUserRequest>): Promise<User> => {
    const response = await apiClient.patch<User>(`/admin/users/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/admin/users/${id}`);
  },

  updateStatus: async (id: string, status: string): Promise<void> => {
    await apiClient.patch(`/admin/users/${id}/status`, { status });
  },
};
