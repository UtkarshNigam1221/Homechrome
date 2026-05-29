'use client';

import { Center } from '@mantine/core';

import HCLoader from '@/components/ui/HCLoader';

interface OrderCardSkeletonProps {
  // Kept for backward compatibility — no longer used (HCLoader is a single instance).
  count?: number;
}

export default function OrderCardSkeleton(_props: OrderCardSkeletonProps) {
  return (
    <Center py={80} w="100%">
      <HCLoader size="lg" label="Loading orders" />
    </Center>
  );
}
