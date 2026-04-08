import Skeleton from '@/components/skeleton/Skeleton';

export default function TrackLoading() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-12 sm:px-6 lg:px-8">
      <div className="mb-8 flex flex-col items-center space-y-2">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-72" />
      </div>
      <div className="flex gap-3 mb-8">
        <Skeleton variant="rectangular" className="h-10 flex-1 rounded-full" />
        <Skeleton variant="rectangular" className="h-10 w-20 rounded-lg" />
      </div>
    </div>
  );
}
