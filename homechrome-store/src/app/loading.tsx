import Skeleton from '@/components/skeleton/Skeleton';
import ProductGridSkeleton from '@/components/skeleton/ProductGridSkeleton';

export default function HomeLoading() {
  return (
    <div>
      {/* Hero skeleton */}
      <section className="bg-foreground">
        <div className="mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 lg:px-8">
          <div className="flex flex-col items-center text-center space-y-6">
            <Skeleton className="h-10 w-3/4 max-w-lg sm:h-12" />
            <Skeleton className="h-5 w-2/3 max-w-md" />
            <Skeleton className="h-5 w-1/2 max-w-sm" />
            <div className="flex gap-4 pt-4">
              <Skeleton variant="rectangular" className="h-12 w-32" />
              <Skeleton variant="rectangular" className="h-12 w-40" />
            </div>
          </div>
        </div>
      </section>

      {/* Features skeleton */}
      <section className="bg-white py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 gap-8 sm:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex flex-col items-center text-center space-y-3">
                <Skeleton variant="circular" className="h-12 w-12" />
                <Skeleton className="h-5 w-24" />
                <Skeleton className="h-4 w-48" />
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Categories skeleton */}
      <section className="py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between mb-8">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-16" />
          </div>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="overflow-hidden rounded-xl bg-white shadow-sm">
                <Skeleton variant="rectangular" className="aspect-[4/3] w-full rounded-none" />
                <div className="p-4 space-y-2">
                  <Skeleton className="h-5 w-2/3" />
                  <Skeleton className="h-4 w-1/3" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Products skeleton */}
      <section className="bg-white py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between mb-8">
            <Skeleton className="h-8 w-40" />
            <Skeleton className="h-4 w-16" />
          </div>
          <ProductGridSkeleton count={8} />
        </div>
      </section>
    </div>
  );
}
