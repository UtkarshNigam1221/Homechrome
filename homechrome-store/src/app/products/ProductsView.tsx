'use client';

import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect } from 'react';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import { ProductsBrowser } from '@/components/catalog/ProductsBrowser';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';
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
  const currentSearch = searchParams.get('search') ?? initialSearch;

  const { filters, setFilters, products, loading } = useFilteredProducts({
    endpoint: ROUTES.CATALOG.PRODUCTS,
    initialProducts,
    initialFilters: parseFiltersFromParams(searchParams),
    skipInitialFetchWhenNoFilters: false,
    extraParams: () => {
      const p = new URLSearchParams();
      if (currentSearch.trim()) p.set('search', currentSearch.trim());
      return p;
    },
    extraDeps: [currentSearch],
  });

  useEffect(() => {
    track('page_view', {
      page_type: 'search',
      search_query: currentSearch || undefined,
    });
  }, [currentSearch]);

  useScrollDepth('products');

  const handleFiltersChange = useCallback(
    (newFilters: FilterValues) => {
      setFilters(newFilters);
      const params = filtersToParams(newFilters);
      if (currentSearch.trim()) params.set('search', currentSearch.trim());
      const qs = params.toString();
      window.history.pushState(null, '', `/products${qs ? `?${qs}` : ''}`);
    },
    [currentSearch, setFilters],
  );

  return (
    <Container py="xl">
      <Breadcrumb
        items={[
          { label: 'Home', href: '/' },
          { label: currentSearch ? 'Search Results' : 'All Products' },
        ]}
      />

      <PageHeader
        title={currentSearch ? `Results for "${currentSearch}"` : 'All Products'}
        description={`${products.length} ${products.length === 1 ? 'product' : 'products'}${currentSearch ? ' found' : ''}`}
      />

      <ProductsBrowser
        products={products}
        loading={loading}
        filtersSidebar={<FilterSidebar filters={filters} onFiltersChange={handleFiltersChange} />}
      />
    </Container>
  );
}
