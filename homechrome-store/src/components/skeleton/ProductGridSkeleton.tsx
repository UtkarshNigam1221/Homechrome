'use client';

import { LoadingBlock } from '@/components/ui/loading-spinner';

interface ProductGridSkeletonProps {
  // Kept for backward compatibility — no longer used.
  count?: number;
}

export default function ProductGridSkeleton(_props: ProductGridSkeletonProps) {
  return <LoadingBlock label="Loading products" />;
}
