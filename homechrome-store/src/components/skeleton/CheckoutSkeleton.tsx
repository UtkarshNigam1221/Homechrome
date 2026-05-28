'use client';

import { Container } from '@mantine/core';

import HCLoader from '@/components/ui/HCLoader';

export default function CheckoutSkeleton() {
  return (
    <Container size="lg" py="xl">
      <div className="flex w-full items-center justify-center py-20">
        <HCLoader size="lg" label="Loading checkout" />
      </div>
    </Container>
  );
}
