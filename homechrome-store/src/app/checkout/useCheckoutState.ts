'use client';

import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useReducer, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { useAuthStore } from '@/stores/auth';
import {
  Address,
  CartWithItems,
  CheckoutResult,
  ServiceabilityResult,
} from '@/types';

import {
  checkoutReducer,
  initialCheckoutState,
} from './checkoutReducer';

export function useCheckoutState() {
  const router = useRouter();
  const customer = useAuthStore((s) => s.customer);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const [state, dispatch] = useReducer(checkoutReducer, initialCheckoutState);
  const [checkingServiceability, setCheckingServiceability] = useState(false);
  const [creatingAddress, setCreatingAddress] = useState(false);
  const [initiatingCheckout, setInitiatingCheckout] = useState(false);

  const addresses = useMemo(() => customer?.addresses || [], [customer?.addresses]);
  const selectedAddress = addresses.find((a) => a.id === state.selectedAddressId) || null;
  const selectedCourier = state.couriers.find((c) => c.id === state.selectedCourierId) || null;

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

  const checkServiceability = useCallback(async () => {
    if (!selectedAddress) return;
    setCheckingServiceability(true);
    dispatch({ type: 'SERVICEABILITY_START' });
    try {
      const { data } = await api.post<ServiceabilityResult>(
        ROUTES.CHECKOUT.SERVICEABILITY,
        { pincode: selectedAddress.postal_code },
      );
      if (!data.serviceable) {
        dispatch({
          type: 'SERVICEABILITY_FAIL',
          error: 'Delivery is not available to this PIN code. Please choose a different address.',
        });
        return;
      }
      dispatch({ type: 'SERVICEABILITY_SUCCESS', couriers: data.couriers });
    } catch {
      dispatch({
        type: 'SERVICEABILITY_FAIL',
        error: 'Failed to check delivery availability. Please try again.',
      });
    } finally {
      setCheckingServiceability(false);
    }
  }, [selectedAddress]);

  const handleAddressNext = useCallback(() => {
    if (!state.selectedAddressId) return;
    dispatch({ type: 'GO_TO_STEP', step: 'shipping' });
    checkServiceability();
  }, [state.selectedAddressId, checkServiceability]);

  const handleShippingNext = useCallback(() => {
    if (!state.selectedCourierId) return;
    dispatch({ type: 'GO_TO_STEP', step: 'review' });
  }, [state.selectedCourierId]);

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
    try {
      const payload: { shipping_address_id: string; courier_id?: number } = {
        shipping_address_id: state.selectedAddressId,
      };
      if (state.selectedCourierId) {
        payload.courier_id = state.selectedCourierId;
      }
      const { data } = await api.post<CheckoutResult>(ROUTES.CHECKOUT.INITIATE, payload);
      window.location.href = data.redirect_url;
    } catch {
      dispatch({
        type: 'PAYMENT_FAILED',
        error: 'Failed to initiate payment. Please try again.',
      });
    } finally {
      setInitiatingCheckout(false);
    }
  }, [state.selectedAddressId, state.selectedCourierId]);

  const goToStep = useCallback((step: 'address' | 'shipping' | 'review') => {
    if (step === 'shipping' && !state.selectedAddressId) return;
    if (step === 'review' && !state.selectedCourierId) return;
    dispatch({ type: 'GO_TO_STEP', step });
    if (step === 'shipping' && state.couriers.length === 0) {
      checkServiceability();
    }
  }, [state.selectedAddressId, state.selectedCourierId, state.couriers.length, checkServiceability]);

  return {
    state,
    dispatch,
    router,
    addresses,
    selectedAddress,
    selectedCourier,
    fetchCart,
    checkServiceability,
    handleAddressNext,
    handleShippingNext,
    handleAddAddress,
    handlePayNow,
    goToStep,
    checkingServiceability,
    creatingAddress,
    initiatingCheckout,
  };
}
