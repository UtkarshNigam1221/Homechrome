'use client';

import { Collapse, Group, Stack, Text, UnstyledButton } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useCallback, useEffect, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatPrice } from '@/lib/utils';
import { CouponOffer } from '@/types';

interface CouponPickerProps {
  // Refetched whenever this moves: every saving below is a function of the subtotal.
  subtotal: number;
  // The one already on the order, so the list can mark it instead of offering it again.
  appliedCode?: string;
  onApply: (code: string, discount: number) => void;
}

// Lining figures in a fixed column so the amounts stack and scan.
const AMOUNT_STYLE = {
  fontVariantNumeric: 'tabular-nums',
  fontWeight: 700,
  minWidth: '4.5rem',
  textAlign: 'right',
} as const;

export default function CouponPicker({ subtotal, appliedCode, onApply }: CouponPickerProps) {
  const [offers, setOffers] = useState<CouponOffer[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, { toggle }] = useDisclosure(true);

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
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [subtotal]);

  const apply = useCallback(
    (offer: CouponOffer) => onApply(offer.code, offer.discount_amount),
    [onApply],
  );

  if (loading) {
    return (
      <Text fz="xs" c="dimmed">
        Checking which offers fit this order…
      </Text>
    );
  }

  if (offers.length === 0) return null;

  // Only counts what you could switch to: not the applied one, not the out-of-reach ones.
  const usable = offers.filter((o) => o.eligible && o.code !== appliedCode).length;

  return (
    <Stack gap={0}>
      <UnstyledButton
        onClick={toggle}
        aria-expanded={open}
        style={{ borderRadius: 'var(--mantine-radius-sm)' }}
      >
        <Group gap="xs" wrap="nowrap" py={4}>
          {/* One name in both states; the chevron and aria-expanded carry it. */}
          <Text fz="sm" fw={500} c="navy.7">
            Offers
          </Text>
          {usable > 0 && (
            <Text
              component="span"
              fz={10}
              fw={700}
              c="navy.7"
              bg="brand.2"
              px={6}
              py={1}
              style={{ borderRadius: 2, letterSpacing: '0.06em' }}
            >
              {usable} USABLE
            </Text>
          )}
          <Text
            component="span"
            aria-hidden
            fz={10}
            c="brand.7"
            style={{
              rotate: open ? '180deg' : '0deg',
              transition: 'rotate 160ms ease',
              lineHeight: 1,
            }}
          >
            ▼
          </Text>
        </Group>
      </UnstyledButton>

      <Collapse expanded={open}>
        <Stack gap={0} pt={4}>
          {offers.map((offer, i) => {
            const rule = i > 0 ? '1px solid var(--mantine-color-brand-2)' : undefined;
            const isApplied = offer.code === appliedCode;
            const selectable = offer.eligible && !isApplied;

            // aria-disabled, not disabled: the reason is the most useful thing on the
            // row and `disabled` would drop it out of the tab order entirely.
            return (
              <UnstyledButton
                key={offer.code}
                onClick={selectable ? () => apply(offer) : undefined}
                aria-disabled={!selectable}
                aria-label={
                  isApplied
                    ? `${offer.code} is already applied`
                    : offer.eligible
                      ? `Apply ${offer.code}, saves ${formatPrice(offer.discount_amount)}`
                      : `${offer.code} unavailable. ${offer.reason ?? ''}`
                }
                data-usable={selectable || undefined}
                style={{
                  borderTop: rule,
                  cursor: selectable ? 'pointer' : 'default',
                  borderRadius: 'var(--mantine-radius-xs)',
                }}
              >
                <Group gap="sm" wrap="nowrap" align="baseline" py={7} px={2}>
                  <Stack gap={1} style={{ flex: 1, minWidth: 0 }}>
                    <Text
                      fz="xs"
                      fw={offer.eligible ? 700 : 500}
                      c={offer.eligible ? 'navy.7' : 'navy.4'}
                      style={{
                        fontFamily: 'var(--mantine-font-family-monospace)',
                        letterSpacing: '0.06em',
                      }}
                    >
                      {offer.code}
                      {isApplied && ' ✓'}
                    </Text>
                    {/* navy.4 either way: navy.3 was 3.66:1, under AA. */}
                    <Text fz={11} c="navy.4" lh={1.35}>
                      {isApplied ? 'On this order' : offer.eligible ? offer.name : offer.reason}
                    </Text>
                  </Stack>
                  <Text
                    fz="sm"
                    c={offer.eligible ? 'navy.7' : 'navy.2'}
                    style={AMOUNT_STYLE}
                  >
                    {offer.eligible ? `−${formatPrice(offer.discount_amount)}` : '—'}
                  </Text>
                </Group>
              </UnstyledButton>
            );
          })}
        </Stack>
      </Collapse>
    </Stack>
  );
}
