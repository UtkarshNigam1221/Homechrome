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

// Full page data loading (for when data is fetching)
export interface DataLoadingProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
}

export function DataLoading({ message, size = 'md' }: DataLoadingProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <LoadingSpinner size={size} />
      {message && <p className="mt-4 text-sm text-gray-500">{message}</p>}
    </div>
  );
}
