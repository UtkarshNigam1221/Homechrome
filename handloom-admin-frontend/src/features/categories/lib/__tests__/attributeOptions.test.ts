import { describe, expect, it } from 'vitest';

import type { CategoryAttribute } from '../../types';
import { mergeAttributeOptions, toAttributeValues } from '../attributeOptions';

const attr = (options?: CategoryAttribute['options']): CategoryAttribute => ({
  name: 'color',
  label: 'Colour',
  type: 'MULTI_SELECT',
  required: false,
  searchable: true,
  display_order: 1,
  options,
});

describe('toAttributeValues', () => {
  it('normalizes the single-value and multi-value forms', () => {
    expect(toAttributeValues(['Green', 'White'])).toEqual(['Green', 'White']);
    expect(toAttributeValues('Green')).toEqual(['Green']);
  });

  it('treats absent values as empty', () => {
    expect(toAttributeValues(undefined)).toEqual([]);
    expect(toAttributeValues(null)).toEqual([]);
    expect(toAttributeValues('')).toEqual([]);
  });

  it('drops absent entries inside an array rather than stringifying them', () => {
    expect(toAttributeValues(['Green', null, '', undefined])).toEqual(['Green']);
  });
});

describe('mergeAttributeOptions', () => {
  const defined = [
    { value: 'White', label: 'White' },
    { value: 'Green', label: 'Green' },
  ];

  it('appends values the definition does not cover', () => {
    expect(mergeAttributeOptions(attr(defined), ['Green', 'Multicolour'])).toEqual([
      ...defined,
      { value: 'Multicolour', label: 'Multicolour' },
    ]);
  });

  it('returns the defined options untouched when every value is known', () => {
    expect(mergeAttributeOptions(attr(defined), ['Green'])).toBe(defined);
  });

  it('discovers options for an attribute that defines none', () => {
    expect(mergeAttributeOptions(attr(undefined), ['Cotton'])).toEqual([
      { value: 'Cotton', label: 'Cotton' },
    ]);
  });

  it('drops repeats and empty values', () => {
    expect(mergeAttributeOptions(attr(defined), ['Red', 'Red', ''])).toEqual([
      ...defined,
      { value: 'Red', label: 'Red' },
    ]);
  });
});
