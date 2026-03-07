'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import ProductGrid from '@/components/catalog/ProductGrid';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import api from '@/lib/api';
import { Category, CategoryAttribute, Product } from '@/types';

interface CategoryProductsViewProps {
  category: Category;
  products: Product[];
}

function parseFiltersFromParams(params: URLSearchParams): FilterValues {
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

function filtersToParams(filters: FilterValues): URLSearchParams {
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

const emptyFilters: FilterValues = {
  minPrice: null,
  maxPrice: null,
  inStockOnly: false,
  attributeFilters: {},
};

function hasActiveFilters(filters: FilterValues): boolean {
  return (
    filters.minPrice !== null ||
    filters.maxPrice !== null ||
    filters.inStockOnly ||
    Object.keys(filters.attributeFilters).length > 0
  );
}

export default function CategoryProductsView({
  category,
  products: initialProducts,
}: CategoryProductsViewProps) {
  const searchParams = useSearchParams();
  const [filters, setFilters] = useState<FilterValues>(() =>
    parseFiltersFromParams(searchParams),
  );
  const [products, setProducts] = useState<Product[]>(initialProducts);
  const [loading, setLoading] = useState(false);
  const [filterOptions, setFilterOptions] = useState<Record<string, string[]>>({});
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const isInitialMount = useRef(true);
  const isProgrammaticNav = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    track('page_view', {
      page_type: 'category',
      category_slug: category.slug,
      utm_source: params.get('utm_source') || undefined,
      utm_medium: params.get('utm_medium') || undefined,
      utm_campaign: params.get('utm_campaign') || undefined,
    });
  }, [category.slug]);

  useScrollDepth('category');

  // Fetch filter options on mount
  useEffect(() => {
    const controller = new AbortController();
    api
      .get<Record<string, string[]>>(
        `/api/v1/store/catalog/products/filter-options/${category.id}`,
        { signal: controller.signal },
      )
      .then((res) => {
        if (!controller.signal.aborted) setFilterOptions(res.data);
      })
      .catch(() => {
        // silently ignore — filters just won't show attribute options
      });
    return () => controller.abort();
  }, [category.id]);

  // Sync URL → filter state only on browser back/forward (not programmatic pushes)
  useEffect(() => {
    if (isProgrammaticNav.current) {
      isProgrammaticNav.current = false;
      return;
    }
    const parsed = parseFiltersFromParams(searchParams);
    setFilters(parsed);
  }, [searchParams]);

  // Re-fetch products when filters change (debounced, with abort)
  // On initial mount: skip if no active filters (server already provided unfiltered products),
  // but fetch immediately if URL has filters (server doesn't know about them).
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      if (!hasActiveFilters(filters)) return;
    }

    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;

    const timer = setTimeout(() => {
      const params = new URLSearchParams();
      params.set('category_id', category.id);
      if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
      if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
      if (filters.inStockOnly) params.set('in_stock', 'true');
      for (const [name, values] of Object.entries(filters.attributeFilters)) {
        if (values.length > 0) {
          params.set(`af_${name}`, values.join(','));
        }
      }

      setLoading(true);
      api
        .get<Product[]>(`/api/v1/store/catalog/products?${params.toString()}`)
        .then((res) => {
          if (!controller.signal.aborted) {
            setProducts(Array.isArray(res.data) ? res.data : []);
          }
        })
        .catch(() => {
          // keep existing products on error
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            setLoading(false);
          }
        });
    }, 300);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [category.id, filters]);

  const handleFiltersChange = useCallback(
    (newFilters: FilterValues) => {
      isProgrammaticNav.current = true;
      setFilters(newFilters);
      // Update URL without triggering a Next.js navigation (avoids duplicate server request)
      const urlParams = filtersToParams(newFilters);
      const qs = urlParams.toString();
      const newUrl = `/c/${category.slug}${qs ? `?${qs}` : ''}`;
      window.history.pushState(null, '', newUrl);
    },
    [category.slug],
  );

  const categoryAttributes: CategoryAttribute[] = category.own_attributes || [];

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav className="mb-6 text-sm text-muted">
        <ol className="flex items-center gap-2">
          <li>
            <Link href="/" className="hover:text-primary">
              Home
            </Link>
          </li>
          <li>/</li>
          <li>
            <Link href="/categories" className="hover:text-primary">
              Categories
            </Link>
          </li>
          <li>/</li>
          <li className="text-foreground">{category.name}</li>
        </ol>
      </nav>

      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-foreground">{category.name}</h1>
        {category.description && (
          <p className="mt-2 text-muted">{category.description}</p>
        )}
        <p className="mt-1 text-sm text-muted">
          {products.length} {products.length === 1 ? 'product' : 'products'}
        </p>
      </div>

      {/* Mobile filter toggle */}
      <div className="mb-4 lg:hidden">
        <button
          type="button"
          onClick={() => setMobileFiltersOpen(!mobileFiltersOpen)}
          className="flex items-center gap-2 rounded-lg border border-border bg-white px-4 py-2 text-sm font-medium text-foreground"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
            className="h-4 w-4"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75"
            />
          </svg>
          Filters
          {hasActiveFilters(filters) && (
            <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-xs text-white">
              {Object.keys(filters.attributeFilters).length +
                (filters.minPrice !== null || filters.maxPrice !== null ? 1 : 0) +
                (filters.inStockOnly ? 1 : 0)}
            </span>
          )}
        </button>
      </div>

      <div className="flex gap-8">
        {/* Sidebar - desktop */}
        <div className="hidden w-64 shrink-0 lg:block">
          <div className="sticky top-32 rounded-xl bg-white p-5 shadow-sm">
            <FilterSidebar
              filters={filters}
              onFiltersChange={handleFiltersChange}
              filterOptions={filterOptions}
              categoryAttributes={categoryAttributes}
            />
          </div>
        </div>

        {/* Mobile filter panel */}
        {mobileFiltersOpen && (
          <div className="fixed inset-0 z-50 lg:hidden">
            <div
              className="fixed inset-0 bg-black/40"
              onClick={() => setMobileFiltersOpen(false)}
            />
            <div className="fixed inset-y-0 right-0 w-full max-w-xs bg-white p-6 shadow-xl">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-foreground">Filters</h2>
                <button
                  type="button"
                  onClick={() => setMobileFiltersOpen(false)}
                  className="text-foreground"
                  aria-label="Close filters"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={1.5}
                    stroke="currentColor"
                    className="h-6 w-6"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M6 18 18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>
              <FilterSidebar
                filters={filters}
                onFiltersChange={handleFiltersChange}
                filterOptions={filterOptions}
                categoryAttributes={categoryAttributes}
              />
            </div>
          </div>
        )}

        {/* Products grid */}
        <div className="flex-1">
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
            </div>
          ) : (
            <ProductGrid products={products} />
          )}
        </div>
      </div>
    </div>
  );
}
