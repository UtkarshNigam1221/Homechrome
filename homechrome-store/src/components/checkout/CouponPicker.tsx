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
        // Not being able to list is not the same as there being none. The code input
        // beside this still works, and Validate is the authority.
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

  return (
    <Menu position="bottom-start" withinPortal>
      <Menu.Target>
        <Button variant="subtle" size="compact-sm">
          View available offers ({offers.filter((o) => o.eligible).length})
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
