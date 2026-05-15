import { clsx } from 'clsx';

import type { ShipmentPriority } from '@/features/shipping';

interface PriorityToggleProps {
  value: ShipmentPriority;
  onChange: (priority: ShipmentPriority) => void;
  disabled?: boolean;
}

const OPTIONS: { value: ShipmentPriority; label: string }[] = [
  { value: 'NORMAL', label: 'Normal' },
  { value: 'PRIORITY', label: 'Priority' },
];

export function PriorityToggle({ value, onChange, disabled }: PriorityToggleProps) {
  return (
    <div
      role="radiogroup"
      aria-label="Shipment priority"
      className={clsx(
        'inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5',
        disabled && 'opacity-50'
      )}
    >
      {OPTIONS.map((opt) => (
        <button
          key={opt.value}
          type="button"
          role="radio"
          aria-checked={value === opt.value}
          disabled={disabled}
          onClick={() => onChange(opt.value)}
          className={clsx(
            'px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
            value === opt.value
              ? 'bg-white text-gray-900 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
