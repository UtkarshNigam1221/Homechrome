'use client';

import { MapPinIcon, ShoppingBagIcon, TruckIcon } from '@heroicons/react/24/outline';
import {
  Anchor,
  Card,
  Group,
  SimpleGrid,
  Stack,
  Text,
  ThemeIcon,
  Title,
} from '@mantine/core';
import Link from 'next/link';

import { formatPrice } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';

const navCards = [
  {
    href: '/account/orders',
    icon: ShoppingBagIcon,
    title: 'My Orders',
    description: 'View your order history and track shipments',
  },
  {
    href: '/account/addresses',
    icon: MapPinIcon,
    title: 'Addresses',
    description: 'Manage your saved delivery addresses',
  },
  {
    href: '/track',
    icon: TruckIcon,
    title: 'Track Order',
    description: 'Track your order with an order number',
  },
];

export default function AccountPage() {
  const customer = useAuthStore((s) => s.customer);

  if (!customer) return null;

  const addressCount = customer.addresses?.length ?? 0;

  return (
    <Stack gap="lg">
      <Card shadow="sm" radius="lg" padding="lg">
        <Stack gap="md">
          <Title order={2} size="md">Profile Information</Title>
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <ProfileField label="Name" value={`${customer.first_name} ${customer.last_name}`} />
            <ProfileField label="Email" value={customer.email || 'Not set'} />
            <ProfileField label="Phone" value={customer.phone} />
            <ProfileField
              label="Saved Addresses"
              value={`${addressCount} ${addressCount === 1 ? 'address' : 'addresses'}`}
            />
          </SimpleGrid>
        </Stack>
      </Card>

      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
        {navCards.map((card) => (
          <Anchor
            key={card.href}
            component={Link}
            href={card.href}
            underline="never"
          >
            <Card shadow="sm" radius="lg" padding="lg" withBorder>
              <Stack gap="xs">
                <ThemeIcon size={40} radius="md" variant="light" color="brand">
                  <card.icon width={20} height={20} />
                </ThemeIcon>
                <Text fw={600} c="navy.7">{card.title}</Text>
                <Text size="sm" c="dimmed">{card.description}</Text>
              </Stack>
            </Card>
          </Anchor>
        ))}
      </SimpleGrid>

      <Card shadow="sm" radius="lg" padding="lg">
        <Stack gap="md">
          <Title order={2} size="md">Account Summary</Title>
          <SimpleGrid cols={2} spacing="md">
            <SummaryStat
              value={String(customer.total_orders)}
              label="Total Orders"
            />
            <SummaryStat
              value={formatPrice(customer.total_spent)}
              label="Total Spent"
            />
          </SimpleGrid>
        </Stack>
      </Card>
    </Stack>
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

function SummaryStat({ value, label }: { value: string; label: string }) {
  return (
    <Group
      bg="gray.0"
      p="md"
      justify="center"
      gap="xs"
      style={{ borderRadius: 'var(--mantine-radius-md)', flexDirection: 'column' }}
    >
      <Text size="xl" fw={700} c="brand.5">{value}</Text>
      <Text size="sm" c="dimmed">{label}</Text>
    </Group>
  );
}
