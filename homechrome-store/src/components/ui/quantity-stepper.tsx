'use client';

import { MinusIcon, PlusIcon } from '@heroicons/react/24/outline';
import { Button as ButtonPrimitive } from '@base-ui/react/button';

import { cn } from '@/lib/utils';

interface QuantityStepperProps {
  value: number;
  onIncrement: () => void;
  onDecrement: () => void;
  disabled?: boolean;
  disableDecrement?: boolean;
  loading?: boolean;
  variant?: 'default' | 'primary';
  size?: 'sm' | 'default' | 'lg';
  className?: string;
}

const sizeConfig = {
  sm: { button: 'px-2.5 py-1.5', icon: 'h-3.5 w-3.5', count: 'w-10 text-sm' },
  default: { button: 'px-3 py-2', icon: 'h-4 w-4', count: 'w-10 text-sm' },
  lg: { button: 'px-4 py-2.5', icon: 'h-4 w-4', count: 'w-12 text-sm font-bold' },
};

export function QuantityStepper({
  value,
  onIncrement,
  onDecrement,
  disabled = false,
  disableDecrement = false,
  loading = false,
  variant = 'default',
  size = 'default',
  className,
}: QuantityStepperProps) {
  const isPrimary = variant === 'primary';
  const s = sizeConfig[size];

  return (
    <div
      className={cn(
        'inline-flex items-center rounded-lg border',
        isPrimary ? 'border-primary bg-primary/5' : 'border-border',
        className,
      )}
    >
      <ButtonPrimitive
        onClick={onDecrement}
        disabled={disabled || disableDecrement}
        className={cn(
          'cursor-pointer transition-colors disabled:cursor-not-allowed disabled:opacity-40',
          s.button,
          isPrimary
            ? 'text-primary hover:bg-primary/10'
            : 'text-foreground hover:bg-background',
        )}
        aria-label="Decrease quantity"
      >
        <MinusIcon className={s.icon} />
      </ButtonPrimitive>
      <span
        className={cn(
          'text-center font-medium',
          s.count,
          isPrimary ? 'text-primary' : 'text-foreground',
        )}
      >
        {loading ? '...' : value}
      </span>
      <ButtonPrimitive
        onClick={onIncrement}
        disabled={disabled}
        className={cn(
          'cursor-pointer transition-colors disabled:cursor-not-allowed disabled:opacity-40',
          s.button,
          isPrimary
            ? 'text-primary hover:bg-primary/10'
            : 'text-foreground hover:bg-background',
        )}
        aria-label="Increase quantity"
      >
        <PlusIcon className={s.icon} />
      </ButtonPrimitive>
    </div>
  );
}
