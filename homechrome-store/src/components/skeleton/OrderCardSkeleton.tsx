'use client';

import { Card, Group, Skeleton, Stack } from '@mantine/core';

interface OrderCardSkeletonProps {
  count?: number;
}

export default function OrderCardSkeleton({ count = 3 }: OrderCardSkeletonProps) {
  return (
    <Stack gap="md" role="status" aria-label="Loading orders">
      {Array.from({ length: count }).map((_, i) => (
        <Card key={i} shadow="sm" radius="lg" padding="md">
          <Group justify="space-between" align="start" wrap="wrap" gap="md">
            <Stack gap="xs">
              <Group gap="xs">
                <Skeleton h={20} w={128} />
                <Skeleton h={20} w={64} radius="xl" />
              </Group>
              <Skeleton h={16} w={192} />
            </Stack>
            <Stack gap="xs" align="end">
              <Skeleton h={24} w={96} />
              <Skeleton h={12} w={128} />
            </Stack>
          </Group>
        </Card>
      ))}
    </Stack>
  );
}
