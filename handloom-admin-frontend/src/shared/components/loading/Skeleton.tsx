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

// Card skeleton for dashboard widgets
export function CardSkeleton() {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
      <div className="flex items-center justify-between mb-4">
        <Skeleton className="h-4 w-24" />
        <Skeleton variant="circular" width={40} height={40} />
      </div>
      <Skeleton className="h-8 w-32 mb-2" />
      <Skeleton className="h-3 w-20" />
    </div>
  );
}

// Table skeleton for list pages
export function TableSkeleton({ rows = 5, columns = 4 }: { rows?: number; columns?: number }) {
  return (
    <div className="animate-pulse">
      {/* Header */}
      <div className="flex gap-4 p-4 border-b border-gray-200 bg-gray-50">
        {Array.from({ length: columns }).map((_, i) => (
          <Skeleton key={i} className="h-4 flex-1" />
        ))}
      </div>
      {/* Rows */}
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div key={rowIndex} className="flex gap-4 p-4 border-b border-gray-100">
          {Array.from({ length: columns }).map((_, colIndex) => (
            <Skeleton key={colIndex} className="h-4 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}

// Stats card skeleton for dashboard
export function StatsSkeleton({ count = 4 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {Array.from({ length: count }).map((_, i) => (
        <CardSkeleton key={i} />
      ))}
    </div>
  );
}

// Chart skeleton
export function ChartSkeleton({ height = 300 }: { height?: number }) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
      <div className="flex items-center justify-between mb-6">
        <Skeleton className="h-5 w-32" />
        <div className="flex gap-2">
          <Skeleton className="h-8 w-20" variant="rectangular" />
          <Skeleton className="h-8 w-20" variant="rectangular" />
        </div>
      </div>
      <Skeleton variant="rectangular" height={height} className="w-full" />
    </div>
  );
}

// Form skeleton
export function FormSkeleton({ fields = 4 }: { fields?: number }) {
  return (
    <div className="space-y-4 animate-pulse">
      {Array.from({ length: fields }).map((_, i) => (
        <div key={i}>
          <Skeleton className="h-4 w-24 mb-2" />
          <Skeleton variant="rectangular" height={40} className="w-full" />
        </div>
      ))}
      <div className="flex gap-3 pt-4">
        <Skeleton variant="rectangular" height={40} width={100} />
        <Skeleton variant="rectangular" height={40} width={100} />
      </div>
    </div>
  );
}

// List item skeleton
export function ListItemSkeleton() {
  return (
    <div className="flex items-center gap-4 p-4 animate-pulse">
      <Skeleton variant="circular" width={48} height={48} />
      <div className="flex-1">
        <Skeleton className="h-4 w-48 mb-2" />
        <Skeleton className="h-3 w-32" />
      </div>
      <Skeleton variant="rectangular" width={80} height={32} />
    </div>
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

// Page skeleton with table - for list pages
export function TablePageSkeleton({ title, subtitle }: { title?: string; subtitle?: string }) {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          {title ? (
            <>
              <h1 className="page-title">{title}</h1>
              {subtitle && <p className="page-subtitle">{subtitle}</p>}
            </>
          ) : (
            <>
              <Skeleton className="h-8 w-32 mb-2" />
              <Skeleton className="h-4 w-48" />
            </>
          )}
        </div>
        <Skeleton variant="rectangular" width={120} height={40} />
      </div>

      {/* Filters Skeleton */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 animate-pulse">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <Skeleton variant="rectangular" height={40} className="w-full" />
          </div>
          <Skeleton variant="rectangular" width={100} height={40} />
        </div>
      </div>

      {/* Table Skeleton */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <TableSkeleton rows={8} columns={6} />
      </div>
    </div>
  );
}
