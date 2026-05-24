'use client';

import { Anchor, Badge, Button, Card, Group, Radio, Stack, Text, Title } from '@mantine/core';
import { useRouter } from 'next/navigation';

import AddressForm from '@/components/checkout/AddressForm';
import { Address } from '@/types';

interface AddressStepProps {
  addresses: Address[];
  selectedAddressId: string | null;
  showAddressForm: boolean;
  addressSaving: boolean;
  onSelectAddress: (id: string) => void;
  onToggleForm: (show: boolean) => void;
  onSaveAddress: (data: Omit<Address, 'id'>) => Promise<void>;
  onContinue: () => void;
}

export function AddressStep({
  addresses,
  selectedAddressId,
  showAddressForm,
  addressSaving,
  onSelectAddress,
  onToggleForm,
  onSaveAddress,
  onContinue,
}: AddressStepProps) {
  const router = useRouter();

  return (
    <Card shadow="sm" radius="lg" padding="lg">
      <Stack gap="md">
        <Title order={2} size="md">Shipping Address</Title>

        {addresses.length > 0 && !showAddressForm && (
          <Stack gap="md">
            <Radio.Group
              value={selectedAddressId || ''}
              onChange={onSelectAddress}
              aria-label="Select address"
            >
              <Stack gap="sm">
                {addresses.map((addr) => (
                  <Radio.Card
                    key={addr.id}
                    value={addr.id}
                    radius="md"
                    p="md"
                  >
                    <Group align="start" wrap="nowrap">
                      <Radio.Indicator color="brand" />
                      <Stack gap={2} flex={1}>
                        <Group gap="xs" align="center">
                          <Text fw={500} c="navy.7">
                            {addr.first_name} {addr.last_name}
                          </Text>
                          {addr.is_default && (
                            <Badge color="brand" variant="light" size="xs">Default</Badge>
                          )}
                        </Group>
                        <Text size="sm" c="dimmed">
                          {addr.address_line1}
                          {addr.address_line2 && `, ${addr.address_line2}`}
                        </Text>
                        <Text size="sm" c="dimmed">
                          {addr.city}, {addr.state} - {addr.postal_code}
                        </Text>
                        <Text size="sm" c="dimmed">Phone: {addr.phone}</Text>
                      </Stack>
                    </Group>
                  </Radio.Card>
                ))}
              </Stack>
            </Radio.Group>

            <Anchor component="button" type="button" size="sm" onClick={() => onToggleForm(true)}>
              + Add a new address
            </Anchor>

            <Group>
              <Button onClick={onContinue} disabled={!selectedAddressId} color="brand">
                Continue to Shipping
              </Button>
            </Group>
          </Stack>
        )}

        {(addresses.length === 0 || showAddressForm) && (
          <AddressForm
            onSubmit={onSaveAddress}
            onCancel={() => {
              if (addresses.length > 0) onToggleForm(false);
              else router.push('/cart');
            }}
            loading={addressSaving}
          />
        )}
      </Stack>
    </Card>
  );
}
