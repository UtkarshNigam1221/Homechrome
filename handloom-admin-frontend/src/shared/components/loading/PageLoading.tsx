import { LoadingSpinner } from './LoadingSpinner';

export interface PageLoadingProps {
  message?: string;
}

export function PageLoading({ message = 'Loading...' }: PageLoadingProps) {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="flex flex-col items-center gap-4">
        <LoadingSpinner size="lg" />
        <p className="text-sm font-medium text-gray-600">{message}</p>
      </div>
    </div>
  );
}
