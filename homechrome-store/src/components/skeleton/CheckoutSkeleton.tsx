import Skeleton from './Skeleton';

export default function CheckoutSkeleton() {
  return (
    <div role="status" aria-label="Loading checkout" className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <Skeleton className="h-8 w-40 mb-2" />
      <Skeleton className="h-4 w-64 mb-8" />

      {/* Progress steps */}
      <div className="mb-8 flex items-center gap-2">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center gap-2">
            {i > 1 && <Skeleton className="h-px w-6 sm:w-12" />}
            <Skeleton variant="rectangular" className="h-8 w-20 rounded-full" />
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <div className="rounded-lg border border-border bg-white p-6 space-y-4">
            <Skeleton className="h-6 w-40" />
            {[1, 2].map((i) => (
              <div key={i} className="rounded-lg border border-border p-4 space-y-2">
                <Skeleton className="h-5 w-48" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-2/3" />
              </div>
            ))}
            <Skeleton variant="rectangular" className="h-10 w-44 mt-4" />
          </div>
        </div>
        <div>
          <div className="rounded-lg border border-border bg-white p-6 space-y-3">
            <Skeleton className="h-5 w-32" />
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex justify-between">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-16" />
              </div>
            ))}
            <div className="border-t border-border pt-3">
              <div className="flex justify-between">
                <Skeleton className="h-5 w-16" />
                <Skeleton className="h-5 w-20" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
