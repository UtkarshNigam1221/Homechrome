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

import { formatPrice } from '@/lib/utils';

interface CartSummaryProps {
  subtotal: number;
  itemCount: number;
  isAuthenticated: boolean;
}

const FREE_SHIPPING_THRESHOLD = 99900;

export default function CartSummary({ subtotal, itemCount, isAuthenticated }: CartSummaryProps) {
  const shippingEstimate = subtotal >= FREE_SHIPPING_THRESHOLD ? 0 : 7900;
  const total = subtotal + shippingEstimate;
  const amountToFreeShipping = FREE_SHIPPING_THRESHOLD - subtotal;

  return (
    <Card shadow="sm" radius="lg" padding="md">
      <Stack gap="md">
        <Title order={2} size="md">Order Summary</Title>

        <Stack gap="xs">
          <Group justify="space-between">
            <Text size="sm" c="dimmed">
              Subtotal ({itemCount} {itemCount === 1 ? 'item' : 'items'})
            </Text>
            <Text size="sm" fw={500} c="navy.7">{formatPrice(subtotal)}</Text>
          </Group>

          <Group justify="space-between">
            <Text size="sm" c="dimmed">Shipping estimate</Text>
            {shippingEstimate === 0 ? (
              <Text size="sm" fw={500} c="teal.7">Free</Text>
            ) : (
              <Text size="sm" fw={500} c="navy.7">{formatPrice(shippingEstimate)}</Text>
            )}
          </Group>

          <Divider />

          <Group justify="space-between">
            <Text fw={600} c="navy.7">Total</Text>
            <Text fw={700} c="navy.7">{formatPrice(total)}</Text>
          </Group>
        </Stack>

        {subtotal > 0 && amountToFreeShipping > 0 && (
          <Text size="xs" c="dimmed">
            Add {formatPrice(amountToFreeShipping)} more for free shipping!
          </Text>
        )}

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
