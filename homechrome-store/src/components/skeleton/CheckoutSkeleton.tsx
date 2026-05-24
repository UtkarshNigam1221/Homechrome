'use client';

import { Card, Container, Group, SimpleGrid, Skeleton, Stack } from '@mantine/core';

export default function CheckoutSkeleton() {
  return (
    <Container size="lg" py="lg" role="status" aria-label="Loading checkout">
      <Skeleton h={32} w={160} mb="xs" />
      <Skeleton h={16} w={256} mb="xl" />

      <Group gap="sm" mb="xl">
        {[1, 2, 3].map((i) => (
          <Group key={i} gap="sm" wrap="nowrap">
            {i > 1 && <Skeleton h={1} w={32} />}
            <Skeleton h={32} w={80} radius="xl" />
          </Group>
        ))}
      </Group>

      <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="xl">
        <Card shadow="sm" radius="lg" padding="md" style={{ gridColumn: 'span 2 / span 2' }}>
          <Stack gap="md">
            <Skeleton h={24} w={160} />
            {[1, 2].map((i) => (
              <Stack key={i} gap="xs" p="md" style={{ border: '1px solid var(--mantine-color-default-border)', borderRadius: 'var(--mantine-radius-md)' }}>
                <Skeleton h={20} w={192} />
                <Skeleton h={16} />
                <Skeleton h={16} w="66%" />
              </Stack>
            ))}
            <Skeleton h={40} w={176} mt="md" />
          </Stack>
        </Card>
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="sm">
            <Skeleton h={20} w={128} />
            {[1, 2, 3].map((i) => (
              <Group key={i} justify="space-between">
                <Skeleton h={16} w={96} />
                <Skeleton h={16} w={64} />
              </Group>
            ))}
            <Group justify="space-between" mt="sm">
              <Skeleton h={20} w={64} />
              <Skeleton h={20} w={80} />
            </Group>
          </Stack>
        </Card>
      </SimpleGrid>
    </Container>
  );
}
