'use client';

import { Group, Radio, Skeleton, Stack, Text } from '@mantine/core';

import { formatPrice } from '@/lib/utils';
import { CourierOption } from '@/types';

interface ShippingOptionsProps {
  couriers: CourierOption[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  loading?: boolean;
}

export default function ShippingOptions({
  couriers,
  selectedId,
  onSelect,
  loading = false,
}: ShippingOptionsProps) {
  if (loading) {
    return (
      <Stack gap="sm">
        {[1, 2].map((i) => (
          <Skeleton key={i} height={80} radius="md" />
        ))}
      </Stack>
    );
  }

  if (couriers.length === 0) {
    return (
      <Text size="sm" c="dimmed">
        No shipping options available for this address.
      </Text>
    );
  }

  return (
    <Radio.Group
      value={selectedId?.toString() || ''}
      onChange={(val) => onSelect(Number(val))}
      aria-label="Shipping options"
    >
      <Stack gap="sm">
        {couriers.map((courier) => (
          <Radio.Card key={courier.id} value={courier.id.toString()} radius="md" p="md">
            <Group justify="space-between" wrap="nowrap" align="center">
              <Group align="center" gap="md" wrap="nowrap">
                <Radio.Indicator color="brand" />
                <Stack gap={2}>
                  <Text fw={500} c="navy.7">{courier.name}</Text>
                  <Text size="sm" c="dimmed">
                    Estimated delivery in {courier.estimated_days}{' '}
                    {courier.estimated_days === 1 ? 'day' : 'days'}
                  </Text>
                </Stack>
              </Group>
              <Text fw={600} c="navy.7">
                {courier.rate === 0 ? 'FREE' : formatPrice(courier.rate)}
              </Text>
            </Group>
          </Radio.Card>
        ))}
      </Stack>
    </Radio.Group>
  );
}
