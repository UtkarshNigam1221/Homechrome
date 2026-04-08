'use client';

import { ChevronDownIcon } from '@heroicons/react/24/outline';
import { useCallback, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
  const [minInput, setMinInput] = useState(
    filters.minPrice !== null ? String(filters.minPrice / 100) : '',
  );
  const [maxInput, setMaxInput] = useState(
    filters.maxPrice !== null ? String(filters.maxPrice / 100) : '',
  );

  const handleApplyPrice = useCallback(() => {
    const min = minInput ? Math.round(parseFloat(minInput) * 100) : null;
    const max = maxInput ? Math.round(parseFloat(maxInput) * 100) : null;
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

  const hasActiveFilters =
    filters.minPrice !== null ||
    filters.maxPrice !== null ||
    filters.inStockOnly ||
    Object.keys(filters.attributeFilters).length > 0;

  const attributeSections = categoryAttributes
    ?.filter((attr) => attr.searchable && filterOptions?.[attr.name]?.length)
    .sort((a, b) => a.display_order - b.display_order);

  return (
    <aside className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-foreground">Filters</h3>
        {hasActiveFilters && (
          <Button variant="link" size="sm" onClick={handleClearAll}>
            Clear all
          </Button>
        )}
      </div>

      {/* Price range */}
      <div>
        <h4 className="text-sm font-medium text-foreground">Price Range</h4>
        <div className="mt-3 flex items-center gap-2">
          <Input
            type="number"
            value={minInput}
            onChange={(e) => setMinInput(e.target.value)}
            placeholder="Min"
            min={0}
            className="h-9 px-3"
          />
          <span className="text-sm text-muted-foreground">to</span>
          <Input
            type="number"
            value={maxInput}
            onChange={(e) => setMaxInput(e.target.value)}
            placeholder="Max"
            min={0}
            className="h-9 px-3"
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          className="mt-3 w-full"
          onClick={handleApplyPrice}
        >
          Apply
        </Button>
      </div>

      {/* In-stock toggle */}
      <Label className="cursor-pointer gap-3">
        <Checkbox
          checked={filters.inStockOnly}
          onCheckedChange={(checked) =>
            onFiltersChange({ ...filters, inStockOnly: !!checked })
          }
        />
        In stock only
      </Label>

      {/* Dynamic attribute filters */}
      {attributeSections?.map((attr) => {
        const values = filterOptions?.[attr.name] || [];
        const selected = filters.attributeFilters[attr.name] || [];

        return (
          <Collapsible key={attr.name} defaultOpen>
            <CollapsibleTrigger className="flex w-full items-center justify-between text-sm font-medium text-foreground">
              <span>{attr.label || attr.name}</span>
              <ChevronDownIcon className="h-4 w-4 transition-transform data-[panel-open]:rotate-180" />
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-2 space-y-2">
                {values.map((value) => (
                  <Label key={value} className="cursor-pointer gap-3">
                    <Checkbox
                      checked={selected.includes(value)}
                      onCheckedChange={() => handleAttributeToggle(attr.name, value)}
                    />
                    <span className="capitalize">{value}</span>
                  </Label>
                ))}
              </div>
            </CollapsibleContent>
          </Collapsible>
        );
      })}
    </aside>
  );
}
