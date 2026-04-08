import Skeleton from './Skeleton';

interface ProductGridSkeletonProps {
  count?: number;
}

export default function ProductGridSkeleton({ count = 8 }: ProductGridSkeletonProps) {
  return (
    <div role="status" aria-label="Loading products" className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="overflow-hidden rounded-xl bg-white shadow-sm">
          <Skeleton variant="rectangular" className="aspect-square w-full rounded-none" />
          <div className="p-4 space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <div className="flex items-baseline gap-2 pt-1">
              <Skeleton className="h-5 w-20" />
              <Skeleton className="h-4 w-14" />
            </div>
            <Skeleton variant="rectangular" className="mt-3 h-9 w-full" />
          </div>
        </div>
      ))}
    </div>
  );
}
