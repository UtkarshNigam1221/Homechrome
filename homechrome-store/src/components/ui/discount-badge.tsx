'use client';

import { Badge, BadgeProps } from '@mantine/core';

interface DiscountBadgeProps extends Omit<BadgeProps, 'children' | 'color' | 'variant'> {
  percent: number;
  variant?: 'solid' | 'soft';
}

export function DiscountBadge({ percent, variant = 'soft', ...rest }: DiscountBadgeProps) {
  if (percent <= 0) return null;
  return (
    <Badge
      color="red"
      variant={variant === 'solid' ? 'filled' : 'light'}
      radius="xl"
      size="sm"
      {...rest}
    >
      {percent}% OFF
    </Badge>
  );
}
