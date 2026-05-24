'use client';

import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { Box } from '@mantine/core';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import { ProductsBrowser } from '@/components/catalog/ProductsBrowser';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';
import { SearchInput } from '@/components/ui/search-input';
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

  const { filters, setFilters, products, loading } = useFilteredProducts({
    endpoint: ROUTES.CATALOG.PRODUCTS,
    initialProducts,
    initialFilters: parseFiltersFromParams(searchParams),
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
      window.history.pushState(null, '', `/products${qs ? `?${qs}` : ''}`);
      setFilters((f) => ({ ...f }));
    },
    [searchQuery, filters, setFilters],
  );

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
      const params = filtersToParams(newFilters);
      if (searchQuery.trim()) params.set('search', searchQuery.trim());
      const qs = params.toString();
      window.history.pushState(null, '', `/products${qs ? `?${qs}` : ''}`);
    },
    [searchQuery, setFilters],
  );

  return (
    <Container py="xl">
      <Breadcrumb
        items={[
          { label: 'Home', href: '/' },
          { label: initialSearch ? 'Search Results' : 'All Products' },
        ]}
      />

      <PageHeader
        title={initialSearch ? `Results for "${initialSearch}"` : 'All Products'}
        description={`${products.length} ${products.length === 1 ? 'product' : 'products'}${initialSearch ? ' found' : ''}`}
      >
        <Box mt="md" maw={512}>
          <SearchInput
            value={searchQuery}
            onChange={setSearchQuery}
            onSubmit={handleSearch}
            placeholder="Search products..."
          />
        </Box>
      </PageHeader>

      <ProductsBrowser
        products={products}
        loading={loading}
        filtersSidebar={<FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />}
      />
    </Container>
  );
}
