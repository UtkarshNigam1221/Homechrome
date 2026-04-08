import apiClient from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';

import type { LoginRequest, User } from './types';

export const authApi = {
  login: async (credentials: LoginRequest): Promise<{ user: User }> => {
    const response = await apiClient.post<{ user: User }>(ROUTES.AUTH.LOGIN, credentials);
    return response.data;
  },

  logout: async (): Promise<void> => {
    await apiClient.post(ROUTES.AUTH.LOGOUT);
  },

  changePassword: async (currentPassword: string, newPassword: string): Promise<void> => {
    await apiClient.post(ROUTES.AUTH.PASSWORD_CHANGE, {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },

  requestPasswordReset: async (email: string): Promise<void> => {
    await apiClient.post(ROUTES.AUTH.PASSWORD_RESET_REQUEST, { email });
  },

  resetPassword: async (token: string, newPassword: string): Promise<void> => {
    await apiClient.post(ROUTES.AUTH.PASSWORD_RESET, { token, new_password: newPassword });
  },

  getCurrentUser: async (): Promise<User> => {
    const response = await apiClient.get<User>(ROUTES.AUTH.ME);
    return response.data;
  },
};
