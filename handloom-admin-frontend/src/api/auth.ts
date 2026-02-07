import type { LoginRequest, LoginResponse, User } from '../types';
import apiClient from './client';

export const authApi = {
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response = await apiClient.post<LoginResponse>('/admin/auth/login', credentials);
    return response.data;
  },

  logout: async (): Promise<void> => {
    await apiClient.post('/admin/auth/logout');
  },

  refreshToken: async (
    refreshToken: string
  ): Promise<{ access_token: string; refresh_token: string }> => {
    const response = await apiClient.post('/admin/auth/refresh', { refresh_token: refreshToken });
    return response.data.tokens || response.data;
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
    const response = await apiClient.get<{ data: User }>('/admin/users/me');
    return response.data.data;
  },
};
