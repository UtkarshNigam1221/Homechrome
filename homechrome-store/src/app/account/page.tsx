'use client';

import { MapPinIcon, ShoppingBagIcon, TruckIcon } from '@heroicons/react/24/outline';
import { Anchor, Card, SimpleGrid, Stack, Text, ThemeIcon } from '@mantine/core';
import Link from 'next/link';

import { useAuthStore } from '@/stores/auth';

import { ProfileCard } from './ProfileCard';

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

  return (
    <Stack gap="lg">
      <ProfileCard customer={customer} />

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
    </Stack>
  );
}
