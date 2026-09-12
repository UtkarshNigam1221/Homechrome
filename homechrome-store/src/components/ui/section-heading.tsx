'use client';

import { ArrowRightIcon } from '@heroicons/react/24/outline';
import { Anchor, Group, Stack, Title } from '@mantine/core';

import { Eyebrow } from '@/components/ui/eyebrow';
import Link from 'next/link';


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
        {eyebrow && <Eyebrow size={12}>{eyebrow}</Eyebrow>}
        <Title
          id={id}
          order={2}
          fz={{ base: '1.75rem', sm: '2.25rem' }}
          fw={600}
          lh={1.15}
          c="navy.9"
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
