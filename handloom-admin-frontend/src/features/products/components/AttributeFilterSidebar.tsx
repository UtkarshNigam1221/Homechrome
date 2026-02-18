import { ChevronDown, ChevronUp, X } from 'lucide-react';
import { useState } from 'react';

import type { CategoryAttribute } from '@/features/categories/types';
import { Button } from '@/shared/components/ui';

interface AttributeFilterSidebarProps {
  attributes: CategoryAttribute[];
  selectedFilters: Record<string, string[]>;
  onFilterChange: (filters: Record<string, string[]>) => void;
  onClearFilters: () => void;
  isLoading?: boolean;
}

export function AttributeFilterSidebar({
  attributes,
  selectedFilters,
  onFilterChange,
  onClearFilters,
  isLoading = false,
}: AttributeFilterSidebarProps) {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(attributes.filter((a) => a.searchable).map((a) => a.name))
  );

  // Only show searchable attributes with options
  const filterableAttributes = attributes.filter(
    (attr) => attr.searchable && attr.options && attr.options.length > 0
  );

  const toggleSection = (name: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  const handleOptionToggle = (attrName: string, optionValue: string) => {
    const currentValues = selectedFilters[attrName] || [];
    const newValues = currentValues.includes(optionValue)
      ? currentValues.filter((v) => v !== optionValue)
      : [...currentValues, optionValue];

    const newFilters = { ...selectedFilters };
    if (newValues.length > 0) {
      newFilters[attrName] = newValues;
    } else {
      delete newFilters[attrName];
    }
    onFilterChange(newFilters);
  };

  const activeFilterCount = Object.values(selectedFilters).reduce(
    (count, values) => count + values.length,
    0
  );

  if (filterableAttributes.length === 0) {
    return null;
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
        <h3 className="font-medium text-gray-900">
          Filters
          {activeFilterCount > 0 && (
            <span className="ml-2 inline-flex items-center justify-center px-2 py-0.5 text-xs font-medium bg-indigo-100 text-indigo-800 rounded-full">
              {activeFilterCount}
            </span>
          )}
        </h3>
        {activeFilterCount > 0 && (
          <Button variant="ghost" size="sm" onClick={onClearFilters} className="text-gray-500">
            <X className="w-4 h-4 mr-1" />
            Clear
          </Button>
        )}
      </div>

      {/* Filter Sections */}
      <div className="divide-y divide-gray-200">
        {filterableAttributes.map((attr) => (
          <div key={attr.name} className="px-4 py-3">
            {/* Section Header */}
            <button
              type="button"
              onClick={() => toggleSection(attr.name)}
              className="flex items-center justify-between w-full text-left"
              disabled={isLoading}
            >
              <span className="text-sm font-medium text-gray-700">{attr.label}</span>
              {expandedSections.has(attr.name) ? (
                <ChevronUp className="w-4 h-4 text-gray-400" />
              ) : (
                <ChevronDown className="w-4 h-4 text-gray-400" />
              )}
            </button>

            {/* Options */}
            {expandedSections.has(attr.name) && attr.options && (
              <div className="mt-2 space-y-2">
                {attr.options.map((option) => {
                  const isSelected = (selectedFilters[attr.name] || []).includes(option.value);
                  return (
                    <label
                      key={option.value}
                      className={`flex items-center gap-2 cursor-pointer ${
                        isLoading ? 'opacity-50 cursor-not-allowed' : ''
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => handleOptionToggle(attr.name, option.value)}
                        disabled={isLoading}
                        className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                      />
                      <span className="text-sm text-gray-600">{option.label}</span>
                    </label>
                  );
                })}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
