import { clsx } from 'clsx';

import { LoadingSpinner } from './LoadingSpinner';

export interface LoadingOverlayProps {
  message?: string;
}

export function LoadingOverlay({ message = 'Loading...' }: LoadingOverlayProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-white/80 backdrop-blur-sm">
      <div className="flex flex-col items-center gap-4">
        <LoadingSpinner size="lg" />
        <p className="text-sm font-medium text-gray-600">{message}</p>
      </div>
    </div>
  );
}

// Inline loading for buttons or small areas
export function InlineLoading({ className }: { className?: string }) {
  return (
    <div className={clsx('flex items-center gap-2', className)}>
      <LoadingSpinner size="sm" />
      <span className="text-sm text-gray-500">Loading...</span>
    </div>
  );
}

// Loading bar (for progress indication)
export function LoadingBar() {
  return (
    <div className="w-full h-1 bg-gray-200 overflow-hidden">
      <div className="h-full bg-primary-600 animate-loading-bar" />
    </div>
  );
}
