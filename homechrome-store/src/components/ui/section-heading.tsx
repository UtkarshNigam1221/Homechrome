'use client';

import { ArrowRightIcon } from '@heroicons/react/24/outline';
import { Anchor, Group, Stack, Text, Title } from '@mantine/core';
import Link from 'next/link';

import { displayFont } from '@/app/fonts';

interface SectionHeadingProps {
  /** Small letterspaced label above the title. */
  eyebrow?: string;
  title: string;
  id?: string;
  linkHref?: string;
  linkLabel?: string;
}

export function SectionHeading({
  eyebrow,
  title,
  id,
  linkHref,
  linkLabel,
}: SectionHeadingProps) {
  return (
    <Group justify="space-between" align="flex-end" mb="xl" gap="md" wrap="wrap">
      <Stack gap={6}>
        {eyebrow && (
          <Text fz={12} fw={700} c="brand.5" style={{ letterSpacing: '0.14em' }}>
            {eyebrow.toUpperCase()}
          </Text>
        )}
        <Title
          id={id}
          order={2}
          fz={{ base: '1.75rem', sm: '2.25rem' }}
          fw={600}
          lh={1.15}
          c="navy.9"
          style={{ fontFamily: displayFont.style.fontFamily }}
        >
          {title}
        </Title>
      </Stack>

      {linkHref && linkLabel && (
        <Anchor component={Link} href={linkHref} underline="never" c="navy.9" fw={600} size="sm">
          <Group gap={6} wrap="nowrap" align="center">
            {linkLabel}
            <ArrowRightIcon width={16} height={16} />
          </Group>
        </Anchor>
      )}
    </Group>
  );
}
