'use client';

import { useCallback, useState } from 'react';

import { previewTotal } from '@/lib/utils';

// Read once by useCheckoutState on mount, to carry a cart-applied coupon
// into checkout. The cart has no order yet, so this is one global slot.
export const CART_COUPON_STORAGE_KEY = 'cart_coupon';

// Tracks a coupon against a subtotal and derives the preview total.
// Advisory only — checkout/initiate re-prices authoritatively.
export function useCoupon(subtotal: number) {
  const [code, setCode] = useState<string>();
  const [discount, setDiscount] = useState(0);

  const apply = useCallback((appliedCode: string, appliedDiscount: number) => {
    setCode(appliedCode);
    setDiscount(appliedDiscount);
    sessionStorage.setItem(
      CART_COUPON_STORAGE_KEY,
      JSON.stringify({ code: appliedCode, discount: appliedDiscount }),
    );
  }, []);

  const remove = useCallback(() => {
    setCode(undefined);
    setDiscount(0);
    sessionStorage.removeItem(CART_COUPON_STORAGE_KEY);
  }, []);

  return { code, discount, total: previewTotal(subtotal, discount), apply, remove };
}
