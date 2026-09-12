'use client';

import { ClipboardDocumentIcon } from '@heroicons/react/24/outline';
import { SparklesIcon } from '@heroicons/react/24/solid';
import { Box, Container, Group, Text, UnstyledButton } from '@mantine/core';
import { useClipboard } from '@mantine/hooks';

import { PublicCoupon } from '@/types';

import { formatPrice } from '@/lib/utils';

const BAND = 'var(--mantine-color-navy-1)';
const INK = 'var(--mantine-color-navy-7)';

function offerParts(coupon: PublicCoupon): { magnitude: string; terms: string } {
  const magnitude =
    coupon.type === 'PERCENTAGE' ? `${coupon.value / 100}%` : formatPrice(coupon.value);
  const above = coupon.min_order_value > 0 ? ` above ${formatPrice(coupon.min_order_value)}` : '';
  // A cap on a FIXED coupon would read "₹500 off · up to ₹300"; nothing forbids it.
  const cap =
    coupon.type === 'PERCENTAGE' && coupon.max_discount
      ? ` · up to ${formatPrice(coupon.max_discount)}`
      : '';
  return { magnitude, terms: `off${above}${cap}` };
}

/** Dashed chip carrying the code; clicking it copies. */
function CodeChip({ code }: { code: string }) {
  const clipboard = useClipboard({ timeout: 2000 });
  const { copied } = clipboard;

  return (
    <UnstyledButton
      onClick={() => clipboard.copy(code)}
      aria-label={`Copy coupon code ${code}`}
      style={{ borderRadius: 'var(--mantine-radius-sm)' }}
    >
      {/* aria-live: aria-label pins the button's name, so "Copied" is otherwise silent.
          userSelect: a refused clipboard write leaves selecting by hand as the only way. */}
      <Group
        component="span"
        aria-live="polite"
        gap={5}
        wrap="nowrap"
        align="center"
        px={9}
        py={2}
        style={{
          userSelect: 'text',
          borderRadius: 'var(--mantine-radius-sm)',
          border: `1px dashed var(--mantine-color-${copied ? 'brand-5' : 'navy-4'})`,
          background: `var(--mantine-color-${copied ? 'brand-1' : 'navy-2'})`,
          transition: 'background 140ms ease, border-color 140ms ease',
        }}
      >
        <Text
          component="span"
          fz={11}
          fw={700}
          c="brand.6"
          tt="uppercase"
          style={{ letterSpacing: '0.1em' }}
        >
          {copied ? 'Copied' : code}
        </Text>
        {!copied && <ClipboardDocumentIcon width={11} height={11} color="var(--mantine-color-brand-6)" />}
      </Group>
    </UnstyledButton>
  );
}

// The checkout picker lists them all, so the band need not. Newest two, not best two:
// ranking a percentage against a fixed amount with no cart would be a guess.
const MAX_OFFERS = 2;

interface OffersBannerProps {
  coupons: PublicCoupon[];
}

export default function OffersBanner({ coupons }: OffersBannerProps) {
  if (coupons.length === 0) return null;

  const shown = coupons.slice(0, MAX_OFFERS);

  return (
    <Box
      component="section"
      aria-label="Current offers"
      py={6}
      style={{ background: BAND, borderBottom: '1px solid var(--mantine-color-navy-2)' }}
    >
      <Container size="xl">
        <Group justify="center" gap="md" wrap="wrap">
          <SparklesIcon width={15} height={15} color="var(--mantine-color-brand-5)" />
          {shown.map((coupon, i) => {
            const { magnitude, terms } = offerParts(coupon);
            return (
              <Group
                key={coupon.code}
                gap={7}
                justify="center"
                wrap="nowrap"
                // One per line on a phone would make the band as tall as the hero.
                visibleFrom={i > 0 ? 'sm' : undefined}
              >
                <Text component="span" c={INK} fz="0.8125rem" fw={500} lh={1.4}>
                  {magnitude} {terms} — use code
                </Text>
                <CodeChip code={coupon.code} />
              </Group>
            );
          })}
          {/* Matches the shipping policy's own wording: free on all orders, no threshold. */}
          <Text component="span" c={INK} fz="0.8125rem" fw={500} visibleFrom="md">
            | Free Shipping Across India
          </Text>
        </Group>
      </Container>
    </Box>
  );
}
