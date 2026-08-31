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
// The one hot colour on the page, carried by the stamp alone. Chalk on it is 4.53:1 —
// over AA for normal text, but only just, so this pair should not be lightened further.
// Kept local rather than added to the theme: a Mantine colour needs a ten-shade ramp and
// only this one element ever wants red.
const RED = '#D92D20';

// The magnitude is what a shopper scans for, so it is split out and set apart from the
// terms that qualify it.
function offerParts(coupon: PublicCoupon): { magnitude: string; terms: string } {
  const magnitude =
    coupon.type === 'PERCENTAGE' ? `${coupon.value / 100}%` : formatPrice(coupon.value);
  const above = coupon.min_order_value > 0 ? ` above ${formatPrice(coupon.min_order_value)}` : '';
  // A cap only means anything on a percentage; on a fixed amount it would read
  // "₹500 off · up to ₹300", which the admin schema does not currently prevent.
  const cap =
    coupon.type === 'PERCENTAGE' && coupon.max_discount
      ? ` · up to ${formatPrice(coupon.max_discount)}`
      : '';
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
// Walked by arc length, not by angle. On a badge four times wider than it is tall, equal
// steps in angle are wildly unequal in distance, which piles the lobes up at the two ends
// and flattens the long edges — the outline comes out an amoeba. Stepping by distance
// travelled puts every lobe the same span apart the whole way round.
//
// Returns the width it assumed as well as the outline, because clip-path percentages
// resolve against the element's real box: the caller has to pin that box to this number,
// or the shorter "Copied" label — and any mono face whose advance is not 0.6em — resizes
// the box and squeezes a polygon that was cut for a different one.
function blockStamp(codeLength: number): { clip: string; width: number } {
  const w = codeLength * STAMP_CHAR_W + STAMP_PAD_X * 2;
  const a = w / 2;
  const b = STAMP_H / 2;

  // Arc length at uniform steps in angle, to be inverted below.
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
  // A whole number of lobes, or the seam where the walk closes shows as a flat spot.
  const lobes = Math.max(8, Math.round(total / LOBE_PITCH));

  // Vertices are placed by arc length too, not just the lobe phase. Stepping the angle
  // uniformly spends vertices where the ellipse is slowest — the pointed ends — and
  // starves the long top and bottom edges, which are the most visible part of the badge:
  // a 20-character code got about two and a half vertices per lobe there, and the outline
  // came out a ragged leaf rather than a stamp.
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
    // Outward normal of the ellipse at t, so the dip cuts perpendicular to the edge.
    const nx = b * Math.cos(t);
    const ny = a * Math.sin(t);
    const n = Math.hypot(nx, ny) || 1;
    const x = a * Math.cos(t) - (inset * nx) / n;
    const y = b * Math.sin(t) - (inset * ny) / n;
    coords.push(`${(((x + a) / w) * 100).toFixed(2)}% ${(((y + b) / STAMP_H) * 100).toFixed(2)}%`);
  }
  return { clip: `polygon(${coords.join(', ')})`, width: w };
}

// The code is the one thing a customer carries to checkout, so it is a control, not a
// label — clicking it copies rather than making them transcribe it by eye.
function CodeStamp({ code, tilt }: { code: string; tilt: string }) {
  const clipboard = useClipboard({ timeout: 2000 });
  const { copied } = clipboard;
  const { clip, width } = useMemo(() => blockStamp(code.length), [code.length]);

  return (
    <UnstyledButton
      onClick={() => clipboard.copy(code)}
      aria-label={`Copy coupon code ${code}`}
      // The focus ring lives here, on the unclipped box — clip-path would eat it. Its
      // colour comes from the site-wide override in globals.css, because the theme's
      // primary shade is unreadable as an indicator on these light grounds.
      //
      // The tilt is what keeps it from reading as a generated shape: a block is pressed
      // by hand and never lands square. Neighbouring stamps lean opposite ways, or two
      // of them read as one sticker duplicated rather than two things placed.
      style={{ borderRadius: 'var(--mantine-radius-xs)', lineHeight: 0, rotate: tilt }}
    >
      {/* aria-label fixes the button's name to "Copy coupon code X", so the swap to
          "Copied" is silent without a live region to announce it. Text stays selectable:
          when the clipboard write is refused nothing visible happens, and highlighting
          the code by hand is the only way left to take it to checkout. */}
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

// The band advertises; the checkout picker enumerates. Every live coupon is already
// listed there, priced against the real cart, so the band does not have to be complete —
// and it should not be: each offer brings its own red stamp, and past two the red stops
// being an accent. Backend order is kept as-is. Ranking them here would mean pricing a
// percentage against a fixed amount with no cart to price against, which is a guess that
// could put the worse offer first. The consequence is that these are the newest two, not
// the best two: a new small coupon displaces a standing large one until checkout.
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
                // so only the first survives below sm. Long terms can still wrap it onto
                // a second line at 360px, which is the intended floor rather than a bug.
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
