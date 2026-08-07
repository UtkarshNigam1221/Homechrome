'use client';

import {
  Anchor,
  Box,
  Button,
  Card,
  Container,
  Group,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { useState } from 'react';

import { PageHeader } from '@/components/ui/page-header';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDateTime as formatDate } from '@/lib/utils';

interface StatusHistoryEntry {
  status: string;
  timestamp: string;
  note?: string;
}

interface ShipmentInfo {
  awb_number?: string;
  courier_name?: string;
  tracking_url?: string;
  status: string;
  estimated_delivery?: string;
}

interface TrackingResult {
  order_number: string;
  status: string;
  status_history: StatusHistoryEntry[];
  shipment?: ShipmentInfo;
}

export default function TrackOrderPage() {
  const [orderNumber, setOrderNumber] = useState('');
  const [tracking, setTracking] = useState<TrackingResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = orderNumber.trim();
    if (!trimmed) return;

    setLoading(true);
    setError(null);
    setTracking(null);

    try {
      const { data } = await api.get<TrackingResult>(ROUTES.TRACK(trimmed));
      setTracking(data);
    } catch {
      setError('Order not found. Please check the order number and try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size="md" py="xl">
      <PageHeader
        title="Track Your Order"
        description="Enter your order number to check the delivery status."
      />

      <form onSubmit={handleSubmit}>
        <Group gap="sm" mb="xl" align="flex-start">
          <TextInput
            flex={1}
            value={orderNumber}
            onChange={(e) => setOrderNumber(e.target.value)}
            placeholder="Enter your order number (e.g., HC-20260220-XXXX)"
            error={error}
          />
          <Button type="submit" loading={loading} color="brand">
            Track
          </Button>
        </Group>
      </form>

      {tracking && (
        <Stack gap="lg">
          <Card shadow="sm" radius="lg" padding="md">
            <Stack gap="md">
              <Title order={2} size="md">Order #{tracking.order_number}</Title>
              <Text size="sm" c="dimmed">
                Current Status:{' '}
                <Text span fw={500} c="navy.7">{tracking.status}</Text>
              </Text>

              {/* The handler returns a non-nil shipment for legacy rows even
                  when every displayable field is blank, so gate on content. */}
              {(tracking.shipment?.courier_name ||
                tracking.shipment?.awb_number ||
                tracking.shipment?.tracking_url) && (
                <Card bg="gray.0" radius="md" padding="md" withBorder={false}>
                  <Stack gap="xs">
                    <Title order={3} size="sm">Shipment Details</Title>
                    <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
                      {tracking.shipment.courier_name && (
                        <Stack gap={0}>
                          <Text size="xs" c="dimmed">Courier</Text>
                          <Text size="sm" fw={500} c="navy.7">{tracking.shipment.courier_name}</Text>
                        </Stack>
                      )}
                      {tracking.shipment.awb_number && (
                        <Stack gap={0}>
                          <Text size="xs" c="dimmed">AWB</Text>
                          <Text size="sm" fw={500} c="navy.7">{tracking.shipment.awb_number}</Text>
                        </Stack>
                      )}
                      {tracking.shipment.estimated_delivery && (
                        <Stack gap={0}>
                          <Text size="xs" c="dimmed">Est. Delivery</Text>
                          <Text size="sm" fw={500} c="navy.7">
                            {tracking.shipment.estimated_delivery}
                          </Text>
                        </Stack>
                      )}
                    </SimpleGrid>
                    {tracking.shipment.tracking_url && (
                      <Anchor
                        href={tracking.shipment.tracking_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        size="sm"
                        c="brand"
                      >
                        Track on courier website →
                      </Anchor>
                    )}
                  </Stack>
                </Card>
              )}
            </Stack>
          </Card>

          {tracking.status_history && tracking.status_history.length > 0 && (
            <Card shadow="sm" radius="lg" padding="md">
              <Stack gap="md">
                <Title order={3} size="sm">Order Timeline</Title>
                <Box pos="relative" pl={40}>
                  <Box
                    pos="absolute"
                    left={12}
                    top={8}
                    bottom={8}
                    w={2}
                    bg="gray.3"
                  />
                  <Stack gap="lg">
                    {tracking.status_history.map((entry, idx) => (
                      <Group key={idx} pos="relative" align="start" wrap="nowrap" gap="md">
                        <Box
                          pos="absolute"
                          left={-32}
                          top={4}
                          w={20}
                          h={20}
                          bg={idx === 0 ? 'brand.5' : 'white'}
                          style={{
                            border: `2px solid var(--mantine-color-${idx === 0 ? 'brand-5' : 'default-border'})`,
                            borderRadius: '50%',
                          }}
                        />
                        <Stack gap={2} flex={1}>
                          <Text fw={500} c="navy.7">{entry.status}</Text>
                          <Text size="xs" c="dimmed">{formatDate(entry.timestamp)}</Text>
                          {entry.note && (
                            <Text size="sm" c="dimmed">{entry.note}</Text>
                          )}
                        </Stack>
                      </Group>
                    ))}
                  </Stack>
                </Box>
              </Stack>
            </Card>
          )}
        </Stack>
      )}
    </Container>
  );
}
