'use client';

import { LoadingBlock } from '@/components/ui/loading-spinner';

interface OrderCardSkeletonProps {
  // Kept for backward compatibility — no longer used (HCLoader is a single instance).
  count?: number;
}

export default function OrderCardSkeleton(_props: OrderCardSkeletonProps) {
  return <LoadingBlock label="Loading orders" />;
}
