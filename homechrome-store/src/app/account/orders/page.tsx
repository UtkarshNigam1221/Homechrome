'use client';

import { ShoppingBagIcon } from '@heroicons/react/24/outline';
import { Alert, Badge, Card, Group, Stack, Text, Title } from '@mantine/core';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';

import { EmptyState } from '@/components/ui/empty-state';
import OrderCardSkeleton from '@/components/skeleton/OrderCardSkeleton';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDate, formatPrice, statusBadgeColor } from '@/lib/utils';
import { Order } from '@/types';

type OrdersResponse = Order[];

export default function OrdersPage() {
  const {
    data: orders = [],
    isLoading: loading,
    isError,
  } = useQuery({
    queryKey: ['orders', 'list', { limit: 10 }],
    queryFn: async () => {
      const { data } = await api.get<OrdersResponse>(ROUTES.ORDERS.LIST, { limit: 10 });
      return Array.isArray(data) ? data : [];
    },
    refetchOnMount: 'always',
  });
  const error = isError ? 'Failed to load orders. Please try again.' : null;

  if (loading) {
    return <OrderCardSkeleton count={3} />;
  }

  if (error) {
    return <Alert color="red" title="Error">{error}</Alert>;
  }

  if (orders.length === 0) {
    return (
      <Card shadow="sm" radius="lg" padding="lg">
        <EmptyState
          icon={<ShoppingBagIcon strokeWidth={1} width={64} height={64} color="var(--mantine-color-dimmed)" />}
          title="No orders yet"
          description="Looks like you have not placed any orders yet."
          actionLabel="Start Shopping"
          actionHref="/products"
        />
      </Card>
    );
  }

  return (
    <Stack gap="md">
      <Title order={2} size="md">Order History</Title>

      {orders.map((order) => (
        <Card
          key={order.id}
          component={Link}
          href={`/account/orders/${order.id}`}
          shadow="sm"
          radius="lg"
          padding="md"
          withBorder
          style={{ textDecoration: 'none', transition: 'border-color 0.15s' }}
        >
          <Group justify="space-between" align="start" wrap="wrap" gap="md">
            <Stack gap={4}>
              <Group gap="xs" align="center">
                <Text fw={600} c="navy.7">#{order.order_number}</Text>
                <Badge color={statusBadgeColor[order.status]} variant="light" size="sm">
                  {order.status}
                </Badge>
              </Group>
              <Text size="sm" c="dimmed">
                {formatDate(order.created_at)} · {order.item_count}{' '}
                {order.item_count === 1 ? 'item' : 'items'}
              </Text>
            </Stack>
            <Stack gap={2} align="end">
              <Text fw={600} size="lg" c="navy.7">
                {formatPrice(order.total_amount)}
              </Text>
              {order.tracking_number && (
                <Text size="xs" c="dimmed">Tracking: {order.tracking_number}</Text>
              )}
            </Stack>
          </Group>
        </Card>
      ))}
    </Stack>
  );
}
