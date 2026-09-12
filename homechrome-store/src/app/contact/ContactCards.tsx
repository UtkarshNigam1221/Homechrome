'use client';

import { ArrowUpRightIcon } from '@heroicons/react/24/outline';
import { Anchor, Box, Card, Group, Stack, Text } from '@mantine/core';
import Link from 'next/link';

import { displayFont } from '../fonts';

export function DeskCard({
  tile,
  icon,
  eyebrow,
  title,
  body,
  detail,
  action,
}: {
  tile: { bg: string; fg: string };
  icon: React.ReactNode;
  eyebrow: string;
  title: string;
  body: string;
  detail: string;
  action?: { label: string; href: string; external?: boolean; tone?: 'leaf' };
}) {
  return (
    <Card shadow="sm" radius="lg" padding="lg" withBorder={false} h="100%">
      <Stack gap="sm" h="100%">
        <Box
          w={40}
          h={40}
          c={tile.fg}
          style={{
            background: tile.bg,
            borderRadius: 'var(--mantine-radius-md)',
            display: 'grid',
            placeItems: 'center',
          }}
        >
          {icon}
        </Box>

        <Stack gap={2}>
          <Text fz={10} fw={700} c="navy.5" style={{ letterSpacing: '0.1em' }}>
            {eyebrow.toUpperCase()}
          </Text>
          <Text
            fz="1.125rem"
            fw={600}
            c="navy.9"
            style={{ fontFamily: displayFont.style.fontFamily }}
          >
            {title}
          </Text>
        </Stack>

        <Text fz="xs" c="navy.6" lh={1.6}>
          {body}
        </Text>

        <Text fz="sm" fw={600} c="navy.8" mt="auto" style={{ wordBreak: 'break-word' }}>
          {detail}
        </Text>

        {action && (
          <ActionLink href={action.href} external={action.external}>
            <Group
              gap={6}
              justify="center"
              py={8}
              px={12}
              c={action.tone === 'leaf' ? 'white' : 'navy.8'}
              style={{
                background: action.tone === 'leaf' ? '#42634C' : 'var(--mantine-color-navy-1)',
                borderRadius: 'var(--mantine-radius-md)',
              }}
            >
              <Text fz="xs" fw={600} c={action.tone === 'leaf' ? 'white' : 'navy.8'}>
                {action.label}
              </Text>
              <ArrowUpRightIcon width={13} height={13} />
            </Group>
          </ActionLink>
        )}
      </Stack>
    </Card>
  );
}

/** Internal routes go through Link; wa.me and tel: need a plain anchor. */
export function ActionLink({
  href,
  external,
  children,
}: {
  href: string;
  external?: boolean;
  children: React.ReactNode;
}) {
  if (external) {
    return (
      <Anchor href={href} target="_blank" rel="noopener noreferrer" underline="never">
        {children}
      </Anchor>
    );
  }
  return (
    <Anchor component={Link} href={href} underline="never">
      {children}
    </Anchor>
  );
}

export function ReachTile({
  href,
  tile,
  icon,
  label,
  value,
}: {
  href: string;
  tile: { bg: string; fg: string };
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <Anchor href={href} underline="never">
      <Card shadow="sm" radius="lg" padding="md" withBorder={false} h="100%">
        <Group gap="sm" wrap="nowrap">
          <Box
            w={34}
            h={34}
            c={tile.fg}
            style={{
              background: tile.bg,
              borderRadius: 'var(--mantine-radius-sm)',
              display: 'grid',
              placeItems: 'center',
              flexShrink: 0,
            }}
          >
            {icon}
          </Box>
          <Stack gap={1} style={{ minWidth: 0 }}>
            <Text fz={9} fw={700} c="navy.5" style={{ letterSpacing: '0.1em' }}>
              {label}
            </Text>
            <Text fz="xs" fw={600} c="navy.9" style={{ wordBreak: 'break-word' }}>
              {value}
            </Text>
          </Stack>
        </Group>
      </Card>
    </Anchor>
  );
}
