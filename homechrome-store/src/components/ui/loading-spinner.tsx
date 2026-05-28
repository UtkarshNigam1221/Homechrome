'use client';

import HCLoader, { HCLoaderSize } from './HCLoader';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const legacyToHC: Record<'sm' | 'md' | 'lg', HCLoaderSize> = {
  sm: 'sm',
  md: 'md',
  lg: 'lg',
};

export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  return <HCLoader size={legacyToHC[size]} className={className} />;
}

interface LoadingBlockProps {
  size?: LoadingSpinnerProps['size'];
  className?: string;
  label?: string;
}

export function LoadingBlock({ size = 'lg', className, label }: LoadingBlockProps) {
  return (
    <div className={`flex w-full items-center justify-center py-20 ${className ?? ''}`}>
      <HCLoader size={legacyToHC[size]} label={label} />
    </div>
  );
}
