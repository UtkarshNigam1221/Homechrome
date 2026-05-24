'use client';

import { Center, Loader, LoaderProps } from '@mantine/core';

interface LoadingSpinnerProps extends Omit<LoaderProps, 'size'> {
  size?: 'sm' | 'md' | 'lg';
}

const sizeMap = { sm: 'sm', md: 'md', lg: 'lg' } as const;

export function LoadingSpinner({ size = 'md', ...rest }: LoadingSpinnerProps) {
  return <Loader size={sizeMap[size]} color="brand" {...rest} />;
}

interface LoadingBlockProps {
  size?: LoadingSpinnerProps['size'];
  className?: string;
  label?: string;
}

export function LoadingBlock({ size = 'lg', className }: LoadingBlockProps) {
  return (
    <Center py={80} className={className}>
      <LoadingSpinner size={size} />
    </Center>
  );
}
