'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import ProductGrid from '@/components/catalog/ProductGrid';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import api from '@/lib/api';
import { Product } from '@/types';

interface ProductsViewProps {
  products: Product[];
  initialSearch: string;
}

function parseFiltersFromParams(params: URLSearchParams): FilterValues {
  const minPrice = params.get('min_price');
  const maxPrice = params.get('max_price');
  const inStock = params.get('in_stock');

  return {
    minPrice: minPrice ? Number(minPrice) : null,
    maxPrice: maxPrice ? Number(maxPrice) : null,
    inStockOnly: inStock === 'true',
    attributeFilters: {},
  };
}

export default function ProductsView({ products: initialProducts, initialSearch }: ProductsViewProps) {
  const searchParams = useSearchParams();
  const [searchQuery, setSearchQuery] = useState(initialSearch);
  const [filters, setFilters] = useState<FilterValues>(() =>
    parseFiltersFromParams(searchParams),
  );
  const [products, setProducts] = useState<Product[]>(initialProducts);
  const [loading, setLoading] = useState(false);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const isInitialMount = useRef(true);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    track('page_view', {
      page_type: 'search',
      search_query: initialSearch || undefined,
    });
  }, [initialSearch]);

  useScrollDepth('products');

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const q = searchQuery.trim();
      const params = new URLSearchParams();
      if (q) params.set('search', q);
      if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
      if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
      if (filters.inStockOnly) params.set('in_stock', 'true');
      const qs = params.toString();
      const newUrl = `/products${qs ? `?${qs}` : ''}`;
      window.history.pushState(null, '', newUrl);
      // Trigger re-fetch by updating filters (search change triggers effect via searchQuery state)
      setFilters({ ...filters });
    },
    [searchQuery, filters],
  );

  // Sync input with URL changes on browser back/forward (popstate)
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      setSearchQuery(params.get('search') || '');
      setFilters(parseFiltersFromParams(params));
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  // Re-fetch products when filters change (debounced, with abort)
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;

    const timer = setTimeout(() => {
      const params = new URLSearchParams();
      if (searchQuery.trim()) params.set('search', searchQuery.trim());
      if (filters.minPrice !== null) params.set('min_price', String(filters.minPrice));
      if (filters.maxPrice !== null) params.set('max_price', String(filters.maxPrice));
      if (filters.inStockOnly) params.set('in_stock', 'true');

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
  }, [filters, searchQuery]);

  const handleFiltersChange = useCallback(
    (newFilters: FilterValues) => {
      setFilters(newFilters);
      // Update URL without triggering a Next.js navigation (avoids duplicate server request)
      const params = new URLSearchParams();
      if (searchQuery.trim()) params.set('search', searchQuery.trim());
      if (newFilters.minPrice !== null) params.set('min_price', String(newFilters.minPrice));
      if (newFilters.maxPrice !== null) params.set('max_price', String(newFilters.maxPrice));
      if (newFilters.inStockOnly) params.set('in_stock', 'true');
      const qs = params.toString();
      window.history.pushState(null, '', `/products${qs ? `?${qs}` : ''}`);
    },
    [searchQuery],
  );

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
          <li className="text-foreground">
            {initialSearch ? 'Search Results' : 'All Products'}
          </li>
        </ol>
      </nav>

      {/* Header with inline search */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-foreground">
          {initialSearch ? `Results for "${initialSearch}"` : 'All Products'}
        </h1>
        <p className="mt-1 text-sm text-muted">
          {products.length} {products.length === 1 ? 'product' : 'products'}
          {initialSearch ? ' found' : ''}
        </p>

        {/* Search bar */}
        <form onSubmit={handleSearch} className="mt-4 max-w-lg">
          <div className="relative">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search products..."
              className="w-full rounded-full border border-border bg-background py-2.5 pl-4 pr-10 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
            />
            <button
              type="submit"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-primary"
              aria-label="Search"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
                stroke="currentColor"
                className="h-5 w-5"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
                />
              </svg>
            </button>
          </div>
        </form>
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
        </button>
      </div>

      <div className="flex gap-8">
        {/* Sidebar - desktop */}
        <div className="hidden w-64 shrink-0 lg:block">
          <div className="sticky top-32 rounded-xl bg-white p-5 shadow-sm">
            <FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />
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
              <FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />
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
