'use client';

import { Card, Container, Group, SimpleGrid, Skeleton, Stack } from '@mantine/core';

export default function CartSkeleton() {
  return (
    <Container size="xl" py="xl" role="status" aria-label="Loading cart">
      <Skeleton h={32} w={192} mb="xl" />
      <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="xl">
        <Stack gap="md" style={{ gridColumn: 'span 2 / span 2' }}>
          {[1, 2, 3].map((i) => (
            <Card key={i} shadow="sm" radius="lg" padding="md">
              <Group gap="md" wrap="nowrap">
                <Skeleton h={96} w={96} radius="md" />
                <Stack flex={1} gap="xs">
                  <Skeleton h={20} w="75%" />
                  <Skeleton h={16} w="33%" />
                  <Group justify="space-between" mt="xs">
                    <Skeleton h={32} w={96} />
                    <Skeleton h={20} w={80} />
                  </Group>
                </Stack>
              </Group>
            </Card>
          ))}
        </Stack>
        <Card shadow="sm" radius="lg" padding="md">
          <Stack gap="md">
            <Skeleton h={20} w={128} />
            <Skeleton h={16} />
            <Skeleton h={16} />
            <Skeleton h={16} w="66%" />
            <Skeleton h={20} mt="sm" />
            <Skeleton h={40} />
          </Stack>
        </Card>
      </SimpleGrid>
    </Container>
  );
}
