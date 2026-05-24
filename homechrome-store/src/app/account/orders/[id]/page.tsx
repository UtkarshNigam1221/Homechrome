'use client';

import { CheckIcon } from '@heroicons/react/24/outline';
import {
  Alert,
  Anchor,
  Badge,
  Box,
  Button,
  Card,
  Center,
  Divider,
  Group,
  Skeleton,
  SimpleGrid,
  Stack,
  Text,
  ThemeIcon,
  Title,
} from '@mantine/core';
import { modals } from '@mantine/modals';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { AssetImage } from '@/components/ui/asset-image';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDateTime as formatDate, formatPrice, statusBadgeColor } from '@/lib/utils';
import { Order, OrderStatus } from '@/types';

const statusTimeline: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
];

function getStatusIndex(status: OrderStatus): number {
  const idx = statusTimeline.indexOf(status);
  return idx === -1 ? -1 : idx;
}

export default function OrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;
  const queryClient = useQueryClient();

  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  const {
    data: order,
    isLoading: loading,
    isError,
  } = useQuery({
    queryKey: ['orders', orderId],
    queryFn: async () => {
      const { data } = await api.get<Order>(ROUTES.ORDERS.DETAIL(orderId));
      return data;
    },
    enabled: !!orderId,
    refetchOnMount: 'always',
  });

  useEffect(() => {
    if (isError) router.replace('/account/orders');
  }, [isError, router]);

  const handleCancel = async () => {
    if (!order) return;
    setCancelling(true);
    setCancelError(null);
    try {
      await api.post(ROUTES.ORDERS.CANCEL(orderId));
      await queryClient.invalidateQueries({ queryKey: ['orders', orderId] });
    } catch {
      setCancelError('Failed to cancel order. Please try again or contact support.');
    } finally {
      setCancelling(false);
    }
  };

  const openCancelModal = () =>
    modals.openConfirmModal({
      title: 'Cancel Order',
      children: (
        <Text size="sm">
          Are you sure you want to cancel this order? This action cannot be undone.
        </Text>
      ),
      labels: { confirm: 'Cancel Order', cancel: 'Keep Order' },
      confirmProps: { color: 'red' },
      onConfirm: handleCancel,
    });

  if (loading) {
    return (
      <Stack gap="lg" role="status" aria-label="Loading order details">
        <Group justify="space-between" align="start">
          <Stack gap="xs">
            <Skeleton height={16} width={112} />
            <Skeleton height={24} width={176} />
            <Skeleton height={16} width={160} />
          </Stack>
          <Skeleton height={28} width={96} radius="xl" />
        </Group>
        <Card><Group justify="space-between">{Array.from({ length: 5 }).map((_, i) => (
          <Stack key={i} align="center" gap={4}><Skeleton height={32} width={32} circle /><Skeleton height={12} width={48} /></Stack>
        ))}</Group></Card>
      </Stack>
    );
  }

  if (!order) return null;

  const canCancel = order.status === 'PENDING' || order.status === 'CONFIRMED';
  const currentStatusIdx = getStatusIndex(order.status);
  const isCancelled = ['CANCELLED', 'RETURNED', 'REFUNDED'].includes(order.status);

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="start" wrap="wrap" gap="md">
        <Stack gap={2}>
          <Anchor component={Link} href="/account/orders" size="sm" c="brand">
            ← Back to Orders
          </Anchor>
          <Title order={2} size="md">Order #{order.order_number}</Title>
          <Text size="sm" c="dimmed">Placed on {formatDate(order.created_at)}</Text>
        </Stack>
        <Badge color={statusBadgeColor[order.status]} variant="light" size="lg" radius="xl">
          {order.status}
        </Badge>
      </Group>

      {cancelError && <Alert color="red" title="Error">{cancelError}</Alert>}

      {!isCancelled && (
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Title order={3} size="sm">Order Status</Title>
            <Group justify="space-between" gap={0} wrap="nowrap">
              {statusTimeline.map((status, idx) => (
                <Group key={status} flex={1} wrap="nowrap" gap={4}>
                  <Stack align="center" gap={4} flex="none">
                    <ThemeIcon
                      size={32}
                      radius="xl"
                      color={idx <= currentStatusIdx ? 'brand' : 'gray'}
                      variant={idx <= currentStatusIdx ? 'filled' : 'light'}
                    >
                      {idx < currentStatusIdx ? (
                        <CheckIcon width={16} height={16} strokeWidth={2.5} />
                      ) : (
                        <Text size="xs" fw={700}>{idx + 1}</Text>
                      )}
                    </ThemeIcon>
                    <Text size="xs" c="dimmed" ta="center">{status}</Text>
                  </Stack>
                  {idx < statusTimeline.length - 1 && (
                    <Box
                      flex={1}
                      h={2}
                      bg={idx < currentStatusIdx ? 'brand.5' : 'gray.3'}
                    />
                  )}
                </Group>
              ))}
            </Group>
          </Stack>
        </Card>
      )}

      {order.tracking_number && (
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Title order={3} size="sm">Tracking Information</Title>
            <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
              {order.shipping_carrier && (
                <Stack gap={2}>
                  <Text size="xs" c="dimmed">Courier</Text>
                  <Text fw={500} c="navy.7">{order.shipping_carrier}</Text>
                </Stack>
              )}
              <Stack gap={2}>
                <Text size="xs" c="dimmed">AWB Number</Text>
                <Text fw={500} c="navy.7">{order.tracking_number}</Text>
              </Stack>
              {order.tracking_url && (
                <Stack gap={2}>
                  <Text size="xs" c="dimmed">Track Shipment</Text>
                  <Anchor href={order.tracking_url} target="_blank" rel="noopener noreferrer" size="sm" c="brand">
                    Track on courier website
                  </Anchor>
                </Stack>
              )}
            </SimpleGrid>
            {order.shipped_at && (
              <Text size="xs" c="dimmed">Shipped on {formatDate(order.shipped_at)}</Text>
            )}
            {order.delivered_at && (
              <Text size="xs" c="dimmed">Delivered on {formatDate(order.delivered_at)}</Text>
            )}
          </Stack>
        </Card>
      )}

      <Card shadow="sm" radius="lg" padding="md">
        <Stack gap="md">
          <Title order={3} size="sm">Items</Title>
          <Stack gap="md" style={{ divider: 'true' }}>
            {order.items.map((item, idx) => (
              <Box key={item.id}>
                {idx > 0 && <Divider mb="md" />}
                <Group gap="md" wrap="nowrap">
                  <Box
                    pos="relative"
                    w={80}
                    h={80}
                    flex="none"
                    style={{
                      overflow: 'hidden',
                      borderRadius: 'var(--mantine-radius-md)',
                      background: 'var(--mantine-color-gray-1)',
                    }}
                  >
                    {item.product_image ? (
                      <AssetImage
                        src={item.product_image}
                        alt={item.product_name}
                        sizes="80px"
                        width={80}
                        height={80}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      />
                    ) : (
                      <Center h="100%">
                        <Text size="xs" c="dimmed">No image</Text>
                      </Center>
                    )}
                  </Box>
                  <Stack flex={1} gap="xs" justify="space-between">
                    <Stack gap={2}>
                      <Text fw={500} c="navy.7">{item.product_name}</Text>
                      <Text size="sm" c="dimmed">SKU: {item.product_sku}</Text>
                    </Stack>
                    <Group justify="space-between">
                      <Text size="sm" c="dimmed">Qty: {item.quantity}</Text>
                      <Text fw={500} c="navy.7">{formatPrice(item.total_price)}</Text>
                    </Group>
                  </Stack>
                </Group>
              </Box>
            ))}
          </Stack>
        </Stack>
      </Card>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="lg">
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Title order={3} size="sm">Payment Summary</Title>
            <Stack gap="xs">
              <Group justify="space-between"><Text size="sm" c="dimmed">Subtotal</Text><Text size="sm">{formatPrice(order.subtotal)}</Text></Group>
              {order.discount_amount > 0 && (
                <Group justify="space-between"><Text size="sm" c="dimmed">Discount</Text><Text size="sm" c="teal.7">-{formatPrice(order.discount_amount)}</Text></Group>
              )}
              <Group justify="space-between"><Text size="sm" c="dimmed">Shipping</Text><Text size="sm">{order.shipping_amount === 0 ? 'FREE' : formatPrice(order.shipping_amount)}</Text></Group>
              {order.tax_amount > 0 && (
                <Group justify="space-between"><Text size="sm" c="dimmed">Tax</Text><Text size="sm">{formatPrice(order.tax_amount)}</Text></Group>
              )}
              <Divider />
              <Group justify="space-between"><Text fw={600}>Total</Text><Text fw={600}>{formatPrice(order.total_amount)}</Text></Group>
            </Stack>
          </Stack>
        </Card>

        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Title order={3} size="sm">Shipping Address</Title>
            <Stack gap={2}>
              <Text fw={500} c="navy.7">
                {order.shipping_address.first_name} {order.shipping_address.last_name}
              </Text>
              <Text size="sm" c="dimmed">{order.shipping_address.address_line1}</Text>
              {order.shipping_address.address_line2 && (
                <Text size="sm" c="dimmed">{order.shipping_address.address_line2}</Text>
              )}
              <Text size="sm" c="dimmed">
                {order.shipping_address.city}, {order.shipping_address.state} -{' '}
                {order.shipping_address.postal_code}
              </Text>
              <Text size="sm" c="dimmed">Phone: {order.shipping_address.phone}</Text>
            </Stack>
          </Stack>
        </Card>
      </SimpleGrid>

      {canCancel && (
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Title order={3} size="sm">Cancel Order</Title>
            <Text size="sm" c="dimmed">
              You can cancel this order since it has not been processed yet.
            </Text>
            <Button color="red" loading={cancelling} onClick={openCancelModal} w="fit-content">
              Cancel Order
            </Button>
          </Stack>
        </Card>
      )}
    </Stack>
  );
}
