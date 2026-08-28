'use client';

import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useReducer, useState } from 'react';

import { CART_COUPON_STORAGE_KEY } from '@/hooks/useCoupon';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { previewTotal } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { Address, CartWithItems, CheckoutResult, CouponValidationResult } from '@/types';

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

  // Preview only — checkout/initiate re-prices authoritatively below.
  const orderTotal = useMemo(
    () => (state.cart ? previewTotal(state.cart.cart.subtotal, state.couponDiscount) : 0),
    [state.cart, state.couponDiscount],
  );

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

  // Carries a cart-applied coupon into checkout. Re-validates rather than
  // trusting the cart-time figure, which may be stale by now.
  useEffect(() => {
    if (!isAuthenticated) return;
    const stashed = sessionStorage.getItem(CART_COUPON_STORAGE_KEY);
    if (!stashed) return;
    sessionStorage.removeItem(CART_COUPON_STORAGE_KEY);

    let code: string | undefined;
    try {
      ({ code } = JSON.parse(stashed) as { code?: string });
    } catch {
      return;
    }
    if (!code) return;

    api
      .post<CouponValidationResult>(ROUTES.CHECKOUT.VALIDATE_COUPON, { code })
      .then(({ data }) => {
        if (data.valid) {
          dispatch({ type: 'COUPON_APPLIED', code: data.code, discount: data.discount_amount ?? 0 });
        }
      })
      .catch(() => {
        // Best-effort — the customer can re-enter it in the review step.
      });
  }, [isAuthenticated]);

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
        coupon_code: state.couponCode ?? undefined,
      });
      // The browser is about to leave for the payment gateway and lose this
      // response — stash the notice so the confirmation page can show it.
      if (data.coupon_notice) {
        sessionStorage.setItem(`coupon_notice:${data.order.id}`, data.coupon_notice);
      }
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
  }, [state.selectedAddressId, state.couponCode]);

  const handleCouponApplied = useCallback((code: string, discount: number) => {
    dispatch({ type: 'COUPON_APPLIED', code, discount });
  }, []);

  const handleCouponRemoved = useCallback(() => {
    dispatch({ type: 'COUPON_REMOVED' });
  }, []);

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
    orderTotal,
    fetchCart,
    handleAddressNext,
    handleAddAddress,
    handlePayNow,
    handleCouponApplied,
    handleCouponRemoved,
    goToStep,
    creatingAddress,
    initiatingCheckout,
  };
}
