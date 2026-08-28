'use client';

import { Box, Container, Group, Text } from '@mantine/core';

import { PublicCoupon } from '@/types';

import { formatPrice } from '@/lib/utils';

// "20% off" or "₹500 off", plus the minimum when there is one.
function offerLabel(coupon: PublicCoupon): string {
  const off =
    coupon.type === 'PERCENTAGE'
      ? `${coupon.value / 100}% off`
      : `${formatPrice(coupon.value)} off`;
  return coupon.min_order_value > 0
    ? `${off} above ${formatPrice(coupon.min_order_value)}`
    : off;
}

interface OffersBannerProps {
  coupons: PublicCoupon[];
}

export default function OffersBanner({ coupons }: OffersBannerProps) {
  if (coupons.length === 0) return null;

  return (
    <Box component="section" bg="teal.7" py="xs">
      <Container size="xl">
        <Group justify="center" gap="lg" wrap="wrap">
          {coupons.map((coupon) => (
            <Text key={coupon.code} c="white" fz="sm" fw={500}>
              {offerLabel(coupon)} with{' '}
              <Text span fw={700}>
                {coupon.code}
              </Text>
            </Text>
          ))}
        </Group>
      </Container>
    </Box>
  );
}
