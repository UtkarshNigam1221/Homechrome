'use client';

import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { ActionIcon, TextInput } from '@mantine/core';

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
      <TextInput
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        radius="xl"
        size="md"
        rightSection={
          <ActionIcon
            type="submit"
            variant="subtle"
            color="navy"
            aria-label="Search"
          >
            <MagnifyingGlassIcon width={18} height={18} />
          </ActionIcon>
        }
      />
    </form>
  );
}
