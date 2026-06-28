import { create } from 'zustand';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
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
    await api.post(ROUTES.AUTH.SEND_OTP, { phone });
  },

  verifyOTP: async (phone: string, code: string) => {
    const { data } = await api.post<{
      customer: Customer;
      is_new_customer: boolean;
    }>(ROUTES.AUTH.VERIFY_OTP, { phone, code });
    set({ customer: data.customer, isAuthenticated: true });
    const secure = window.location.protocol === 'https:' ? '; secure' : '';
    document.cookie = `hc_session=1; path=/; max-age=604800; samesite=lax${secure}`;
    return data;
  },

  logout: async () => {
    try {
      await api.post(ROUTES.AUTH.LOGOUT);
    } finally {
      set({ customer: null, isAuthenticated: false });
      document.cookie = 'hc_session=; path=/; max-age=0';
    }
  },

  checkAuth: async () => {
    try {
      const { data } = await api.get<Customer>(ROUTES.ME.PROFILE);
      set({ customer: data, isAuthenticated: true, isLoading: false });
    } catch {
      set({ customer: null, isAuthenticated: false, isLoading: false });
      document.cookie = 'hc_session=; path=/; max-age=0';
    }
  },
}));
