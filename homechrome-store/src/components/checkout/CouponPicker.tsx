'use client';

import { Button, Menu, Stack, Text } from '@mantine/core';
import { useCallback, useEffect, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatPrice } from '@/lib/utils';
import { CouponOffer } from '@/types';

interface CouponPickerProps {
  // Refetched whenever this moves: every saving below is a function of the subtotal.
  subtotal: number;
  onApply: (code: string, discount: number) => void;
}

export default function CouponPicker({ subtotal, onApply }: CouponPickerProps) {
  const [offers, setOffers] = useState<CouponOffer[]>([]);

  useEffect(() => {
    let current = true;
    api
      .get<CouponOffer[]>(ROUTES.CHECKOUT.COUPONS)
      .then(({ data }) => {
        if (current) setOffers(data || []);
      })
      .catch(() => {
        // A failed refetch must not leave a stale saving on screen against a cart
        // it no longer describes. The code input beside this still works.
        if (current) setOffers([]);
      });
    return () => {
      current = false;
    };
  }, [subtotal]);

  const apply = useCallback(
    (offer: CouponOffer) => onApply(offer.code, offer.discount_amount),
    [onApply],
  );

  if (offers.length === 0) return null;

  // A bare "(0)" reads as "nothing here" even though the dropdown still explains
  // why each code is out of reach, so the count only appears once it means "usable".
  const eligibleCount = offers.filter((o) => o.eligible).length;

  return (
    <Menu position="bottom-start" withinPortal>
      <Menu.Target>
        <Button variant="subtle" size="compact-sm">
          {eligibleCount > 0 ? `View available offers (${eligibleCount})` : 'View offers'}
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        {offers.map((offer) => (
          <Menu.Item
            key={offer.code}
            disabled={!offer.eligible}
            onClick={() => apply(offer)}
          >
            <Stack gap={2}>
              <Text fz="sm" fw={600}>
                {offer.code}
                {offer.eligible ? ` — saves ${formatPrice(offer.discount_amount)}` : ''}
              </Text>
              <Text fz="xs" c="dimmed">
                {offer.eligible ? offer.name : offer.reason}
              </Text>
            </Stack>
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}
