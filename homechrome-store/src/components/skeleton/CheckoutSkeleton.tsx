'use client';

import { Center, Container } from '@mantine/core';

import HCLoader from '@/components/ui/HCLoader';

export default function CheckoutSkeleton() {
  return (
    <Container size="lg" py="xl">
      <Center py={80} w="100%">
        <HCLoader size="lg" label="Loading checkout" />
      </Center>
    </Container>
  );
}
