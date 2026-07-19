'use client';

import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useReducer, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { useAuthStore } from '@/stores/auth';
import { Address, CartWithItems, CheckoutResult } from '@/types';

import {
  checkoutReducer,
  initialCheckoutState,
} from './checkoutReducer';

export function useCheckoutState() {
  const router = useRouter();
  const customer = useAuthStore((s) => s.customer);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const [state, dispatch] = useReducer(checkoutReducer, initialCheckoutState);
  const [creatingAddress, setCreatingAddress] = useState(false);
  const [initiatingCheckout, setInitiatingCheckout] = useState(false);

  const addresses = useMemo(() => customer?.addresses || [], [customer?.addresses]);
  const selectedAddress = addresses.find((a) => a.id === state.selectedAddressId) || null;

  const fetchCart = useCallback(async () => {
    dispatch({ type: 'CART_LOADING' });
    try {
      const { data } = await api.get<CartWithItems>(ROUTES.CART.ROOT);
      dispatch({ type: 'CART_LOADED', cart: data });
      if (!data.items || data.items.length === 0) {
        router.replace('/cart');
      }
    } catch {
      router.replace('/cart');
    }
  }, [router]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchCart();
    }
  }, [isAuthenticated, fetchCart]);

  useEffect(() => {
    if (addresses.length > 0 && !state.selectedAddressId) {
      const defaultAddr = addresses.find((a) => a.is_default);
      dispatch({
        type: 'SELECT_ADDRESS',
        id: defaultAddr?.id || addresses[0].id,
      });
    }
  }, [addresses, state.selectedAddressId]);

  const handleAddressNext = useCallback(() => {
    if (!state.selectedAddressId) return;
    dispatch({ type: 'GO_TO_STEP', step: 'review' });
  }, [state.selectedAddressId]);

  const handleAddAddress = useCallback(async (data: Omit<Address, 'id'>) => {
    setCreatingAddress(true);
    dispatch({ type: 'ADDRESS_SAVE_START' });
    try {
      const { data: newAddr } = await api.post<Address>(ROUTES.ME.ADDRESSES, data);
      await useAuthStore.getState().checkAuth();
      dispatch({ type: 'ADDRESS_SAVED', addressId: newAddr.id });
    } catch {
      dispatch({
        type: 'ADDRESS_SAVE_FAILED',
        error: 'Failed to save address. Please try again.',
      });
    } finally {
      setCreatingAddress(false);
    }
  }, []);

  const handlePayNow = useCallback(async () => {
    if (!state.selectedAddressId) return;
    setInitiatingCheckout(true);
    dispatch({ type: 'PAYMENT_START' });
    // Keep the overlay visible across the redirect handoff. window.location.href
    // is async — clearing the flag in finally causes the overlay to flash off
    // a few hundred ms before the new page actually unloads.
    let redirecting = false;
    try {
      const { data } = await api.post<CheckoutResult>(ROUTES.CHECKOUT.INITIATE, {
        shipping_address_id: state.selectedAddressId,
      });
      redirecting = true;
      window.location.href = data.redirect_url;
    } catch {
      dispatch({
        type: 'PAYMENT_FAILED',
        error: 'Failed to initiate payment. Please try again.',
      });
    } finally {
      if (!redirecting) setInitiatingCheckout(false);
    }
  }, [state.selectedAddressId]);

  const goToStep = useCallback((step: 'address' | 'review') => {
    if (step === 'review' && !state.selectedAddressId) return;
    dispatch({ type: 'GO_TO_STEP', step });
  }, [state.selectedAddressId]);

  return {
    state,
    dispatch,
    router,
    addresses,
    selectedAddress,
    fetchCart,
    handleAddressNext,
    handleAddAddress,
    handlePayNow,
    goToStep,
    creatingAddress,
    initiatingCheckout,
  };
}
