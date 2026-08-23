import { clsx } from 'clsx';
import { forwardRef } from 'react';

import type { SelectOption } from './Select';

export interface MultiSelectProps extends Omit<
  React.SelectHTMLAttributes<HTMLSelectElement>,
  'multiple'
> {
  label?: string;
  error?: string;
  hint?: string;
  options: SelectOption[];
}

// A native <select multiple>. Deliberately not a combobox library: the catalogue is
// small enough that the platform control is enough, and it is keyboard accessible
// for free. Reach for a picker when the option count makes this unusable.
export const MultiSelect = forwardRef<HTMLSelectElement, MultiSelectProps>(
  ({ className, label, error, hint, options, id, ...props }, ref) => {
    const selectId = id || label?.toLowerCase().replace(/\s+/g, '-');

    return (
      <div className="w-full">
        {label && (
          <label htmlFor={selectId} className="label">
            {label}
            {props.required && <span className="text-red-500 ml-1">*</span>}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          multiple
          size={Math.min(options.length || 1, 5)}
          className={clsx(
            'select h-auto',
            error && 'border-red-500 focus:ring-red-500 focus:border-red-500',
            className
          )}
          {...props}
        >
          {options.map((option) => (
            <option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </option>
          ))}
        </select>
        {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
        {hint && !error && <p className="mt-1 text-sm text-gray-500">{hint}</p>}
      </div>
    );
  }
);

MultiSelect.displayName = 'MultiSelect';
