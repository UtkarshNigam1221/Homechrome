interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular';
}

const variantClasses = {
  text: 'h-4 rounded-md',
  circular: 'rounded-full',
  rectangular: 'rounded-lg',
};

export default function Skeleton({ className = '', variant = 'text' }: SkeletonProps) {
  return (
    <div
      aria-hidden="true"
      className={`skeleton-shimmer ${variantClasses[variant]} ${className}`}
    />
  );
}
