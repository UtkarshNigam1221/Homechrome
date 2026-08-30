'use client';

import { Box, Container, Group, Text, UnstyledButton } from '@mantine/core';
import { useClipboard } from '@mantine/hooks';
import { useMemo } from 'react';

import { PublicCoupon } from '@/types';

import { displayFont } from '@/app/fonts';
import { formatPrice } from '@/lib/utils';

// Undyed cotton ground closed by a sand rule — the selvedge where a handloom border
// meets the field. Light on purpose: it has to separate the white header from the navy
// hero rather than blend into either.
const INDIGO = 'var(--mantine-color-navy-7)';
const COTTON = 'var(--mantine-color-brand-1)';
const CHALK = 'var(--mantine-color-brand-0)';
// The one hot colour on the page, carried by the stamp alone. 6.1:1 under chalk text.
const RED = '#D92D20';

// The magnitude is what a shopper scans for, so it is split out and set apart from the
// terms that qualify it.
function offerParts(coupon: PublicCoupon): { magnitude: string; terms: string } {
  const magnitude =
    coupon.type === 'PERCENTAGE' ? `${coupon.value / 100}%` : formatPrice(coupon.value);
  const above = coupon.min_order_value > 0 ? ` above ${formatPrice(coupon.min_order_value)}` : '';
  const cap = coupon.max_discount ? ` · up to ${formatPrice(coupon.max_discount)}` : '';
  return { magnitude, terms: `off${above}${cap}` };
}

// The stamp's rendered size, from the type it holds: 12px mono at 0.09em tracking inside
// 20px of side padding, on a line 1.5 tall inside 6px of vertical padding. Only the ratio
// matters — it is what stops a wide badge scalloping unevenly.
const STAMP_CHAR_W = 12 * 0.6 + 12 * 0.09;
const STAMP_PAD_X = 20;
const STAMP_H = 12 * 1.5 + 12;

// Lobe pitch and depth in px, so a long code and a short one scallop identically.
const LOBE_PITCH = 9;
const LOBE_DEPTH = 2.2;

// A block-print stamp: the carved-block seal pressed onto handloom cloth, not the spiked
// sticker of a clearance rack. Scallops rather than spikes are the register this brand
// is in.
//
// Walked by arc length, not by angle. On a badge six times wider than it is tall, equal
// steps in angle are wildly unequal in distance, which piles the lobes up at the two ends
// and flattens the long edges — the outline comes out an amoeba. Stepping by distance
// travelled puts every lobe the same span apart the whole way round.
function blockStamp(codeLength: number): string {
  const w = codeLength * STAMP_CHAR_W + STAMP_PAD_X * 2;
  const a = w / 2;
  const b = STAMP_H / 2;

  const samples = 720;
  const pts: { t: number; s: number }[] = [];
  let s = 0;
  for (let i = 0; i <= samples; i += 1) {
    const t = (i / samples) * Math.PI * 2;
    if (i > 0) {
      const prev = ((i - 1) / samples) * Math.PI * 2;
      const dx = a * (Math.cos(t) - Math.cos(prev));
      const dy = b * (Math.sin(t) - Math.sin(prev));
      s += Math.hypot(dx, dy);
    }
    pts.push({ t, s });
  }
  const total = pts[pts.length - 1].s;
  // A whole number of lobes, or the seam where the walk closes shows as a flat spot.
  const lobes = Math.max(8, Math.round(total / LOBE_PITCH));

  const coords: string[] = [];
  const stride = Math.ceil(samples / 200);
  for (let i = 0; i < samples; i += stride) {
    const { t, s: arc } = pts[i];
    const phase = (arc / total) * lobes * Math.PI * 2;
    const inset = (LOBE_DEPTH * (1 - Math.cos(phase))) / 2;
    // Outward normal of the ellipse at t, so the dip cuts perpendicular to the edge.
    const nx = b * Math.cos(t);
    const ny = a * Math.sin(t);
    const n = Math.hypot(nx, ny) || 1;
    const x = a * Math.cos(t) - (inset * nx) / n;
    const y = b * Math.sin(t) - (inset * ny) / n;
    coords.push(`${(((x + a) / w) * 100).toFixed(2)}% ${(((y + b) / STAMP_H) * 100).toFixed(2)}%`);
  }
  return `polygon(${coords.join(', ')})`;
}

// The code is the one thing a customer carries to checkout, so it is a control, not a
// label — clicking it copies rather than making them transcribe it by eye.
function CodeStamp({ code, tilt }: { code: string; tilt: string }) {
  const clipboard = useClipboard({ timeout: 2000 });
  const { copied } = clipboard;
  const clip = useMemo(() => blockStamp(code.length), [code.length]);

  return (
    <UnstyledButton
      onClick={() => clipboard.copy(code)}
      aria-label={`Copy coupon code ${code}`}
      // The focus ring lives here, on the unclipped box — clip-path would eat it. The
      // tilt is what keeps it from reading as a generated shape: a block is pressed by
      // hand and never lands square. Neighbouring stamps lean opposite ways, or two of
      // them read as one sticker duplicated rather than two things placed.
      style={{ borderRadius: 'var(--mantine-radius-xs)', lineHeight: 0, rotate: tilt }}
    >
      {/* aria-label fixes the button's name to "Copy coupon code X", so the swap to
          "Copied" is silent without a live region to announce it. */}
      <span
        aria-live="polite"
        style={{
          display: 'block',
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

// The band advertises; the checkout picker enumerates. Every live coupon is already
// listed there, priced against the real cart, so the band does not have to be complete —
// and it should not be: each offer brings its own red stamp, and past two the red stops
// being an accent. Backend order is kept as-is. Ranking them here would mean pricing a
// percentage against a fixed amount with no cart to price against, which is a guess that
// could put the worse offer first.
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
                // One offer per line on a phone would make the band as tall as the hero,
                // so only the first survives below sm.
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
