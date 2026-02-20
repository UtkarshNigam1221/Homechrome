import { create } from 'zustand';

import api from '@/lib/api';
import { Customer } from '@/types';

interface AuthState {
  customer: Customer | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setCustomer: (customer: Customer | null) => void;
  sendOTP: (phone: string) => Promise<void>;
  verifyOTP: (
    phone: string,
    code: string,
  ) => Promise<{ customer: Customer; is_new_customer: boolean }>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  customer: null,
  isAuthenticated: false,
  isLoading: true,

  setCustomer: (customer) => set({ customer, isAuthenticated: !!customer }),

  sendOTP: async (phone: string) => {
    await api.post('/api/v1/store/auth/otp/send', { phone });
  },

  verifyOTP: async (phone: string, code: string) => {
    const { data } = await api.post<{
      customer: Customer;
      is_new_customer: boolean;
    }>('/api/v1/store/auth/otp/verify', { phone, code });
    set({ customer: data.customer, isAuthenticated: true });
    return data;
  },

  logout: async () => {
    try {
      await api.post('/api/v1/store/auth/logout');
    } finally {
      set({ customer: null, isAuthenticated: false });
    }
  },

  checkAuth: async () => {
    try {
      const { data } = await api.get<Customer>('/api/v1/store/me');
      set({ customer: data, isAuthenticated: true, isLoading: false });
    } catch {
      set({ customer: null, isAuthenticated: false, isLoading: false });
    }
  },
}));
