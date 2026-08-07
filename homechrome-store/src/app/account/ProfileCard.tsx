'use client';

import {
  Alert,
  Button,
  Card,
  Group,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { useAuthStore } from '@/stores/auth';
import { Customer } from '@/types';

export function ProfileCard({ customer }: { customer: Customer }) {
  const checkAuth = useAuthStore((s) => s.checkAuth);

  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addressCount = customer.addresses?.length ?? 0;

  const form = useForm({
    initialValues: {
      first_name: customer.first_name || '',
      last_name: customer.last_name || '',
      email: customer.email || '',
    },
    validate: {
      first_name: (v) => (!v.trim() ? 'First name is required' : null),
      last_name: (v) => (!v.trim() ? 'Last name is required' : null),
      email: (v) => {
        const trimmed = v.trim();
        if (!trimmed) return null;
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed) ? null : 'Enter a valid email';
      },
    },
  });

  const startEditing = () => {
    form.setValues({
      first_name: customer.first_name || '',
      last_name: customer.last_name || '',
      email: customer.email || '',
    });
    setError(null);
    setEditing(true);
  };

  const handleSubmit = form.onSubmit(async (values) => {
    setSaving(true);
    setError(null);
    try {
      // PATCH /me ignores empty strings server-side, so a cleared email keeps
      // its previous value rather than being removed.
      await api.patch(ROUTES.ME.PROFILE, {
        first_name: values.first_name.trim(),
        last_name: values.last_name.trim(),
        email: values.email.trim(),
      });
      await checkAuth();
      setEditing(false);
    } catch {
      setError('Failed to save your profile. Please try again.');
    } finally {
      setSaving(false);
    }
  });

  return (
    <Card shadow="sm" radius="lg" padding="lg">
      <Stack gap="md">
        <Group justify="space-between" align="center">
          <Title order={2} size="md">Profile Information</Title>
          {!editing && (
            <Button size="xs" variant="light" color="brand" onClick={startEditing}>
              Edit
            </Button>
          )}
        </Group>

        {error && <Alert color="red" title="Error">{error}</Alert>}

        {editing ? (
          <form onSubmit={handleSubmit}>
            <Stack gap="md">
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
                <TextInput
                  label="First Name"
                  placeholder="First name"
                  {...form.getInputProps('first_name')}
                />
                <TextInput
                  label="Last Name"
                  placeholder="Last name"
                  {...form.getInputProps('last_name')}
                />
              </SimpleGrid>

              <TextInput
                label="Email"
                type="email"
                placeholder="you@example.com"
                {...form.getInputProps('email')}
              />

              <ProfileField label="Phone" value={customer.phone} />

              <Group gap="sm" mt="xs">
                <Button type="submit" color="brand" loading={saving}>
                  Save Changes
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  color="navy"
                  onClick={() => setEditing(false)}
                >
                  Cancel
                </Button>
              </Group>
            </Stack>
          </form>
        ) : (
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <ProfileField label="Name" value={`${customer.first_name} ${customer.last_name}`.trim() || 'Not set'} />
            <ProfileField label="Email" value={customer.email || 'Not set'} />
            <ProfileField label="Phone" value={customer.phone} />
            <ProfileField
              label="Saved Addresses"
              value={`${addressCount} ${addressCount === 1 ? 'address' : 'addresses'}`}
            />
          </SimpleGrid>
        )}
      </Stack>
    </Card>
  );
}

function ProfileField({ label, value }: { label: string; value: string }) {
  return (
    <Stack gap={2}>
      <Text size="sm" c="dimmed">{label}</Text>
      <Text fw={500} c="navy.7">{value}</Text>
    </Stack>
  );
}
