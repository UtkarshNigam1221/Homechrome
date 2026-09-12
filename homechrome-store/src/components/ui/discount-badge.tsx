'use client';

import { Badge, BadgeProps } from '@mantine/core';

interface DiscountBadgeProps extends Omit<BadgeProps, 'children' | 'color' | 'variant'> {
  percent: number;
}

// The design flips the badge to its green at a steep discount, so the headline
// saving reads differently from an everyday one.
const STANDOUT_PERCENT = 50;

// Design tokens: primary-fixed / on-primary-fixed, tertiary-fixed /
// on-tertiary-fixed-variant. Neither has a Mantine scale slot.
const EVERYDAY = { bg: 'var(--mantine-color-brand-1)', fg: '#3A0B00' };
const STANDOUT = { bg: 'var(--mantine-color-leaf-1)', fg: 'var(--mantine-color-leaf-6)' };

export function DiscountBadge({ percent, ...rest }: DiscountBadgeProps) {
  if (percent <= 0) return null;

  const tone = percent >= STANDOUT_PERCENT ? STANDOUT : EVERYDAY;

  return (
    <Badge
      radius="sm"
      size="sm"
      variant="filled"
      styles={{
        root: { backgroundColor: tone.bg, color: tone.fg },
        label: { fontWeight: 700, letterSpacing: '0.02em' },
      }}
      {...rest}
    >
      {percent}% OFF
    </Badge>
  );
}
