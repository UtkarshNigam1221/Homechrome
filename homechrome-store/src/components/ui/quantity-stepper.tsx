'use client';

import { MinusIcon, PlusIcon } from '@heroicons/react/24/outline';
import { ActionIcon, Group, Text } from '@mantine/core';

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
  fullWidth?: boolean;
}

const sizeConfig = {
  sm: { icon: 'sm' as const, iconSize: 14, count: 36, fontSize: 'sm' as const },
  default: { icon: 'md' as const, iconSize: 16, count: 40, fontSize: 'sm' as const },
  lg: { icon: 'lg' as const, iconSize: 18, count: 48, fontSize: 'md' as const },
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
  fullWidth = false,
}: QuantityStepperProps) {
  const isPrimary = variant === 'primary';
  const s = sizeConfig[size];
  const color = isPrimary ? 'brand' : 'navy';

  return (
    <Group
      gap={0}
      wrap="nowrap"
      justify={fullWidth ? 'space-between' : undefined}
      className={className}
      style={{
        display: fullWidth ? 'flex' : 'inline-flex',
        width: fullWidth ? '100%' : undefined,
        border: `1px solid var(--mantine-color-${isPrimary ? 'brand-5' : 'default-border'})`,
        borderRadius: 'var(--mantine-radius-md)',
        backgroundColor: isPrimary ? 'var(--mantine-color-brand-0)' : undefined,
      }}
    >
      <ActionIcon
        size={s.icon}
        variant="subtle"
        color={color}
        onClick={onDecrement}
        disabled={disabled || disableDecrement}
        aria-label="Decrease quantity"
      >
        <MinusIcon width={s.iconSize} height={s.iconSize} />
      </ActionIcon>
      <Text
        ta="center"
        fw={isPrimary ? 700 : 500}
        size={s.fontSize}
        c={isPrimary ? 'brand.5' : 'navy.7'}
        w={s.count}
      >
        {loading ? '...' : value}
      </Text>
      <ActionIcon
        size={s.icon}
        variant="subtle"
        color={color}
        onClick={onIncrement}
        disabled={disabled}
        aria-label="Increase quantity"
      >
        <PlusIcon width={s.iconSize} height={s.iconSize} />
      </ActionIcon>
    </Group>
  );
}
