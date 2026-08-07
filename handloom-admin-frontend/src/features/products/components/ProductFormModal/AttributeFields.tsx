import {
  mergeAttributeOptions,
  toAttributeValues,
} from '@/features/categories/lib/attributeOptions';
import type { CategoryAttribute } from '@/features/categories/types';
import { Input, Select } from '@/shared/components/ui';

interface AttributeFieldsProps {
  attributes: CategoryAttribute[];
  attributeValues: Record<string, unknown>;
  onAttributeChange: (attrName: string, value: unknown) => void;
  onMultiSelectToggle: (attrName: string, optionValue: string) => void;
}

function renderAttributeField(
  attr: CategoryAttribute,
  attributeValues: Record<string, unknown>,
  onAttributeChange: (attrName: string, value: unknown) => void,
  onMultiSelectToggle: (attrName: string, optionValue: string) => void
) {
  const value = attributeValues[attr.name];

  switch (attr.type) {
    case 'TEXT':
      return (
        <Input
          key={attr.name}
          label={attr.label}
          placeholder={`Enter ${attr.label.toLowerCase()}`}
          value={(value as string) || ''}
          onChange={(e) => onAttributeChange(attr.name, e.target.value)}
          required={attr.required}
        />
      );

    case 'NUMBER':
      return (
        <Input
          key={attr.name}
          label={attr.label}
          type="number"
          step="any"
          placeholder={`Enter ${attr.label.toLowerCase()}`}
          value={value !== undefined && value !== null ? String(value) : ''}
          onChange={(e) =>
            onAttributeChange(attr.name, e.target.value ? Number(e.target.value) : '')
          }
          required={attr.required}
        />
      );

    case 'SELECT':
      return (
        <Select
          key={attr.name}
          label={attr.label}
          options={mergeAttributeOptions(attr, toAttributeValues(value))}
          placeholder={`Select ${attr.label.toLowerCase()}`}
          value={(value as string) || ''}
          onChange={(e) => onAttributeChange(attr.name, e.target.value)}
          required={attr.required}
        />
      );

    case 'MULTI_SELECT': {
      const selectedValues = toAttributeValues(value);
      const options = mergeAttributeOptions(attr, selectedValues);
      return (
        <div key={attr.name}>
          <label className="label">
            {attr.label}
            {attr.required && <span className="text-red-500 ml-1">*</span>}
          </label>
          <div className="mt-1 space-y-2 p-3 border border-gray-200 rounded-lg max-h-40 overflow-y-auto">
            {options.length > 0 ? (
              options.map((opt) => (
                <label key={opt.value} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedValues.includes(opt.value)}
                    onChange={() => onMultiSelectToggle(attr.name, opt.value)}
                    className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                  />
                  <span className="text-sm text-gray-700">{opt.label}</span>
                </label>
              ))
            ) : (
              <p className="text-sm text-gray-500">No options available</p>
            )}
          </div>
        </div>
      );
    }

    case 'BOOLEAN':
      return (
        <div key={attr.name} className="flex items-center gap-3 pt-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!value}
              onChange={(e) => onAttributeChange(attr.name, e.target.checked)}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">
              {attr.label}
              {attr.required && <span className="text-red-500 ml-1">*</span>}
            </span>
          </label>
        </div>
      );

    default:
      return (
        <Input
          key={attr.name}
          label={attr.label}
          placeholder={`Enter ${attr.label.toLowerCase()}`}
          value={(value as string) || ''}
          onChange={(e) => onAttributeChange(attr.name, e.target.value)}
          required={attr.required}
        />
      );
  }
}

export function AttributeFields({
  attributes,
  attributeValues,
  onAttributeChange,
  onMultiSelectToggle,
}: AttributeFieldsProps) {
  if (attributes.length === 0) return null;

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-1">Category Attributes</h3>
      <p className="text-xs text-gray-500 mb-3">
        These attributes are specific to the selected category
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
        {attributes
          .sort((a, b) => a.display_order - b.display_order)
          .map((attr) =>
            renderAttributeField(attr, attributeValues, onAttributeChange, onMultiSelectToggle)
          )}
      </div>
    </div>
  );
}
