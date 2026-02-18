import { describe, expect, it } from 'vitest';

import { extractItems } from '../api';

describe('extractItems', () => {
  it('extracts from { items: [...] }', () => {
    expect(extractItems({ items: [1, 2, 3] })).toEqual([1, 2, 3]);
  });

  it('extracts from named key like { products: [...] }', () => {
    expect(extractItems({ products: [{ id: 1 }] }, 'products')).toEqual([{ id: 1 }]);
  });

  it('returns the data directly if it is an array', () => {
    expect(extractItems([1, 2, 3])).toEqual([1, 2, 3]);
  });

  it('extracts from { data: [...] }', () => {
    expect(extractItems({ data: [1, 2] })).toEqual([1, 2]);
  });

  it('returns empty array for null/undefined', () => {
    expect(extractItems(null)).toEqual([]);
    expect(extractItems(undefined)).toEqual([]);
  });

  it('prefers named key over items', () => {
    expect(extractItems({ items: [1], products: [2] }, 'products')).toEqual([2]);
  });
});
