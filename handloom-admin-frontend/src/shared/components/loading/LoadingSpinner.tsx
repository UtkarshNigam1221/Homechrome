import { clsx } from 'clsx';

export interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  const sizes = {
    sm: 'w-4 h-4 border-2',
    md: 'w-8 h-8 border-2',
    lg: 'w-12 h-12 border-3',
  };

  return (
    <div
      className={clsx(
        'animate-spin rounded-full border-gray-200 border-t-primary-600',
        sizes[size],
        className
      )}
    />
  );
}
