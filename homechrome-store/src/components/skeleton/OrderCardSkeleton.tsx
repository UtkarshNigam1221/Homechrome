'use client';

import HCLoader from '@/components/ui/HCLoader';

interface OrderCardSkeletonProps {
  // Kept for backward compatibility — no longer used (HCLoader is a single instance).
  count?: number;
}

export default function OrderCardSkeleton(_props: OrderCardSkeletonProps) {
  return (
    <div className="flex w-full items-center justify-center py-20">
      <HCLoader size="lg" label="Loading orders" />
    </div>
  );
}
