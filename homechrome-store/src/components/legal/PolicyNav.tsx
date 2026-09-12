'use client';

import {
  ArrowPathIcon,
  BuildingLibraryIcon,
  ChevronRightIcon,
  LockClosedIcon,
  ScaleIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';
import { Anchor, Box, Card, Group, Stack, Text } from '@mantine/core';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import {
  SUPPORT_EMAIL,
  SUPPORT_PHONE,
} from '@/lib/constants';
import { whatsappHref } from '@/lib/whatsapp';

import { displayFont } from '@/app/fonts';

const CHAPTERS = [
  { href: '/refund-policy', label: 'Refund & Replacement', icon: ArrowPathIcon },
  { href: '/shipping-policy', label: 'Shipping & Dispatch', icon: TruckIcon },
  { href: '/terms', label: 'Terms & Conditions', icon: ScaleIcon },
  { href: '/privacy-policy', label: 'Privacy & Data', icon: LockClosedIcon },
  { href: '/contact#grievance', label: 'Grievance Officer', icon: BuildingLibraryIcon },
];

export function PolicyNav() {
  const pathname = usePathname();

  return (
    <Stack gap="md">
      <Card shadow="sm" radius="lg" padding="md" withBorder={false}>
        <Stack gap={2} mb="xs" px={6}>
          <Text fz={10} fw={700} c="navy.5" style={{ letterSpacing: '0.12em' }}>
            POLICY CHAPTERS
          </Text>
          <Text
            fz="1.125rem"
            fw={600}
            c="navy.9"
            style={{ fontFamily: displayFont.style.fontFamily }}
          >
            Legal &amp; Trust Directory
          </Text>
        </Stack>

        <Stack gap={2}>
          {CHAPTERS.map(({ href, label, icon: Icon }) => {
            const active = pathname === href.split('#')[0];
            return (
              <Anchor key={href} component={Link} href={href} underline="never">
                <Group
                  justify="space-between"
                  wrap="nowrap"
                  px={10}
                  py={9}
                  bg={active ? 'brand.5' : undefined}
                  style={{ borderRadius: 'var(--mantine-radius-md)' }}
                >
                  <Group gap={9} wrap="nowrap">
                    <Icon
                      width={17}
                      height={17}
                      color={`var(--mantine-color-${active ? 'white' : 'navy-6'})`}
                    />
                    <Text fz="sm" fw={active ? 600 : 500} c={active ? 'white' : 'navy.8'}>
                      {label}
                    </Text>
                  </Group>
                  <ChevronRightIcon
                    width={14}
                    height={14}
                    color={`var(--mantine-color-${active ? 'white' : 'navy-5'})`}
                  />
                </Group>
              </Anchor>
            );
          })}
        </Stack>
      </Card>

      <Card shadow="sm" radius="lg" padding="md" withBorder={false} bg="navy.1">
        <Stack gap="xs">
          <Text
            fz="1rem"
            fw={600}
            c="navy.9"
            style={{ fontFamily: displayFont.style.fontFamily }}
          >
            Need a hand?
          </Text>
          <Text fz="xs" c="navy.6" lh={1.6}>
            Raising a replacement or unsure whether something counts as a defect? Message us with
            your order ID.
          </Text>

          <Anchor
            href={whatsappHref('Hi, I have a question about a policy')}
            target="_blank"
            rel="noopener noreferrer"
            underline="never"
          >
            <Group gap={8} justify="center" c="white" py={9} style={{ background: 'var(--mantine-color-leaf-5)', borderRadius: 'var(--mantine-radius-md)' }}>
              <WhatsAppIcon size={16} />
              <Text fz="xs" fw={600} c="white">
                WhatsApp {SUPPORT_PHONE}
              </Text>
            </Group>
          </Anchor>

          <Anchor href={`mailto:${SUPPORT_EMAIL}`} underline="never">
            <Box py={8} style={{ borderRadius: 'var(--mantine-radius-md)', background: 'var(--mantine-color-navy-2)' }}>
              <Text fz="xs" fw={600} c="navy.8" ta="center">
                {SUPPORT_EMAIL}
              </Text>
            </Box>
          </Anchor>
        </Stack>
      </Card>
    </Stack>
  );
}
