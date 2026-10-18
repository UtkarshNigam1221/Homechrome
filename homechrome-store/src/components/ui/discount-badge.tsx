'use client';

import { Badge, BadgeProps } from '@mantine/core';

interface DiscountBadgeProps extends Omit<BadgeProps, 'children' | 'color' | 'variant'> {
  percent: number;
}

export function DiscountBadge({ percent, ...rest }: DiscountBadgeProps) {
  if (percent <= 0) return null;

  return (
    <Badge radius="sm" size="sm" color="red" variant="filled" styles={{ label: { fontWeight: 700 } }} {...rest}>
      {percent}% OFF
    </Badge>
  );
}
