'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';

import { track } from '@/lib/analytics';
import { FilterValues } from '@/components/catalog/FilterSidebar';
import { fetchProductsPage } from '@/lib/api';
import { PRODUCTS_PAGE_SIZE } from '@/lib/constants';
import { Product } from '@/types';

/**
 * Parses URL search params into a FilterValues object.
 * Handles common filters (min_price, max_price, in_stock) plus
 * optional `af_*` attribute filters used on category pages.
 */
export function parseFiltersFromParams(params: URLSearchParams): FilterValues {
  const minPrice = params.get('min_price');
  const maxPrice = params.get('max_price');
  const inStock = params.get('in_stock');

  const attributeFilters: Record<string, string[]> = {};
  for (const [key, value] of params.entries()) {
    if (key.startsWith('af_') && value) {
      attributeFilters[key.slice(3)] = value.split(',').filter(Boolean);
    }
  }

  return {
    minPrice: minPrice ? Number(minPrice) : null,
    maxPrice: maxPrice ? Number(maxPrice) : null,
    inStockOnly: inStock === 'true',
    attributeFilters,
  };
}

/**
 * Serializes a FilterValues object back to URLSearchParams.
 * Includes common filters and `af_*` attribute filters.
 */
export function filtersToParams(filters: FilterValues): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
  if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
  if (filters.inStockOnly) params.set('in_stock', 'true');
  for (const [name, values] of Object.entries(filters.attributeFilters)) {
    if (values.length > 0) {
      params.set(`af_${name}`, values.join(','));
    }
  }
  return params;
}

/**
 * Returns true when any filter is active (non-default).
 */
export function hasActiveFilters(filters: FilterValues): boolean {
  return (
    filters.minPrice !== null ||
    filters.maxPrice !== null ||
    filters.inStockOnly ||
    Object.values(filters.attributeFilters).some((v) => v.length > 0)
  );
}

/**
 * Returns the list of filter-key names that differ between two filter
 * snapshots. Keys returned are the human-readable names the backend
 * stamps on `catalog_filter_applied.filter_key`: min_price, max_price,
 * in_stock, or any attribute name (material, color, ...). Attribute
 * filters are compared as sorted-string sets to ignore reordering.
 */
function diffFilterKeys(a: FilterValues, b: FilterValues): string[] {
  const changed: string[] = [];
  if (a.minPrice !== b.minPrice) changed.push('min_price');
  if (a.maxPrice !== b.maxPrice) changed.push('max_price');
  if (a.inStockOnly !== b.inStockOnly) changed.push('in_stock');
  const keys = new Set([
    ...Object.keys(a.attributeFilters),
    ...Object.keys(b.attributeFilters),
  ]);
  for (const k of keys) {
    const av = [...(a.attributeFilters[k] ?? [])].sort().join(',');
    const bv = [...(b.attributeFilters[k] ?? [])].sort().join(',');
    if (av !== bv) changed.push(k);
  }
  return changed;
}

/** Build a stable query key string from filters + extra params. */
function buildQueryString(filters: FilterValues, extraParams?: () => URLSearchParams): string {
  const params = filtersToParams(filters);
  if (extraParams) {
    for (const [key, value] of extraParams().entries()) {
      params.set(key, value);
    }
  }
  params.sort();
  return params.toString();
}

export interface UseFilteredProductsOptions {
  /** API endpoint to fetch products from, e.g. `/api/v1/store/catalog/products` */
  endpoint: string;
  /**
   * Extra static query params to include on every request (e.g. `category_id`).
   * These are merged with the filter params before each fetch.
   */
  extraParams?: () => URLSearchParams;
  /** First page of products (from SSR). */
  initialProducts: Product[];
  /** `meta.next_cursor` of that SSR page — omitted when it was the last page. */
  initialCursor?: string;
  /** Initial filter state (parsed from URL on the server or client). */
  initialFilters: FilterValues;
  /**
   * Additional params that should trigger a re-fetch when they change
   * (e.g. searchQuery in ProductsView). The caller must include them in
   * `extraParams` as well so they're sent to the API.
   */
  extraDeps?: unknown[];
}

