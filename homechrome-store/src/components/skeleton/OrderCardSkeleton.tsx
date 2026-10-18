'use client';

import { LoadingBlock } from '@/components/ui/loading-spinner';

export default function OrderCardSkeleton() {
  return <LoadingBlock label="Loading orders" />;
}
