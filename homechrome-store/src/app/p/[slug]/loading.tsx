import Skeleton from '@/components/skeleton/Skeleton';

export default function ProductDetailLoading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <Skeleton className="h-4 w-48 mb-6" />
      <div className="grid grid-cols-1 gap-10 lg:grid-cols-2">
        <div>
          <Skeleton variant="rectangular" className="aspect-square w-full rounded-xl" />
          <div className="mt-4 flex gap-3">
            {[1, 2, 3, 4].map((i) => (
              <Skeleton key={i} variant="rectangular" className="h-20 w-20 flex-shrink-0" />
            ))}
          </div>
        </div>
        <div className="space-y-4">
          <Skeleton className="h-8 w-3/4" />
          <Skeleton className="h-4 w-24" />
          <div className="flex items-baseline gap-3 pt-2">
            <Skeleton className="h-9 w-28" />
            <Skeleton className="h-5 w-20" />
            <Skeleton variant="rectangular" className="h-6 w-14 rounded-md" />
          </div>
          <Skeleton className="h-4 w-20" />
          <div className="pt-4 space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </div>
          <div className="pt-4 space-y-2">
            <Skeleton className="h-4 w-16" />
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex gap-4">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-24" />
              </div>
            ))}
          </div>
          <div className="border-t border-border pt-6 mt-4">
            <div className="flex items-center gap-4">
              <Skeleton variant="rectangular" className="h-11 w-28" />
              <Skeleton variant="rectangular" className="h-11 flex-1" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
