'use client';

import { Card, Group, SimpleGrid, Skeleton, Stack } from '@mantine/core';

interface ProductGridSkeletonProps {
  count?: number;
}

export default function ProductGridSkeleton({ count = 8 }: ProductGridSkeletonProps) {
  return (
    <SimpleGrid
      role="status"
      aria-label="Loading products"
      cols={{ base: 2, sm: 3, lg: 4 }}
      spacing="md"
    >
      {Array.from({ length: count }).map((_, i) => (
        <Card key={i} shadow="sm" radius="lg" padding={0}>
          <Skeleton h={200} radius={0} />
          <Stack p="md" gap="xs">
            <Skeleton h={16} w="75%" />
            <Skeleton h={16} w="50%" />
            <Group align="baseline" gap="xs">
              <Skeleton h={20} w={80} />
              <Skeleton h={16} w={56} />
            </Group>
            <Skeleton h={36} mt="xs" />
          </Stack>
        </Card>
      ))}
    </SimpleGrid>
  );
}
