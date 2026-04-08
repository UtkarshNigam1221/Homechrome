'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { AdjustmentsHorizontalIcon } from '@heroicons/react/24/outline';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import ProductGrid from '@/components/catalog/ProductGrid';
import ProductGridSkeleton from '@/components/skeleton/ProductGridSkeleton';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';
import { SearchInput } from '@/components/ui/search-input';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import {
  filtersToParams,
  parseFiltersFromParams,
  useFilteredProducts,
} from '@/hooks/useFilteredProducts';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import { ROUTES } from '@/lib/routes';
import { Product } from '@/types';

interface ProductsViewProps {
  products: Product[];
  initialSearch: string;
}

export default function ProductsView({ products: initialProducts, initialSearch }: ProductsViewProps) {
  const searchParams = useSearchParams();
  const [searchQuery, setSearchQuery] = useState(initialSearch);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

  const { filters, setFilters, products, loading } = useFilteredProducts({
    endpoint: ROUTES.CATALOG.PRODUCTS,
    initialProducts,
    initialFilters: parseFiltersFromParams(searchParams),
    // ProductsView always re-fetches on change (no SSR unfiltered shortcut)
    skipInitialFetchWhenNoFilters: false,
    extraParams: () => {
      const p = new URLSearchParams();
      if (searchQuery.trim()) p.set('search', searchQuery.trim());
      return p;
    },
    extraDeps: [searchQuery],
  });

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
      const params = filtersToParams(filters);
      if (q) params.set('search', q);
      const qs = params.toString();
      const newUrl = `/products${qs ? `?${qs}` : ''}`;
      window.history.pushState(null, '', newUrl);
      // Trigger re-fetch by spreading filters to produce a new reference
      setFilters((f) => ({ ...f }));
    },
    [searchQuery, filters, setFilters],
  );

  // Sync search input with URL on browser back/forward — the hook handles filter
  // state via its own popstate listener; we only need to update searchQuery here.
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      setSearchQuery(params.get('search') || '');
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const handleFiltersChange = useCallback(
    (newFilters: FilterValues) => {
      setFilters(newFilters);
      // Update URL without triggering a Next.js navigation (avoids duplicate server request)
      const params = filtersToParams(newFilters);
      if (searchQuery.trim()) params.set('search', searchQuery.trim());
      const qs = params.toString();
      window.history.pushState(null, '', `/products${qs ? `?${qs}` : ''}`);
    },
    [searchQuery, setFilters],
  );

  return (
    <Container className="py-10">
      {/* Breadcrumb */}
      <nav className="mb-6 text-sm text-muted-foreground">
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

      <PageHeader
        title={initialSearch ? `Results for "${initialSearch}"` : 'All Products'}
        description={`${products.length} ${products.length === 1 ? 'product' : 'products'}${initialSearch ? ' found' : ''}`}
      >

        <SearchInput
          value={searchQuery}
          onChange={setSearchQuery}
          onSubmit={handleSearch}
          placeholder="Search products..."
          className="mt-4 max-w-lg"
        />
      </PageHeader>

      {/* Mobile filter toggle */}
      <div className="mb-4 lg:hidden">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setMobileFiltersOpen(!mobileFiltersOpen)}
        >
          <AdjustmentsHorizontalIcon className="h-4 w-4" />
          Filters
        </Button>
      </div>

      <div className="flex gap-8">
        {/* Sidebar - desktop */}
        <div className="hidden w-64 shrink-0 lg:block">
          <Card className="sticky top-32">
            <CardContent>
              <FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />
            </CardContent>
          </Card>
        </div>

        {/* Mobile filter panel */}
        <Sheet open={mobileFiltersOpen} onOpenChange={setMobileFiltersOpen}>
          <SheetContent side="right" className="flex w-full max-w-xs flex-col">
            <SheetHeader>
              <SheetTitle>Filters</SheetTitle>
            </SheetHeader>
            <ScrollArea className="flex-1">
              <div className="px-4 pb-4">
                <FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />
              </div>
            </ScrollArea>
          </SheetContent>
        </Sheet>

        {/* Products grid */}
        <div className="flex-1">
          {loading ? (
            <ProductGridSkeleton count={8} />
          ) : (
            <ProductGrid products={products} />
          )}
        </div>
      </div>
    </Container>
  );
}
