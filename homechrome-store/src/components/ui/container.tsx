import { cn } from '@/lib/utils';

interface ContainerProps extends React.ComponentProps<'div'> {
  size?: 'default' | 'narrow';
}

export function Container({ size = 'default', className, ...props }: ContainerProps) {
  return (
    <div
      className={cn(
        'mx-auto px-4 sm:px-6 lg:px-8',
        size === 'default' ? 'max-w-7xl' : 'max-w-6xl',
        className,
      )}
      {...props}
    />
  );
}
