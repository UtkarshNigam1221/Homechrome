'use client';

import { Anchor, Button, Card, Group, Stack, Text, Title } from '@mantine/core';

import { formatPrice } from '@/lib/utils';
import { Address, CartItem, CourierOption } from '@/types';

interface ReviewStepProps {
  selectedAddress: Address | null;
  selectedCourier: CourierOption | null;
  items: CartItem[];
  initiating: boolean;
  onChangeAddress: () => void;
  onChangeShipping: () => void;
  onPayNow: () => void;
}

export function ReviewStep({
  selectedAddress,
  selectedCourier,
  items,
  initiating,
  onChangeAddress,
  onChangeShipping,
  onPayNow,
}: ReviewStepProps) {
  return (
    <Card shadow="sm" radius="lg" padding="lg">
      <Stack gap="md">
        <Title order={2} size="md">Review Your Order</Title>

        {selectedAddress && (
          <SummaryBlock title="Shipping Address" onChange={onChangeAddress}>
            {selectedAddress.first_name} {selectedAddress.last_name},{' '}
            {selectedAddress.address_line1}, {selectedAddress.city},{' '}
            {selectedAddress.state} - {selectedAddress.postal_code}
          </SummaryBlock>
        )}

        {selectedCourier && (
          <SummaryBlock title="Shipping Method" onChange={onChangeShipping}>
            {selectedCourier.name} - Est. {selectedCourier.estimated_days}{' '}
            {selectedCourier.estimated_days === 1 ? 'day' : 'days'}
          </SummaryBlock>
        )}

        <Stack gap="xs">
          <Title order={3} size="sm">Items</Title>
          {items.map((item) => (
            <Group key={item.product_id} justify="space-between">
              <Text size="sm" c="dimmed">
                {item.product_name} x {item.quantity}
              </Text>
              <Text size="sm" c="navy.7">{formatPrice(item.total_price)}</Text>
            </Group>
          ))}
        </Stack>

        <Group gap="sm">
          <Button onClick={onPayNow} loading={initiating} color="brand">
            Pay Now
          </Button>
          <Button variant="outline" color="navy" onClick={onChangeShipping}>
            Back
          </Button>
        </Group>
      </Stack>
    </Card>
  );
}

interface SummaryBlockProps {
  title: string;
  onChange: () => void;
  children: React.ReactNode;
}

function SummaryBlock({ title, onChange, children }: SummaryBlockProps) {
  return (
    <Card bg="gray.0" radius="md" padding="md" withBorder={false}>
      <Stack gap={4}>
        <Group justify="space-between" align="center">
          <Title order={3} size="sm">{title}</Title>
          <Anchor component="button" type="button" size="xs" onClick={onChange}>
            Change
          </Anchor>
        </Group>
        <Text size="sm" c="dimmed">{children}</Text>
      </Stack>
    </Card>
  );
}
