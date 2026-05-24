'use client';

import { ChevronDownIcon } from '@heroicons/react/24/outline';
import {
  Anchor,
  Button,
  Checkbox,
  Collapse,
  Group,
  NumberInput,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useCallback, useState } from 'react';

import { CategoryAttribute } from '@/types';

export interface FilterValues {
  minPrice: number | null;
  maxPrice: number | null;
  inStockOnly: boolean;
  attributeFilters: Record<string, string[]>;
}

interface FilterSidebarProps {
  filters: FilterValues;
  onFiltersChange: (filters: FilterValues) => void;
  filterOptions?: Record<string, string[]>;
  categoryAttributes?: CategoryAttribute[];
}

export default function FilterSidebar({
  filters,
  onFiltersChange,
  filterOptions,
  categoryAttributes,
}: FilterSidebarProps) {
  const [minInput, setMinInput] = useState<number | ''>(
    filters.minPrice !== null ? filters.minPrice / 100 : '',
  );
  const [maxInput, setMaxInput] = useState<number | ''>(
    filters.maxPrice !== null ? filters.maxPrice / 100 : '',
  );

  const handleApplyPrice = useCallback(() => {
    const min = typeof minInput === 'number' ? Math.round(minInput * 100) : null;
    const max = typeof maxInput === 'number' ? Math.round(maxInput * 100) : null;
    onFiltersChange({ ...filters, minPrice: min, maxPrice: max });
  }, [minInput, maxInput, filters, onFiltersChange]);

  const handleClearAll = useCallback(() => {
    setMinInput('');
    setMaxInput('');
    onFiltersChange({ minPrice: null, maxPrice: null, inStockOnly: false, attributeFilters: {} });
  }, [onFiltersChange]);

  const handleAttributeToggle = useCallback(
    (attrName: string, value: string) => {
      const current = filters.attributeFilters[attrName] || [];
      const updated = current.includes(value)
        ? current.filter((v) => v !== value)
        : [...current, value];
      const newAttrFilters = { ...filters.attributeFilters };
      if (updated.length === 0) {
        delete newAttrFilters[attrName];
      } else {
        newAttrFilters[attrName] = updated;
      }
      onFiltersChange({ ...filters, attributeFilters: newAttrFilters });
    },
    [filters, onFiltersChange],
  );

  const hasActive =
    filters.minPrice !== null ||
    filters.maxPrice !== null ||
    filters.inStockOnly ||
    Object.keys(filters.attributeFilters).length > 0;

  const attributeSections = categoryAttributes
    ?.filter((attr) => attr.searchable && filterOptions?.[attr.name]?.length)
    .sort((a, b) => a.display_order - b.display_order);

  return (
    <Stack gap="lg" component="aside">
      <Group justify="space-between">
        <Title order={3} size="md">Filters</Title>
        {hasActive && (
          <Anchor component="button" size="sm" onClick={handleClearAll}>
            Clear all
          </Anchor>
        )}
      </Group>

      <Stack gap="xs">
        <Text size="sm" fw={500}>Price Range</Text>
        <Group gap="xs" wrap="nowrap" align="center">
          <NumberInput
            value={minInput}
            onChange={(v) => setMinInput(typeof v === 'number' ? v : '')}
            placeholder="Min"
            min={0}
            hideControls
            size="sm"
          />
          <Text size="sm" c="dimmed">to</Text>
          <NumberInput
            value={maxInput}
            onChange={(v) => setMaxInput(typeof v === 'number' ? v : '')}
            placeholder="Max"
            min={0}
            hideControls
            size="sm"
          />
        </Group>
        <Button variant="outline" color="navy" size="sm" onClick={handleApplyPrice}>
          Apply
        </Button>
      </Stack>

      <Checkbox
        label="In stock only"
        checked={filters.inStockOnly}
        onChange={(e) => onFiltersChange({ ...filters, inStockOnly: e.currentTarget.checked })}
      />

      {attributeSections?.map((attr) => (
        <AttributeSection
          key={attr.name}
          label={attr.label || attr.name}
          values={filterOptions?.[attr.name] || []}
          selected={filters.attributeFilters[attr.name] || []}
          onToggle={(value) => handleAttributeToggle(attr.name, value)}
        />
      ))}
    </Stack>
  );
}

interface AttributeSectionProps {
  label: string;
  values: string[];
  selected: string[];
  onToggle: (value: string) => void;
}

function AttributeSection({ label, values, selected, onToggle }: AttributeSectionProps) {
  const [opened, { toggle }] = useDisclosure(true);

  return (
    <Stack gap="xs">
      <Group
        component="button"
        justify="space-between"
        onClick={toggle}
        style={{ background: 'none', border: 0, cursor: 'pointer', padding: 0 }}
      >
        <Text size="sm" fw={500}>{label}</Text>
        <ChevronDownIcon
          width={16}
          height={16}
          style={{
            transform: opened ? 'rotate(180deg)' : 'rotate(0)',
            transition: 'transform 0.15s',
          }}
        />
      </Group>
      <Collapse expanded={opened}>
        <Stack gap="xs">
          {values.map((value) => (
            <Checkbox
              key={value}
              label={<Text size="sm" tt="capitalize">{value}</Text>}
              checked={selected.includes(value)}
              onChange={() => onToggle(value)}
            />
          ))}
        </Stack>
      </Collapse>
    </Stack>
  );
}
