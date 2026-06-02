'use client';

import { Center } from '@mantine/core';

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
    <Center py={80} w="100%" className={className}>
      <HCLoader size={legacyToHC[size]} label={label} />
    </Center>
  );
}
