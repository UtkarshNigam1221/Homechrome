import { clsx } from 'clsx';
import type { LucideIcon } from 'lucide-react';

interface SegmentedTabsProps<T extends string> {
  value: T;
  onChange: (value: T) => void;
  options: { id: T; label: string; icon?: LucideIcon }[];
}

export function SegmentedTabs<T extends string>({
  value,
  onChange,
  options,
}: SegmentedTabsProps<T>) {
  return (
    <div className="flex gap-1 rounded-lg bg-gray-100 p-0.5" role="tablist">
      {options.map(({ id, label, icon: Icon }) => (
        <button
          key={id}
          type="button"
          role="tab"
          aria-selected={value === id}
          onClick={() => onChange(id)}
          className={clsx(
            'flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500',
            value === id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'
          )}
        >
          {Icon && <Icon className="h-3 w-3" />}
          {label}
        </button>
      ))}
    </div>
  );
}
