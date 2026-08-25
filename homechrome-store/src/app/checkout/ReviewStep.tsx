'use client';

import { Anchor, Box, Button, Card, Group, Stack, Text, Title } from '@mantine/core';

import CouponInput from '@/components/cart/CouponInput';
import InlineLoaderOverlay from '@/components/ui/InlineLoaderOverlay';
import { formatPrice } from '@/lib/utils';
import { Address, CartItem } from '@/types';

interface ReviewStepProps {
  selectedAddress: Address | null;
  items: CartItem[];
  initiating: boolean;
  initiatingCheckout: boolean;
  couponCode: string | null;
  couponDiscount: number;
  onChangeAddress: () => void;
  onPayNow: () => void;
  onCouponApplied: (code: string, discount: number) => void;
  onCouponRemoved: () => void;
}

export function ReviewStep({
  selectedAddress,
  items,
  initiating,
  initiatingCheckout,
  couponCode,
  couponDiscount,
  onChangeAddress,
  onPayNow,
  onCouponApplied,
  onCouponRemoved,
}: ReviewStepProps) {
  return (
    <Box pos="relative">
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

        <Stack gap="xs">
          <Title order={3} size="sm">Coupon</Title>
          <CouponInput
            appliedCode={couponCode ?? undefined}
            appliedDiscount={couponDiscount}
            onApplied={onCouponApplied}
            onRemoved={onCouponRemoved}
          />
        </Stack>

        <Group gap="sm">
          <Button onClick={onPayNow} loading={initiating} color="brand">
            Pay Now
          </Button>
          <Button variant="outline" color="navy" onClick={onChangeAddress}>
            Back
          </Button>
        </Group>
      </Stack>
    </Card>
    <InlineLoaderOverlay visible={initiatingCheckout} size="md" label="Initiating payment" />
    </Box>
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
