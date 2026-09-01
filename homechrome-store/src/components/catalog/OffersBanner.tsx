'use client';

import { Box, Container, Group, Text, UnstyledButton } from '@mantine/core';
import { useClipboard } from '@mantine/hooks';
import { useMemo } from 'react';

import { PublicCoupon } from '@/types';

import { displayFont } from '@/app/fonts';
import { formatPrice } from '@/lib/utils';

const INDIGO = 'var(--mantine-color-navy-7)';
const COTTON = 'var(--mantine-color-brand-1)';
const CHALK = 'var(--mantine-color-brand-0)';
// Chalk on this red is 4.53:1 — over AA, but only just: do not lighten either side.
const RED = '#D92D20';

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

// Must track the span's own type and padding below, or the outline stops fitting the box.
const STAMP_CHAR_W = 12 * 0.6 + 12 * 0.09;
const STAMP_PAD_X = 20;
const STAMP_H = 12 * 1.5 + 12;

// In px, so a long code and a short one scallop identically.
const LOBE_PITCH = 9;
const LOBE_DEPTH = 2.2;

// Walked by arc length, not angle: on a box this wide, equal angles are unequal distances
// and the lobes pile up at the ends. Returns its assumed width — the caller must pin it.
function blockStamp(codeLength: number): { clip: string; width: number } {
  const w = codeLength * STAMP_CHAR_W + STAMP_PAD_X * 2;
  const a = w / 2;
  const b = STAMP_H / 2;

  const samples = 256;
  const ts: number[] = [];
  const arc: number[] = [];
  let s = 0;
  for (let i = 0; i <= samples; i += 1) {
    const t = (i / samples) * Math.PI * 2;
    if (i > 0) {
      const prev = ((i - 1) / samples) * Math.PI * 2;
      s += Math.hypot(a * (Math.cos(t) - Math.cos(prev)), b * (Math.sin(t) - Math.sin(prev)));
    }
    ts.push(t);
    arc.push(s);
  }
  const total = arc[samples];
  // A whole number, or the seam where the walk closes shows as a flat spot.
  const lobes = Math.max(8, Math.round(total / LOBE_PITCH));

  // Vertices by arc length too, or the long edges starve and the outline goes ragged.
  const steps = lobes * 6;
  const coords: string[] = [];
  let j = 0;
  for (let k = 0; k < steps; k += 1) {
    const target = (k / steps) * total;
    while (j < samples - 1 && arc[j + 1] < target) j += 1;
    const span = arc[j + 1] - arc[j] || 1;
    const t = ts[j] + ((target - arc[j]) / span) * (ts[j + 1] - ts[j]);

    const phase = (target / total) * lobes * Math.PI * 2;
    const inset = (LOBE_DEPTH * (1 - Math.cos(phase))) / 2;
    // Outward normal, so the dip cuts perpendicular to the edge.
    const nx = b * Math.cos(t);
    const ny = a * Math.sin(t);
    const n = Math.hypot(nx, ny) || 1;
    const x = a * Math.cos(t) - (inset * nx) / n;
    const y = b * Math.sin(t) - (inset * ny) / n;
    coords.push(`${(((x + a) / w) * 100).toFixed(2)}% ${(((y + b) / STAMP_H) * 100).toFixed(2)}%`);
  }
  return { clip: `polygon(${coords.join(', ')})`, width: w };
}

function CodeStamp({ code, tilt }: { code: string; tilt: string }) {
  const clipboard = useClipboard({ timeout: 2000 });
  const { copied } = clipboard;
  const { clip, width } = useMemo(() => blockStamp(code.length), [code.length]);

  return (
    <UnstyledButton
      onClick={() => clipboard.copy(code)}
      aria-label={`Copy coupon code ${code}`}
      // Focus ring lives on this unclipped box; clip-path on the child would eat it.
      style={{ borderRadius: 'var(--mantine-radius-xs)', lineHeight: 0, rotate: tilt }}
    >
      {/* aria-live: aria-label pins the button's name, so "Copied" is otherwise silent.
          userSelect: a refused clipboard write leaves selecting by hand as the only way. */}
      <span
        aria-live="polite"
        style={{
          display: 'block',
          minWidth: width,
          boxSizing: 'border-box',
          textAlign: 'center',
          userSelect: 'text',
          fontFamily: 'var(--mantine-font-family-monospace)',
          fontSize: '0.75rem',
          fontWeight: 700,
          letterSpacing: '0.09em',
          lineHeight: 1.5,
          color: CHALK,
          background: copied ? INDIGO : RED,
          clipPath: clip,
          padding: '0.375rem 1.25rem',
          transition: 'background 140ms ease',
        }}
      >
        {copied ? 'Copied' : code}
      </span>
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
      py="0.375rem"
      style={{ background: COTTON, borderBottom: '2px solid var(--mantine-color-brand-5)' }}
    >
      <Container size="xl">
        <Group justify="center" gap="xl" wrap="wrap">
          {shown.map((coupon, i) => {
            const { magnitude, terms } = offerParts(coupon);
            return (
              <Group
                key={coupon.code}
                gap="0.5rem"
                justify="center"
                // One per line on a phone would make the band as tall as the hero.
                visibleFrom={i > 0 ? 'sm' : undefined}
              >
                <Text
                  component="span"
                  c={INDIGO}
                  fz="1.0625rem"
                  fw={700}
                  lh={1.2}
                  style={{ fontFamily: displayFont.style.fontFamily }}
                >
                  {magnitude}
                </Text>
                <Text component="span" c={INDIGO} fz="0.8125rem" fw={500} lh={1.4}>
                  {terms}
                </Text>
                <CodeStamp code={coupon.code} tilt={i % 2 === 0 ? '-1.5deg' : '1.5deg'} />
              </Group>
            );
          })}
        </Group>
      </Container>
    </Box>
  );
}
