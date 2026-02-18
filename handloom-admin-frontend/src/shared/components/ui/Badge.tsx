import { clsx } from 'clsx';

export interface BadgeProps {
  children: React.ReactNode;
  variant?: 'success' | 'warning' | 'danger' | 'info' | 'gray' | 'primary';
  size?: 'sm' | 'md';
  dot?: boolean;
  className?: string;
}

export function Badge({
  children,
  variant = 'gray',
  size = 'sm',
  dot = false,
  className,
}: BadgeProps) {
  // Premium badge styles with ring effect - PrebuiltUI inspired
  const variants = {
    success: 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20',
    warning: 'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20',
    danger: 'bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/20',
    info: 'bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-600/20',
    gray: 'bg-gray-100 text-gray-700 ring-1 ring-inset ring-gray-500/10',
    primary: 'bg-primary-50 text-primary-700 ring-1 ring-inset ring-primary-600/20',
  };

  const sizes = {
    sm: 'px-2.5 py-1 text-xs',
    md: 'px-3 py-1.5 text-sm',
  };

  const dotColors = {
    success: 'bg-emerald-500',
    warning: 'bg-amber-500',
    danger: 'bg-red-500',
    info: 'bg-blue-500',
    gray: 'bg-gray-500',
    primary: 'bg-primary-500',
  };

  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-lg font-semibold tracking-wide',
        variants[variant],
        sizes[size],
        className
      )}
    >
      {dot && <span className={clsx('w-1.5 h-1.5 rounded-full', dotColors[variant])} />}
      {children}
    </span>
  );
}
