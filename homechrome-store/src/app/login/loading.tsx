import Skeleton from '@/components/skeleton/Skeleton';

export default function LoginLoading() {
  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4 py-16">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center space-y-3">
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-4 w-36" />
        </div>
        <div className="rounded-xl bg-white p-6 shadow-sm space-y-4">
          <Skeleton className="h-4 w-28" />
          <div className="flex gap-2">
            <Skeleton variant="rectangular" className="h-10 w-14" />
            <Skeleton variant="rectangular" className="h-10 flex-1" />
          </div>
          <Skeleton variant="rectangular" className="h-10 w-full" />
        </div>
      </div>
    </div>
  );
}
