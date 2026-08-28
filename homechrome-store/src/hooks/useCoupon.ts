'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { previewTotal } from '@/lib/utils';
import { CouponValidationResult } from '@/types';

// Read by useCheckoutState on mount, to carry a cart-applied coupon
// into checkout. The cart has no order yet, so this is one global slot.
export const CART_COUPON_STORAGE_KEY = 'cart_coupon';

// The stash holds a discount too, but only the code is read back: the figure it was
// worth is a function of the cart, which has probably moved since.
function stashedCode(): string | undefined {
  const raw = sessionStorage.getItem(CART_COUPON_STORAGE_KEY);
  if (!raw) return undefined;
  try {
    return (JSON.parse(raw) as { code?: string }).code || undefined;
  } catch {
    return undefined;
  }
}

// Tracks a coupon against a subtotal and derives the preview total.
// Advisory only — checkout/initiate re-prices authoritatively.
//
// The discount is re-validated whenever the subtotal moves, because the subtotal is its
// only input: applying 20% to a ₹3,000 cart and then deleting items down to ₹1,000 used
// to leave ₹600 on screen, showing a total of ₹400 against the ₹800 that would be
// charged. The server is asked rather than the percentage re-derived — the client never
// computes a discount. The same effect recovers a code left in sessionStorage, so a page
// refresh can no longer drop the chip while the stash quietly survives to checkout.
export function useCoupon(subtotal: number, isAuthenticated: boolean) {
  const [code, setCode] = useState<string>();
  const [discount, setDiscount] = useState(0);
  // Held in a ref as well as state so the effect below can read the current code without
  // depending on it — applying one already has the server's answer for this subtotal,
  // and re-running on it would just repeat the request.
  const codeRef = useRef<string | undefined>(undefined);

  const apply = useCallback((appliedCode: string, appliedDiscount: number) => {
    codeRef.current = appliedCode;
    setCode(appliedCode);
    setDiscount(appliedDiscount);
    sessionStorage.setItem(
      CART_COUPON_STORAGE_KEY,
      JSON.stringify({ code: appliedCode, discount: appliedDiscount }),
    );
  }, []);

  const remove = useCallback(() => {
    codeRef.current = undefined;
    setCode(undefined);
    setDiscount(0);
    sessionStorage.removeItem(CART_COUPON_STORAGE_KEY);
  }, []);

  useEffect(() => {
    // The cart is guest-capable (OptionalCartAuth) but validate-coupon is not
    // (CustomerAuth.Authenticate), so asking as a guest is a 401 that api.ts turns into
    // a redirect to /login — /cart is not on its exemption list. Same guard
    // useCheckoutState uses. Re-runs on sign-in, which restores a stashed code.
    if (!isAuthenticated) return;

    const held = codeRef.current ?? stashedCode();
    if (!held) return;

    // Guards against a slower earlier request landing after a newer one: the cleanup
    // runs before the next effect, so only the latest attempt can write.
    let current = true;
    api
      .post<CouponValidationResult>(ROUTES.CHECKOUT.VALIDATE_COUPON, { code: held })
      .then(({ data }) => {
        if (!current) return;
        if (data.valid) {
          apply(data.code, data.discount_amount ?? 0);
        } else {
          // The server answered and said no — a minimum order value the cart has
          // dropped under, say. Showing nothing beats showing a discount that will
          // not apply.
          remove();
        }
      })
      .catch(() => {
        // Not being able to ask is not the same as being told no. A timeout or a 5xx
        // leaves the coupon and the stash alone; the next subtotal change retries.
      });

    return () => {
      current = false;
    };
  }, [subtotal, isAuthenticated, apply, remove]);

  return { code, discount, total: previewTotal(subtotal, discount), apply, remove };
}
