import { clsx } from 'clsx';

export interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular';
  width?: string | number;
  height?: string | number;
}

export function Skeleton({ className, variant = 'text', width, height }: SkeletonProps) {
  const variants = {
    text: 'rounded',
    circular: 'rounded-full',
    rectangular: 'rounded-lg',
  };

  return (
    <div
      className={clsx(
        'bg-gray-200 animate-pulse',
        variants[variant],
        variant === 'text' && 'h-4',
        className
      )}
      style={{ width, height }}
    />
  );
}

// Dashboard skeleton - complete dashboard loading state
export function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="page-header">
        <Skeleton className="h-8 w-40 mb-2" />
        <Skeleton className="h-4 w-80" />
      </div>

      {/* Stats Skeleton */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
            <div className="flex items-center justify-between mb-4">
              <Skeleton className="h-4 w-24" />
              <Skeleton variant="circular" width={40} height={40} />
            </div>
            <Skeleton className="h-8 w-32 mb-2" />
            <Skeleton className="h-3 w-20" />
          </div>
        ))}
      </div>

      {/* Charts Skeleton */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
            <div className="flex items-center justify-between mb-6">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-20" />
            </div>
            <Skeleton variant="rectangular" height={280} className="w-full" />
          </div>
        ))}
      </div>

      {/* Bottom Row Skeleton */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {Array.from({ length: 2 }).map((_, i) => (
          <div
            key={i}
            className="bg-white rounded-lg border border-gray-200 overflow-hidden animate-pulse"
          >
            <div className="p-6 border-b border-gray-200">
              <Skeleton className="h-5 w-32 mb-2" />
              <Skeleton className="h-4 w-48" />
            </div>
            <div className="divide-y divide-gray-200">
              {Array.from({ length: 5 }).map((_, j) => (
                <div key={j} className="p-4 flex items-center justify-between">
                  <div>
                    <Skeleton className="h-4 w-24 mb-2" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <div className="text-right">
                    <Skeleton className="h-4 w-16 mb-2" />
                    <Skeleton className="h-5 w-20" variant="rectangular" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
