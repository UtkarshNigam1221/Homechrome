'use client';

import { useCallback, useState } from 'react';

import Button from '@/components/common/Button';
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
  const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>({});

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

  const toggleSection = useCallback((name: string) => {
    setCollapsedSections((prev) => ({ ...prev, [name]: !prev[name] }));
  }, []);

  const hasActiveFilters =
    filters.minPrice !== null ||
    filters.maxPrice !== null ||
    filters.inStockOnly ||
    Object.keys(filters.attributeFilters).length > 0;

  // Build attribute sections from filterOptions + categoryAttributes metadata
  const attributeSections = categoryAttributes
    ?.filter((attr) => attr.searchable && filterOptions?.[attr.name]?.length)
    .sort((a, b) => a.display_order - b.display_order);

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

      {/* Dynamic attribute filters */}
      {attributeSections?.map((attr) => {
        const values = filterOptions?.[attr.name] || [];
        const selected = filters.attributeFilters[attr.name] || [];
        const isCollapsed = collapsedSections[attr.name];

        return (
          <div key={attr.name}>
            <button
              type="button"
              onClick={() => toggleSection(attr.name)}
              className="flex w-full items-center justify-between text-sm font-medium text-foreground"
            >
              <span>{attr.label || attr.name}</span>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
                stroke="currentColor"
                className={`h-4 w-4 transition-transform ${isCollapsed ? '' : 'rotate-180'}`}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
              </svg>
            </button>
            {!isCollapsed && (
              <div className="mt-2 space-y-2">
                {values.map((value) => (
                  <label key={value} className="flex cursor-pointer items-center gap-3">
                    <input
                      type="checkbox"
                      checked={selected.includes(value)}
                      onChange={() => handleAttributeToggle(attr.name, value)}
                      className="h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
                    />
                    <span className="text-sm text-foreground capitalize">{value}</span>
                  </label>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </aside>
  );
}
