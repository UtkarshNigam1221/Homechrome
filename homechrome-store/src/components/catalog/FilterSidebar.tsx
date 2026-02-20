'use client';

import { useCallback, useState } from 'react';

import Button from '@/components/common/Button';

export interface FilterValues {
  minPrice: number | null;
  maxPrice: number | null;
  inStockOnly: boolean;
}

interface FilterSidebarProps {
  filters: FilterValues;
  onFiltersChange: (filters: FilterValues) => void;
}

export default function FilterSidebar({ filters, onFiltersChange }: FilterSidebarProps) {
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
    onFiltersChange({ minPrice: null, maxPrice: null, inStockOnly: false });
  }, [onFiltersChange]);

  const hasActiveFilters =
    filters.minPrice !== null || filters.maxPrice !== null || filters.inStockOnly;

  return (
    <aside className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-foreground">Filters</h3>
        {hasActiveFilters && (
          <button
            type="button"
            onClick={handleClearAll}
            className="text-sm text-primary hover:text-primary-dark"
          >
            Clear all
          </button>
        )}
      </div>

      {/* Price range */}
      <div>
        <h4 className="text-sm font-medium text-foreground">Price Range</h4>
        <div className="mt-3 flex items-center gap-2">
          <div className="flex-1">
            <input
              type="number"
              value={minInput}
              onChange={(e) => setMinInput(e.target.value)}
              placeholder="Min"
              min={0}
              className="w-full rounded-lg border border-border bg-white px-3 py-2 text-sm text-foreground placeholder:text-muted focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
            />
          </div>
          <span className="text-sm text-muted">to</span>
          <div className="flex-1">
            <input
              type="number"
              value={maxInput}
              onChange={(e) => setMaxInput(e.target.value)}
              placeholder="Max"
              min={0}
              className="w-full rounded-lg border border-border bg-white px-3 py-2 text-sm text-foreground placeholder:text-muted focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
            />
          </div>
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
      <div>
        <label className="flex cursor-pointer items-center gap-3">
          <input
            type="checkbox"
            checked={filters.inStockOnly}
            onChange={(e) =>
              onFiltersChange({ ...filters, inStockOnly: e.target.checked })
            }
            className="h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
          />
          <span className="text-sm text-foreground">In stock only</span>
        </label>
      </div>
    </aside>
  );
}
