'use client';

import { Button, Stack, Text, Title } from '@mantine/core';
import Link from 'next/link';

interface EmptyStateProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  actionLabel?: string;
  actionHref?: string;
  className?: string;
}

export function EmptyState({
  icon,
  title,
  description,
  actionLabel,
  actionHref,
  className,
}: EmptyStateProps) {
  return (
    <Stack align="center" gap="xs" py="xl" ta="center" className={className}>
      {icon}
      <Title order={2} size="md" mt="xs">{title}</Title>
      <Text size="sm" c="dimmed">{description}</Text>
      {actionLabel && actionHref && (
        <Button component={Link} href={actionHref} mt="md" color="brand">
          {actionLabel}
        </Button>
      )}
    </Stack>
  );
}
