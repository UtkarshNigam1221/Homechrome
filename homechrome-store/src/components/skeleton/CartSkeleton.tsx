'use client';

import { Container } from '@mantine/core';

import { LoadingBlock } from '@/components/ui/loading-spinner';

export default function CartSkeleton() {
  return (
    <Container size="lg" py="xl">
      <LoadingBlock label="Loading cart" />
    </Container>
  );
}
