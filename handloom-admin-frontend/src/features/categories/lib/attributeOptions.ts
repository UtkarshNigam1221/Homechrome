import type { AttributeOption, CategoryAttribute } from '../types';

/**
 * Normalizes an attribute value to a list. Single-value attributes come back
 * from the API as a plain string, multi-value ones as an array.
 */
export function toAttributeValues(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String);
  if (value === undefined || value === null || value === '') return [];
  return [String(value)];
}

/**
 * Returns the attribute's option list widened to cover `extraValues`.
 *
 * A category's option list is a definition, not a closed set: options get
 * edited after products are saved, and import scripts write values directly
 * into product_attribute_values. Values that exist in the data but not in the
 * definition must still be selectable, or they disappear from every UI that
 * renders options — a saved value looks unset, and a filter can't reach the
 * products carrying it.
 *
 * Callers supply whatever values they know about: the product form passes the
 * values saved on the product being edited, the filter sidebar passes the
 * distinct values discovered across the category.
 */
export function mergeAttributeOptions(
  attr: CategoryAttribute,
  extraValues: string[]
): AttributeOption[] {
  const options = attr.options || [];
  const seen = new Set(options.map((opt) => opt.value));
  const extra: AttributeOption[] = [];
  for (const value of extraValues) {
    if (value === '' || seen.has(value)) continue;
    seen.add(value); // also dedupes repeats within extraValues
    extra.push({ value, label: value });
  }
  return extra.length > 0 ? [...options, ...extra] : options;
}
