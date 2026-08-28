'use client';

import {
  Anchor,
  Button,
  Card,
  Divider,
  Group,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Link from 'next/link';

import CouponInput from '@/components/cart/CouponInput';
import { useCoupon } from '@/hooks/useCoupon';
import { formatPrice } from '@/lib/utils';

interface CartSummaryProps {
  subtotal: number;
  itemCount: number;
  isAuthenticated: boolean;
}

export default function CartSummary({ subtotal, itemCount, isAuthenticated }: CartSummaryProps) {
  // Shipping is free — deliveries are scheduled manually.
  const coupon = useCoupon(subtotal, isAuthenticated);

  return (
    <Card shadow="sm" radius="lg" padding="md">
      <Stack gap="md">
        <Title order={2} size="md">Order Summary</Title>

        <CouponInput
          appliedCode={coupon.code}
          appliedDiscount={coupon.discount}
          onApplied={coupon.apply}
          onRemoved={coupon.remove}
        />

        <Stack gap="xs">
          <Group justify="space-between">
            <Text size="sm" c="dimmed">
              Subtotal ({itemCount} {itemCount === 1 ? 'item' : 'items'})
            </Text>
            <Text size="sm" fw={500} c="navy.7">{formatPrice(subtotal)}</Text>
          </Group>

          {/* Rows render only when non-zero — a discount-free cart looks exactly
              as it did before. No tax row: prices are tax-inclusive. */}
          {coupon.discount > 0 && (
            <Group justify="space-between">
              <Text size="sm" c="dimmed">
                Discount{coupon.code ? ` (${coupon.code})` : ''}
              </Text>
              <Text size="sm" fw={500} c="teal.7">-{formatPrice(coupon.discount)}</Text>
            </Group>
          )}

          <Group justify="space-between">
            <Text size="sm" c="dimmed">Shipping</Text>
            <Text size="sm" fw={500} c="teal.7">Free</Text>
          </Group>

          <Divider />

          <Group justify="space-between">
            <Text fw={600} c="navy.7">Total</Text>
            <Text fw={700} c="navy.7">{formatPrice(coupon.total)}</Text>
          </Group>
        </Stack>

        <Button
          component={Link}
          href={isAuthenticated ? '/checkout' : '/login?redirect=/cart'}
          color="brand"
          size="lg"
          fullWidth
          disabled={itemCount === 0}
        >
          {isAuthenticated ? 'Proceed to Checkout' : 'Sign in to Checkout'}
        </Button>

        <Anchor
          component={Link}
          href="/products"
          c="brand"
          size="sm"
          underline="never"
          ta="center"
        >
          Continue Shopping
        </Anchor>
      </Stack>
    </Card>
  );
}
