'use client';

import HCLoader from '@/components/ui/HCLoader';

interface ProductGridSkeletonProps {
  // Kept for backward compatibility — no longer used.
  count?: number;
}

export default function ProductGridSkeleton(_props: ProductGridSkeletonProps) {
  return (
    <div className="flex w-full items-center justify-center py-20">
      <HCLoader size="lg" label="Loading products" />
    </div>
  );
}
