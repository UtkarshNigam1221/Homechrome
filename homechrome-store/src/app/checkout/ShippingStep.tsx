'use client';

import { Alert, Anchor, Button, Card, Group, Stack, Text, Title } from '@mantine/core';

import ShippingOptions from '@/components/checkout/ShippingOptions';
import { Address, CourierOption } from '@/types';

interface ShippingStepProps {
  selectedAddress: Address | null;
  couriers: CourierOption[];
  selectedCourierId: number | null;
  serviceabilityLoading: boolean;
  serviceabilityError: string | null;
  onSelectCourier: (id: number) => void;
  onChangeAddress: () => void;
  onContinue: () => void;
}

export function ShippingStep({
  selectedAddress,
  couriers,
  selectedCourierId,
  serviceabilityLoading,
  serviceabilityError,
  onSelectCourier,
  onChangeAddress,
  onContinue,
}: ShippingStepProps) {
  return (
    <Card shadow="sm" radius="lg" padding="lg">
      <Stack gap="md">
        <Title order={2} size="md">Shipping Method</Title>

        {selectedAddress && (
          <Card bg="gray.0" radius="md" padding="sm" withBorder={false}>
            <Stack gap={4}>
              <Text fw={500} size="sm" c="navy.7">
                Delivering to: {selectedAddress.first_name} {selectedAddress.last_name}
              </Text>
              <Text size="sm" c="dimmed">
                {selectedAddress.address_line1}, {selectedAddress.city},{' '}
                {selectedAddress.state} - {selectedAddress.postal_code}
              </Text>
              <Anchor component="button" type="button" size="xs" onClick={onChangeAddress}>
                Change address
              </Anchor>
            </Stack>
          </Card>
        )}

        {serviceabilityError && (
          <Alert color="red">
            <Stack gap="xs">
              <Text size="sm">{serviceabilityError}</Text>
              <Anchor component="button" type="button" size="sm" onClick={onChangeAddress} c="red.7">
                Choose a different address
              </Anchor>
            </Stack>
          </Alert>
        )}

        <ShippingOptions
          couriers={couriers}
          selectedId={selectedCourierId}
          onSelect={onSelectCourier}
          loading={serviceabilityLoading}
        />

        {!serviceabilityLoading && !serviceabilityError && couriers.length > 0 && (
          <Group gap="sm">
            <Button onClick={onContinue} disabled={!selectedCourierId} color="brand">
              Continue to Review
            </Button>
            <Button variant="outline" color="navy" onClick={onChangeAddress}>
              Back
            </Button>
          </Group>
        )}
      </Stack>
    </Card>
  );
}
