'use client';

import { Anchor, Group, Title } from '@mantine/core';
import Link from 'next/link';

interface SectionHeadingProps {
  title: string;
  id?: string;
  linkHref?: string;
  linkLabel?: string;
}

export function SectionHeading({ title, id, linkHref, linkLabel }: SectionHeadingProps) {
  return (
    <Group justify="space-between" align="baseline" mb="xl" gap="md" wrap="wrap">
      <Title id={id} order={2} fz={{ base: '1.75rem', sm: '2.125rem' }} fw={600} lh={1.15} c="navy.9">
        {title}
      </Title>

      {linkHref && linkLabel && (
        <Anchor component={Link} href={linkHref} underline="hover" c="navy.8" fw={500} size="sm">
          {linkLabel}
        </Anchor>
      )}
    </Group>
  );
}
