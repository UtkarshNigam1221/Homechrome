'use client';

import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import FilterSidebar, { FilterValues } from '@/components/catalog/FilterSidebar';
import { ProductsBrowser } from '@/components/catalog/ProductsBrowser';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';
import {
  filtersToParams,
  hasActiveFilters,
  parseFiltersFromParams,
  useFilteredProducts,
} from '@/hooks/useFilteredProducts';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { Category, CategoryAttribute, Product } from '@/types';

interface CategoryProductsViewProps {
  category: Category;
  products: Product[];
  initialCursor?: string;
}

export default function CategoryProductsView({
  category,
  products: initialProducts,
  initialCursor,
}: CategoryProductsViewProps) {
  const searchParams = useSearchParams();
  const [filterOptions, setFilterOptions] = useState<Record<string, string[]>>({});

  const { filters, setFilters, products, loading, hasMore, loadMore } =
    useFilteredProducts({
      endpoint: ROUTES.CATALOG.PRODUCTS,
      initialProducts,
      initialCursor,
      initialFilters: parseFiltersFromParams(searchParams),
      extraParams: () => {
        const p = new URLSearchParams();
        p.set('category_id', category.id);
        return p;
      },
      extraDeps: [category.id],
    });

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

  useEffect(() => {
    const controller = new AbortController();
    api
      .get<Record<string, string[]>>(
        ROUTES.CATALOG.FILTER_OPTIONS(category.id),
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

  const handleFiltersChange = useCallback(
    (newFilters: FilterValues) => {
      setFilters(newFilters);
      const urlParams = filtersToParams(newFilters);
      const qs = urlParams.toString();
      const newUrl = `/c/${category.slug}${qs ? `?${qs}` : ''}`;
      window.history.pushState(null, '', newUrl);
    },
    [category.slug, setFilters],
  );

  const categoryAttributes: CategoryAttribute[] = category.own_attributes || [];

  // Unfiltered: the category's own count is the real total (the list holds only
  // the pages loaded so far). Filtered: what's loaded, "+" while more remain.
  const filtered = hasActiveFilters(filters);
  const count = filtered ? products.length : category.product_count;

  const activeFilterCount = hasActiveFilters(filters)
    ? Object.keys(filters.attributeFilters).length +
      (filters.minPrice !== null || filters.maxPrice !== null ? 1 : 0) +
      (filters.inStockOnly ? 1 : 0)
    : 0;

  return (
    <Container py="xl">
      <Breadcrumb
        items={[
          { label: 'Home', href: '/' },
          { label: 'Categories', href: '/categories' },
          { label: category.name },
        ]}
      />

      <PageHeader
        title={category.name}
        description={`${category.description ? category.description + ' · ' : ''}${count}${filtered && hasMore ? '+' : ''} ${count === 1 ? 'product' : 'products'}`}
      />

      <ProductsBrowser
        products={products}
        loading={loading}
        hasMore={hasMore}
        onLoadMore={loadMore}
        activeFilterCount={activeFilterCount}
        filtersSidebar={
          <FilterSidebar
            filters={filters}
            onFiltersChange={handleFiltersChange}
            filterOptions={filterOptions}
            categoryAttributes={categoryAttributes}
          />
        }
      />
    </Container>
  );
}
