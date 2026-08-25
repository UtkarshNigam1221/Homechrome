'use client';

import { useCallback, useState } from 'react';

import { previewTotal } from '@/lib/utils';

// Tracks a coupon against a subtotal and derives the preview total.
// Advisory only — checkout/initiate re-prices authoritatively.
export function useCoupon(subtotal: number) {
  const [code, setCode] = useState<string>();
  const [discount, setDiscount] = useState(0);

  const apply = useCallback((appliedCode: string, appliedDiscount: number) => {
    setCode(appliedCode);
    setDiscount(appliedDiscount);
  }, []);

  const remove = useCallback(() => {
    setCode(undefined);
    setDiscount(0);
  }, []);

  return { code, discount, total: previewTotal(subtotal, discount), apply, remove };
}