export interface UseFilteredProductsResult {
  filters: FilterValues;
  setFilters: React.Dispatch<React.SetStateAction<FilterValues>>;
  /** Every page loaded so far, flattened. */
  products: Product[];
  loading: boolean;
  /** True while a further page exists — drive the scroll sentinel off this. */
  hasMore: boolean;
  loadMore: () => void;
}

/**
 * Shared hook that encapsulates:
 * - React Query–backed cursor-paginated product fetch (infinite scroll)
 * - Popstate listener to sync filter state on browser back/forward
 *
 * Both ProductsView and CategoryProductsView use this hook; they differ only
 * in the fetch endpoint and extra params.
 */
export function useFilteredProducts({
  endpoint,
  extraParams,
  initialProducts,
  initialCursor,
  initialFilters,
  extraDeps = [],
}: UseFilteredProductsOptions): UseFilteredProductsResult {
  const [filters, setFilters] = useState<FilterValues>(initialFilters);
  // Debounce filters so rapid changes (slider drag, typing) don't spam requests
  const [debouncedFilters, setDebouncedFilters] = useState<FilterValues>(initialFilters);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    debounceRef.current = setTimeout(() => setDebouncedFilters(filters), 300);
    return () => clearTimeout(debounceRef.current);
  }, [filters]);

  // Search input lives in extraDeps (the `q` query param). It has a longer
  // debounce than the filter sliders because the user is typing — a 1s
  // pause feels right for "I've finished my query" and dramatically cuts
  // intermediate-keystroke fetches.

  // Fire catalog_filter_applied per filter that actually changed once the
  // debounce settles. Diff against the previous debounced state so dragging
  // a slider only counts once at rest, not per intermediate value.
  const prevDebouncedRef = useRef<FilterValues | null>(null);
  useEffect(() => {
    const prev = prevDebouncedRef.current;
    prevDebouncedRef.current = debouncedFilters;
    if (prev === null) return; // first run — no diff to emit
    const changed = diffFilterKeys(prev, debouncedFilters);
    for (const key of changed) {
      track('catalog_filter_applied', { filter_key: key });
    }
  }, [debouncedFilters]);

  // Also debounce extra deps (e.g. searchQuery)
  const extraDepsKey = JSON.stringify(extraDeps);
  const [debouncedExtraDepsKey, setDebouncedExtraDepsKey] = useState(extraDepsKey);
  const extraDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    extraDebounceRef.current = setTimeout(() => setDebouncedExtraDepsKey(extraDepsKey), 1000);
    return () => clearTimeout(extraDebounceRef.current);
  }, [extraDepsKey]);

  // Sync filter state on browser back/forward
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      setFilters(parseFiltersFromParams(params));
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const queryString = buildQueryString(debouncedFilters, extraParams);

  // The SSR page is a valid page 1 only for the request the server made: no
  // filters, extra deps still at mount values. Seed it then, else fetch it.
  const [mountExtraDepsKey] = useState(extraDepsKey);
  const canSeedSSR =
    !hasActiveFilters(debouncedFilters) && debouncedExtraDepsKey === mountExtraDepsKey;

  const { data, isFetching, isFetchingNextPage, hasNextPage, fetchNextPage } = useInfiniteQuery({
    queryKey: ['filtered-products', endpoint, queryString, debouncedExtraDepsKey],
    queryFn: ({ pageParam }) => {
      const params = new URLSearchParams(queryString);
      params.set('limit', String(PRODUCTS_PAGE_SIZE));
      if (pageParam) params.set('cursor', pageParam);
      return fetchProductsPage(`${endpoint}?${params.toString()}`);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialData: canSeedSSR
      ? {
          pages: [{ products: initialProducts, nextCursor: initialCursor }],
          pageParams: [undefined],
        }
      : undefined,
    staleTime: 2 * 60 * 1000, // 2 minutes — same filters return cached results
    gcTime: 5 * 60 * 1000,
  });

  const products = data?.pages.flatMap((page) => page.products) ?? initialProducts;

  return {
    filters,
    setFilters,
    products,
    // Skeleton only when there is nothing to show — appending a page, and the
    // background refetch that freshens the SSR snapshot, keep the list up.
    loading: isFetching && !isFetchingNextPage && products.length === 0,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
  };
}
