import { cn } from '@/lib/utils';

import { Input } from './input';

interface FormFieldProps extends React.ComponentProps<'input'> {
  label?: string;
  error?: string;
  helperText?: string;
}

function FormField({ label, error, helperText, className, id, ...props }: FormFieldProps) {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');

  return (
    <div className="w-full">
      {label && (
        <label
          htmlFor={inputId}
          className="mb-1.5 block text-sm font-medium text-foreground"
        >
          {label}
        </label>
      )}
      <Input
        id={inputId}
        className={cn(
          'h-10 rounded-lg px-4 py-2.5',
          error && 'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
          className,
        )}
        {...props}
      />
      {error && <p className="mt-1 text-sm text-destructive">{error}</p>}
      {helperText && !error && (
        <p className="mt-1 text-sm text-muted-foreground">{helperText}</p>
      )}
    </div>
  );
}

export default FormField;
