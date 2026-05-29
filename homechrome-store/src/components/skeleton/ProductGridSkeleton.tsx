'use client';

import { Center } from '@mantine/core';

import HCLoader from '@/components/ui/HCLoader';

interface ProductGridSkeletonProps {
  // Kept for backward compatibility — no longer used.
  count?: number;
}

export default function ProductGridSkeleton(_props: ProductGridSkeletonProps) {
  return (
    <Center py={80} w="100%">
      <HCLoader size="lg" label="Loading products" />
    </Center>
  );
}
