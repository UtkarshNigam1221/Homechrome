'use client';

import { Box, Card, Center, Divider, Group, ScrollArea, Stack, Text, Title } from '@mantine/core';
import { AssetImage } from '@/components/ui/asset-image';

import { formatPrice } from '@/lib/utils';
import { CartItem, CourierOption } from '@/types';

interface OrderSummaryProps {
  items: CartItem[];
  subtotal: number;
  shippingCourier?: CourierOption | null;
}

export default function OrderSummary({
  items,
  subtotal,
  shippingCourier,
}: OrderSummaryProps) {
  const shippingCost = shippingCourier?.rate ?? 0;
  const total = subtotal + shippingCost;

  return (
    <Card shadow="sm" radius="lg" padding="md" withBorder>
      <Stack gap="md">
        <Title order={3} size="md">Order Summary</Title>

        <ScrollArea.Autosize mah={256}>
          <Stack gap="sm">
            {items.map((item) => (
              <Group key={item.product_id} gap="sm" wrap="nowrap">
                <Box
                  pos="relative"
                  w={64}
                  h={64}
                  flex="none"
                  style={{
                    overflow: 'hidden',
                    borderRadius: 'var(--mantine-radius-md)',
                    background: 'var(--mantine-color-gray-1)',
                  }}
                >
                  {item.product_image ? (
                    <AssetImage
                      src={item.product_image}
                      alt={item.product_name}
                      sizes="64px"
                      width={64}
                      height={64}
                      style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                    />
                  ) : (
                    <Center h="100%">
                      <Text size="xs" c="dimmed">No image</Text>
                    </Center>
                  )}
                </Box>
                <Group flex={1} justify="space-between" align="start" wrap="nowrap">
                  <Stack gap={2}>
                    <Text size="sm" fw={500} c="navy.7" lineClamp={2}>
                      {item.product_name}
                    </Text>
                    <Text size="xs" c="dimmed">Qty: {item.quantity}</Text>
                  </Stack>
                  <Text size="sm" fw={500} c="navy.7">
                    {formatPrice(item.total_price)}
                  </Text>
                </Group>
              </Group>
            ))}
          </Stack>
        </ScrollArea.Autosize>

        <Divider />

        <Stack gap="xs">
          <Group justify="space-between">
            <Text size="sm" c="dimmed">Subtotal</Text>
            <Text size="sm" c="navy.7">{formatPrice(subtotal)}</Text>
          </Group>
          <Group justify="space-between">
            <Text size="sm" c="dimmed">Shipping</Text>
            <Text size="sm" c="navy.7">
              {shippingCourier
                ? shippingCost === 0
                  ? 'FREE'
                  : formatPrice(shippingCost)
                : '--'}
            </Text>
          </Group>
          <Divider />
          <Group justify="space-between">
            <Text fw={600} c="navy.7">Total</Text>
            <Text fw={600} c="navy.7">{formatPrice(total)}</Text>
          </Group>
        </Stack>
      </Stack>
    </Card>
  );
}
