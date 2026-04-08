import Skeleton from '@/components/skeleton/Skeleton';

export default function OrderDetailLoading() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-6 w-44" />
          <Skeleton className="h-4 w-40" />
        </div>
        <Skeleton variant="rectangular" className="h-7 w-24 self-start rounded-full" />
      </div>
      <div className="rounded-lg border border-border bg-white p-6">
        <Skeleton className="h-4 w-24 mb-4" />
        <div className="flex items-center justify-between">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="flex flex-1 items-center">
              <div className="flex flex-col items-center">
                <Skeleton variant="circular" className="h-8 w-8" />
                <Skeleton className="mt-1 h-3 w-12" />
              </div>
              {i < 5 && <Skeleton className="mx-1 h-0.5 flex-1" />}
            </div>
          ))}
        </div>
      </div>
      <div className="rounded-lg border border-border bg-white p-6 space-y-4">
        <Skeleton className="h-4 w-12" />
        {[1, 2].map((i) => (
          <div key={i} className="flex gap-4 py-4">
            <Skeleton variant="rectangular" className="h-20 w-20 flex-shrink-0 rounded-md" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-5 w-48" />
              <Skeleton className="h-4 w-24" />
              <div className="flex items-center justify-between pt-1">
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-5 w-20" />
              </div>
            </div>
          </div>
        ))}
      </div>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <div className="rounded-lg border border-border bg-white p-6 space-y-3">
          <Skeleton className="h-4 w-32" />
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex justify-between">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-16" />
            </div>
          ))}
        </div>
        <div className="rounded-lg border border-border bg-white p-6 space-y-2">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-4 w-56" />
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-32" />
        </div>
      </div>
    </div>
  );
}
