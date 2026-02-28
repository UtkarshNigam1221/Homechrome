'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import ProductGrid from '@/components/catalog/ProductGrid';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import { Product } from '@/types';

interface ProductsViewProps {
  products: Product[];
  initialSearch: string;
}

export default function ProductsView({ products, initialSearch }: ProductsViewProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [searchQuery, setSearchQuery] = useState(initialSearch);
  const [filters, setFilters] = useState<FilterValues>({
    minPrice: null,
    maxPrice: null,
    inStockOnly: false,
  });
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

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
      if (q) {
        router.push(`/products?search=${encodeURIComponent(q)}`);
      } else {
        router.push('/products');
      }
    },
    [searchQuery, router],
  );

  // Sync input with URL changes
  useEffect(() => {
    const urlSearch = searchParams.get('search') || '';
    setSearchQuery(urlSearch);
  }, [searchParams]);

  const filteredProducts = useMemo(() => {
    return products.filter((product) => {
      if (filters.inStockOnly && !product.in_stock) return false;
      if (filters.minPrice !== null && product.selling_price < filters.minPrice) return false;
      if (filters.maxPrice !== null && product.selling_price > filters.maxPrice) return false;
      return true;
    });
  }, [products, filters]);

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
          {filteredProducts.length} {filteredProducts.length === 1 ? 'product' : 'products'}
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
            <FilterSidebar filters={filters} onFiltersChange={setFilters} />
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
              <FilterSidebar filters={filters} onFiltersChange={setFilters} />
            </div>
          </div>
        )}

        {/* Products grid */}
        <div className="flex-1">
          <ProductGrid products={filteredProducts} />
        </div>
      </div>
    </div>
  );
}
