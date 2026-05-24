'use client';

import { MapPinIcon } from '@heroicons/react/24/outline';
import {
  Alert,
  Button,
  Card,
  Center,
  Group,
  SimpleGrid,
  Stack,
  Title,
} from '@mantine/core';

import AddressForm from '@/components/checkout/AddressForm';
import { EmptyState } from '@/components/ui/empty-state';
import { useAddressManager } from '@/hooks/useAddressManager';

import { AddressListCard } from './AddressListCard';

export default function AddressesPage() {
  const {
    addresses,
    showForm,
    editingAddress,
    saving,
    deletingId,
    error,
    setShowForm,
    setEditingAddress,
    add,
    update,
    remove,
  } = useAddressManager();

  const showEmptyState = addresses.length === 0 && !showForm;
  const showList = addresses.length > 0 && !showForm && !editingAddress;

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <Title order={2} size="md">Saved Addresses</Title>
        {!showForm && !editingAddress && (
          <Button size="sm" color="brand" onClick={() => setShowForm(true)}>
            Add Address
          </Button>
        )}
      </Group>

      {error && <Alert color="red" title="Error">{error}</Alert>}

      {showForm && (
        <Card shadow="sm" radius="lg" padding="lg">
          <Stack gap="md">
            <Title order={3} size="md">Add New Address</Title>
            <AddressForm
              onSubmit={add}
              onCancel={() => setShowForm(false)}
              loading={saving}
            />
          </Stack>
        </Card>
      )}

      {editingAddress && (
        <Card shadow="sm" radius="lg" padding="lg">
          <Stack gap="md">
            <Title order={3} size="md">Edit Address</Title>
            <AddressForm
              initialData={editingAddress}
              onSubmit={update}
              onCancel={() => setEditingAddress(null)}
              loading={saving}
            />
          </Stack>
        </Card>
      )}

      {showEmptyState && (
        <Card shadow="sm" radius="lg" padding="lg">
          <EmptyState
            icon={<MapPinIcon strokeWidth={1} width={64} height={64} color="var(--mantine-color-dimmed)" />}
            title="No saved addresses"
            description="Add an address to speed up your checkout."
          />
          <Center>
            <Button color="brand" onClick={() => setShowForm(true)}>
              Add Address
            </Button>
          </Center>
        </Card>
      )}

      {showList && (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {addresses.map((addr) => (
            <AddressListCard
              key={addr.id}
              address={addr}
              isDeleting={deletingId === addr.id}
              onEdit={setEditingAddress}
              onDelete={remove}
            />
          ))}
        </SimpleGrid>
      )}
    </Stack>
  );
}
