import Skeleton from './Skeleton';

interface OrderCardSkeletonProps {
  count?: number;
}

export default function OrderCardSkeleton({ count = 3 }: OrderCardSkeletonProps) {
  return (
    <div role="status" aria-label="Loading orders" className="space-y-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-lg border border-border bg-white p-4 sm:p-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Skeleton className="h-5 w-32" />
                <Skeleton variant="rectangular" className="h-5 w-16 rounded-full" />
              </div>
              <Skeleton className="h-4 w-48" />
            </div>
            <div className="text-right space-y-1">
              <Skeleton className="h-6 w-24 ml-auto" />
              <Skeleton className="h-3 w-32 ml-auto" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
