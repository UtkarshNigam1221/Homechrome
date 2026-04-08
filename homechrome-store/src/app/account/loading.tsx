import Skeleton from '@/components/skeleton/Skeleton';

export default function AccountLoading() {
  return (
    <div className="space-y-6">
      {/* Profile Information */}
      <div className="rounded-lg border border-border bg-white p-6">
        <Skeleton className="h-6 w-44 mb-4" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {[1, 2, 3, 4].map((i) => (
            <div key={i}>
              <Skeleton className="h-4 w-20 mb-1" />
              <Skeleton className="h-5 w-36" />
            </div>
          ))}
        </div>
      </div>

      {/* Navigation cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="rounded-lg border border-border bg-white p-5"
          >
            <Skeleton variant="rectangular" className="mb-2 h-10 w-10 rounded-lg" />
            <Skeleton className="h-5 w-24" />
            <Skeleton className="mt-1 h-4 w-full" />
          </div>
        ))}
      </div>

      {/* Account Summary */}
      <div className="rounded-lg border border-border bg-white p-6">
        <Skeleton className="h-6 w-40 mb-2" />
        <div className="grid grid-cols-2 gap-4">
          {[1, 2].map((i) => (
            <div key={i} className="rounded-lg bg-background p-4 text-center">
              <Skeleton className="mx-auto h-8 w-16" />
              <Skeleton className="mx-auto mt-1 h-4 w-20" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
