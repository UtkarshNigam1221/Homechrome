'use client';

import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  placeholder?: string;
  className?: string;
}

export function SearchInput({
  value,
  onChange,
  onSubmit,
  placeholder = 'Search...',
  className,
}: Readonly<SearchInputProps>) {
  return (
    <form onSubmit={onSubmit} className={className}>
      <div className="relative">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="cursor-text w-full rounded-full border border-input bg-background py-2 pl-4 pr-10 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring/30"
        />
        <button
          type="submit"
          className="cursor-pointer absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-primary"
          aria-label="Search"
        >
          <MagnifyingGlassIcon className="h-5 w-5" />
        </button>
      </div>
    </form>
  );
}
