'use client';

import { Badge, Button, Card, Group, Stack, Text } from '@mantine/core';
import { modals } from '@mantine/modals';

import { Address } from '@/types';

interface AddressListCardProps {
  address: Address;
  isDeleting: boolean;
  onEdit: (addr: Address) => void;
  onDelete: (id: string) => void;
}

export function AddressListCard({ address, isDeleting, onEdit, onDelete }: AddressListCardProps) {
  const openDeleteModal = () =>
    modals.openConfirmModal({
      title: 'Delete Address',
      children: (
        <Text size="sm">
          Are you sure you want to delete this address? This action cannot be undone.
        </Text>
      ),
      labels: { confirm: 'Delete', cancel: 'Cancel' },
      confirmProps: { color: 'red' },
      onConfirm: () => onDelete(address.id),
    });

  return (
    <Card shadow="sm" radius="lg" padding="md">
      <Stack gap="sm">
        <Group gap="xs" align="center">
          <Text fw={500} c="navy.7">
            {address.first_name} {address.last_name}
          </Text>
          {address.is_default && (
            <Badge color="brand" variant="light" size="sm" radius="sm">
              Default
            </Badge>
          )}
        </Group>

        <Stack gap={2}>
          <Text size="sm" c="dimmed">{address.address_line1}</Text>
          {address.address_line2 && <Text size="sm" c="dimmed">{address.address_line2}</Text>}
          <Text size="sm" c="dimmed">
            {address.city}, {address.state} - {address.postal_code}
          </Text>
          <Text size="sm" c="dimmed">Phone: {address.phone}</Text>
        </Stack>

        <Group gap="xs">
          <Button variant="outline" color="navy" size="xs" onClick={() => onEdit(address)}>
            Edit
          </Button>
          <Button
            variant="light"
            color="red"
            size="xs"
            loading={isDeleting}
            onClick={openDeleteModal}
          >
            Delete
          </Button>
        </Group>
      </Stack>
    </Card>
  );
}
