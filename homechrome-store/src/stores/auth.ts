'use client';

import { isAxiosError } from 'axios';
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
    } catch (e) {
      set({ isLoading: false });

      // Only a 401 means the session is actually gone. Clearing on any error
      // is worse than it looks: dropping hc_session stops AuthInit from ever
      // calling checkAuth again (providers.tsx gates on that cookie), so one
      // network blip or 5xx leaves the header logged out for good while
      // store_token/store_refresh are still valid and middleware.ts still
      // admits the customer to /account.
      if (isAxiosError(e) && e.response?.status === 401) {
        set({ customer: null, isAuthenticated: false });
        document.cookie = 'hc_session=; path=/; max-age=0';
      }
    }
  },
}));
