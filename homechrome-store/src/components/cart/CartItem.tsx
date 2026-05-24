'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { Box, Button, Card, Center, Group, Stack, Text } from '@mantine/core';
import Image from 'next/image';
import { useState } from 'react';

import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { formatPrice } from '@/lib/utils';
import { CartItem as CartItemType } from '@/types';

interface CartItemProps {
  item: CartItemType;
  onUpdateQuantity: (productId: string, quantity: number) => Promise<void>;
  onRemove: (productId: string) => Promise<void>;
}

export default function CartItem({ item, onUpdateQuantity, onRemove }: CartItemProps) {
  const [updating, setUpdating] = useState(false);
  const [removing, setRemoving] = useState(false);

  const handleQuantityChange = async (newQuantity: number) => {
    if (newQuantity < 1) return;
    setUpdating(true);
    try {
      await onUpdateQuantity(item.product_id, newQuantity);
    } finally {
      setUpdating(false);
    }
  };

  const handleRemove = async () => {
    setRemoving(true);
    try {
      await onRemove(item.product_id);
    } finally {
      setRemoving(false);
    }
  };

  return (
    <Card component="article" shadow="sm" radius="lg" padding="md">
      <Group gap="md" wrap="nowrap" align="stretch">
        <Box
          pos="relative"
          flex="none"
          style={{
            width: 96,
            height: 96,
            overflow: 'hidden',
            borderRadius: 'var(--mantine-radius-md)',
            background: 'var(--mantine-color-gray-1)',
          }}
          visibleFrom="sm"
          hiddenFrom="sm"
        >
          {/* mobile image */}
        </Box>
        <Box
          pos="relative"
          flex="none"
          w={{ base: 96, sm: 128 }}
          h={{ base: 96, sm: 128 }}
          style={{
            overflow: 'hidden',
            borderRadius: 'var(--mantine-radius-md)',
            background: 'var(--mantine-color-gray-1)',
          }}
        >
          {item.product_image ? (
            <Image
              src={item.product_image}
              alt={item.product_name}
              fill
              sizes="128px"
              style={{ objectFit: 'cover' }}
            />
          ) : (
            <Center bg="brand.1" h="100%">
              <PhotoIcon width={32} height={32} color="var(--mantine-color-brand-5)" opacity={0.4} />
            </Center>
          )}
        </Box>

        <Stack flex={1} gap="xs" justify="space-between">
          <Group justify="space-between" align="start" wrap="nowrap">
            <Stack gap={2}>
              <Text fw={500} size="sm" c="navy.7" lh={1.4}>
                {item.product_name}
              </Text>
              <Text size="xs" c="dimmed">SKU: {item.product_sku}</Text>
            </Stack>
            <Text fw={700} size="sm" c="navy.7">
              {formatPrice(item.total_price)}
            </Text>
          </Group>

          <Group justify="space-between" align="center" mt="auto">
            <QuantityStepper
              value={item.quantity}
              onIncrement={() => handleQuantityChange(item.quantity + 1)}
              onDecrement={() => handleQuantityChange(item.quantity - 1)}
              disableDecrement={item.quantity <= 1 || updating}
              disabled={updating}
              loading={updating}
              size="sm"
            />

            <Text size="sm" c="dimmed" visibleFrom="sm">
              {formatPrice(item.unit_price)} each
            </Text>

            <Button
              variant="light"
              color="red"
              size="xs"
              onClick={handleRemove}
              loading={removing}
            >
              Remove
            </Button>
          </Group>
        </Stack>
      </Group>
    </Card>
  );
}
