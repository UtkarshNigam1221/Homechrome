import { cn } from '@/lib/utils';

interface AlertProps {
  variant?: 'error' | 'warning' | 'info' | 'success';
  children: React.ReactNode;
  className?: string;
}

const variantStyles = {
  error: 'border-red-200 bg-red-50 text-red-700',
  warning: 'border-amber-200 bg-amber-50 text-amber-700',
  info: 'border-blue-200 bg-blue-50 text-blue-700',
  success: 'border-green-200 bg-green-50 text-green-800',
};

export function Alert({ variant = 'error', children, className }: AlertProps) {
  return (
    <div
      role="alert"
      className={cn('rounded-lg border p-4 text-sm', variantStyles[variant], className)}
    >
      {children}
    </div>
  );
}
