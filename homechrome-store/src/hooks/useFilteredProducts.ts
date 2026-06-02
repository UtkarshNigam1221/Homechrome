'use client';

import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';

import { FilterValues } from '@/components/catalog/FilterSidebar';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
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
  /** Initial product list (from SSR). */
  initialProducts: Product[];
  /** Initial filter state (parsed from URL on the server or client). */
  initialFilters: FilterValues;
  /**
   * When true, the hook uses initialProducts as placeholder data when no filters
   * are active. Use this when SSR already returned unfiltered products.
   */
  skipInitialFetchWhenNoFilters?: boolean;
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
  products: Product[];
  loading: boolean;
}

/**
 * Shared hook that encapsulates:
 * - React Query–backed product fetch with caching and deduplication
 * - Popstate listener to sync filter state on browser back/forward
 *
 * Both ProductsView and CategoryProductsView use this hook; they differ only
 * in the fetch endpoint, extra params, and whether the initial fetch is skipped.
 */
export function useFilteredProducts({
  endpoint,
  extraParams,
  initialProducts,
  initialFilters,
  skipInitialFetchWhenNoFilters = false,
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

  // Also debounce extra deps (e.g. searchQuery)
  const extraDepsKey = JSON.stringify(extraDeps);
  const [debouncedExtraDepsKey, setDebouncedExtraDepsKey] = useState(extraDepsKey);
  const extraDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    extraDebounceRef.current = setTimeout(() => setDebouncedExtraDepsKey(extraDepsKey), 300);
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
  const filtersActive = hasActiveFilters(debouncedFilters) || debouncedExtraDepsKey !== JSON.stringify([]);

  // Use initialProducts as placeholder data when no filters are active and
  // skipInitialFetchWhenNoFilters is set (avoids refetching what SSR provided).
  const useInitialData = skipInitialFetchWhenNoFilters && !filtersActive;

  // Search-aware routing: when the caller's extraParams includes a non-empty
  // `q` we hit the embedder /search endpoint (semantic + filters); otherwise
  // we hit the legacy /products endpoint (filters only). Both return the
  // same JSON envelope so the response handling below is identical.
  const params = new URLSearchParams(queryString);
  const hasSearchQuery = params.get('q') !== null && params.get('q')!.trim() !== '';
  const targetEndpoint = hasSearchQuery ? ROUTES.CATALOG.SEARCH : endpoint;

  const { data, isFetching } = useQuery<Product[]>({
    queryKey: ['filtered-products', targetEndpoint, queryString, debouncedExtraDepsKey],
    queryFn: async () => {
      const { data } = await api.get<Product[]>(`${targetEndpoint}?${queryString}`);
      return Array.isArray(data) ? data : [];
    },
    placeholderData: (prev) => prev ?? initialProducts,
    enabled: !useInitialData,
    staleTime: 2 * 60 * 1000, // 2 minutes — same filters return cached results
    gcTime: 5 * 60 * 1000,
  });

  const products = useInitialData ? initialProducts : (data ?? initialProducts);

  return {
    filters,
    setFilters,
    products,
    loading: isFetching && !useInitialData,
  };
}
