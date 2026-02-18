import type { LoginRequest, User } from '@/types';
import apiClient from './client';

export const authApi = {
  login: async (credentials: LoginRequest): Promise<{ user: User }> => {
    const response = await apiClient.post<{ user: User }>('/admin/auth/login', credentials);
    return response.data;
  },

  logout: async (): Promise<void> => {
    await apiClient.post('/admin/auth/logout');
  },

  changePassword: async (currentPassword: string, newPassword: string): Promise<void> => {
    await apiClient.post('/admin/auth/password/change', {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },

  requestPasswordReset: async (email: string): Promise<void> => {
    await apiClient.post('/admin/auth/password/reset-request', { email });
  },

  resetPassword: async (token: string, newPassword: string): Promise<void> => {
    await apiClient.post('/admin/auth/password/reset', { token, new_password: newPassword });
  },

  getCurrentUser: async (): Promise<User> => {
    const response = await apiClient.get<User>('/admin/auth/me');
    return response.data;
  },
};
