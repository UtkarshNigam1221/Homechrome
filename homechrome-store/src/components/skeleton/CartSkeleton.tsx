import Skeleton from './Skeleton';

export default function CartSkeleton() {
  return (
    <div role="status" aria-label="Loading cart" className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <Skeleton className="h-8 w-48 mb-8" />
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex gap-4 rounded-lg border border-border bg-white p-4">
              <Skeleton variant="rectangular" className="h-24 w-24 flex-shrink-0" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-4 w-1/3" />
                <div className="flex items-center justify-between pt-2">
                  <Skeleton variant="rectangular" className="h-8 w-24" />
                  <Skeleton className="h-5 w-20" />
                </div>
              </div>
            </div>
          ))}
        </div>
        <div>
          <div className="rounded-lg border border-border bg-white p-6 space-y-4">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
            <div className="border-t border-border pt-4">
              <Skeleton className="h-5 w-full" />
            </div>
            <Skeleton variant="rectangular" className="h-10 w-full" />
          </div>
        </div>
      </div>
    </div>
  );
}
