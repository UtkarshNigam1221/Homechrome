import Skeleton from '@/components/skeleton/Skeleton';
import ProductGridSkeleton from '@/components/skeleton/ProductGridSkeleton';

export default function CategoryDetailLoading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <Skeleton className="h-4 w-48 mb-6" />
      <div className="mb-8">
        <Skeleton className="h-9 w-56" />
        <Skeleton className="h-4 w-80 mt-2" />
        <Skeleton className="h-4 w-24 mt-1" />
      </div>
      <div className="flex gap-8">
        <div className="hidden w-64 shrink-0 lg:block">
          <div className="rounded-xl bg-white p-5 shadow-sm space-y-4">
            <Skeleton className="h-5 w-20" />
            <Skeleton variant="rectangular" className="h-8 w-full" />
            <Skeleton variant="rectangular" className="h-8 w-full" />
            <Skeleton className="h-5 w-24 mt-4" />
            <Skeleton variant="rectangular" className="h-5 w-full" />
          </div>
        </div>
        <div className="flex-1">
          <ProductGridSkeleton count={8} />
        </div>
      </div>
    </div>
  );
}
