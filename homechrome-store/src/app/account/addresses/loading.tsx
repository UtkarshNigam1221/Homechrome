import Skeleton from '@/components/skeleton/Skeleton';

export default function AddressesLoading() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-6 w-36" />
        <Skeleton variant="rectangular" className="h-9 w-28 rounded-lg" />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {[1, 2].map((i) => (
          <div key={i} className="rounded-lg border border-border bg-white p-5 space-y-3">
            <Skeleton className="h-5 w-36" />
            <div className="space-y-1">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
            </div>
            <div className="flex gap-2 pt-1">
              <Skeleton variant="rectangular" className="h-7 w-14 rounded-md" />
              <Skeleton variant="rectangular" className="h-7 w-16 rounded-md" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
