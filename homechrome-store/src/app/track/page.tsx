'use client';

import {
  ClockIcon,
  LockClosedIcon,
  MagnifyingGlassIcon,
  ShieldCheckIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';
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

import { Breadcrumb } from '@/components/ui/breadcrumb';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import api from '@/lib/api';
import {
  DELIVERY_DAYS,
  DISPATCH_DAYS,
  SUPPORT_PHONE,
  SUPPORT_WHATSAPP,
} from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { formatDateTime as formatDate } from '@/lib/utils';

import { displayFont } from '../fonts';

// Policy-backed only. The design's "7-Day Free Exchange", insured transit and
// OTP handover tiles are absent: the refund policy grants replacements for
// damage alone, and the other two are not things the site can attest to.
const ASSURANCES = [
  { icon: TruckIcon, title: 'Free delivery', body: 'Every order ships free across India' },
  { icon: ClockIcon, title: `Dispatch in ${DISPATCH_DAYS} days`, body: `Then ${DELIVERY_DAYS} days in transit` },
  { icon: ShieldCheckIcon, title: 'Damage replaced', body: 'Send an unboxing video within 48 hours' },
  { icon: LockClosedIcon, title: 'Secure payments', body: 'Encrypted PhonePe checkout' },
];

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
    <Container size="lg" py="xl">
      <Breadcrumb items={[{ label: 'Home', href: '/' }, { label: 'Track Order' }]} />

      <Stack gap="md" mb="xl" maw={640}>
        <Group gap={8} w="fit-content" px={12} py={6} bg="navy.2" style={{ borderRadius: 999 }}>
          <Box w={7} h={7} bg="#42634C" style={{ borderRadius: 999 }} />
          <Text fz={11} fw={700} c="navy.7" style={{ letterSpacing: '0.12em' }}>
            DISPATCH TO DOORSTEP
          </Text>
        </Group>

        <Title
          order={1}
          fz={{ base: '2rem', sm: '2.75rem' }}
          fw={600}
          lh={1.12}
          c="navy.9"
          style={{ fontFamily: displayFont.style.fontFamily }}
        >
          Track Your{' '}
          <Text span inherit c="brand.5" fs="italic">
            Order
          </Text>
        </Title>

        <Text c="navy.6" lh={1.65}>
          Enter the order number from your confirmation and we will show you where the parcel has
          reached.
        </Text>
      </Stack>

      <Card shadow="sm" radius="lg" padding="lg" withBorder={false} mb="lg">
        <form onSubmit={handleSubmit}>
          <Stack gap="sm">
            <Text fz={11} fw={700} c="navy.5" style={{ letterSpacing: '0.1em' }}>
              ORDER NUMBER
            </Text>
            <Group gap="sm" align="flex-start" wrap="nowrap">
              <TextInput
                flex={1}
                size="md"
                value={orderNumber}
                onChange={(e) => setOrderNumber(e.target.value)}
                placeholder="e.g. HC-20260220-XXXX"
                error={error}
              />
              <Button
                type="submit"
                loading={loading}
                color="brand"
                size="md"
                radius="sm"
                leftSection={<MagnifyingGlassIcon width={17} height={17} />}
              >
                Track
              </Button>
            </Group>
            <Text fz="xs" c="navy.6">
              Your order number is in the confirmation message we sent when the order was placed.
            </Text>
          </Stack>
        </form>
      </Card>

      {tracking && (
        <Stack gap="lg">
          <Card shadow="sm" radius="lg" padding="lg">
            <Stack gap="md">
              <Group justify="space-between" align="center" wrap="wrap" gap="sm">
                <Title
                  order={2}
                  fz={{ base: '1.25rem', sm: '1.5rem' }}
                  fw={600}
                  c="navy.9"
                  style={{ fontFamily: displayFont.style.fontFamily }}
                >
                  Order #{tracking.order_number}
                </Title>
                <Group gap={7} px={11} py={5} bg="#C7ECCE" style={{ borderRadius: 999 }}>
                  <Box w={7} h={7} bg="#2E4E37" style={{ borderRadius: 999 }} />
                  <Text fz={11} fw={700} c="#2E4E37" tt="uppercase" style={{ letterSpacing: '0.06em' }}>
                    {tracking.status}
                  </Text>
                </Group>
              </Group>

              {/* The handler returns a non-nil shipment for legacy rows even
                  when every displayable field is blank, so gate on content. */}
              {(tracking.shipment?.courier_name ||
                tracking.shipment?.awb_number ||
                tracking.shipment?.tracking_url) && (
                <Card bg="navy.1" radius="md" padding="md" withBorder={false}>
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
                <Title
                  order={3}
                  fz="1.125rem"
                  fw={600}
                  c="navy.9"
                  style={{ fontFamily: displayFont.style.fontFamily }}
                >
                  Dispatch log
                </Title>
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

          <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="navy.1">
            <Group justify="space-between" align="center" gap="md" wrap="wrap">
              <Stack gap={2}>
                <Text
                  fz="1.125rem"
                  fw={600}
                  c="navy.9"
                  style={{ fontFamily: displayFont.style.fontFamily }}
                >
                  Something not right with this parcel?
                </Text>
                <Text fz="sm" c="navy.6">
                  Message us with your order number and we will pick it up from there.
                </Text>
              </Stack>
              <Anchor
                href={`https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent(`Hi, I need help with order ${tracking.order_number}`)}`}
                target="_blank"
                rel="noopener noreferrer"
                underline="never"
              >
                <Group gap={8} c="white" px={18} py={10} style={{ background: '#42634C', borderRadius: 'var(--mantine-radius-md)' }}>
                  <WhatsAppIcon size={17} />
                  <Text fz="sm" fw={600} c="white">
                    WhatsApp {SUPPORT_PHONE}
                  </Text>
                </Group>
              </Anchor>
            </Group>
          </Card>
        </Stack>
      )}

      <SimpleGrid cols={{ base: 2, md: 4 }} spacing="md" mt="xl">
        {ASSURANCES.map(({ icon: Icon, title, body }) => (
          <Card key={title} shadow="sm" radius="lg" padding="md" withBorder={false}>
            <Stack gap={5}>
              <Icon width={20} height={20} color="var(--mantine-color-brand-6)" />
              <Text fz="sm" fw={600} c="navy.9">
                {title}
              </Text>
              <Text fz={11} c="navy.6" lh={1.5}>
                {body}
              </Text>
            </Stack>
          </Card>
        ))}
      </SimpleGrid>
    </Container>
  );
}
