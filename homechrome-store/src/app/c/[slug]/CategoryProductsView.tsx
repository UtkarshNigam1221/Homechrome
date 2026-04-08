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
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
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
}

export default function CategoryProductsView({
  category,
  products: initialProducts,
}: CategoryProductsViewProps) {
  const searchParams = useSearchParams();
  const [filterOptions, setFilterOptions] = useState<Record<string, string[]>>({});
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

  const { filters, setFilters, products, loading } = useFilteredProducts({
    endpoint: ROUTES.CATALOG.PRODUCTS,
    initialProducts,
    initialFilters: parseFiltersFromParams(searchParams),
    // Skip initial fetch when no filters active: SSR already returned unfiltered products.
    // But fetch immediately if URL has filters (server doesn't apply them).
    skipInitialFetchWhenNoFilters: true,
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

  // Fetch filter options on mount
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
      // Update URL without triggering a Next.js navigation (avoids duplicate server request)
      const urlParams = filtersToParams(newFilters);
      const qs = urlParams.toString();
      const newUrl = `/c/${category.slug}${qs ? `?${qs}` : ''}`;
      window.history.pushState(null, '', newUrl);
    },
    [category.slug, setFilters],
  );

  const categoryAttributes: CategoryAttribute[] = category.own_attributes || [];

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
          <li>
            <Link href="/categories" className="hover:text-primary">
              Categories
            </Link>
          </li>
          <li>/</li>
          <li className="text-foreground">{category.name}</li>
        </ol>
      </nav>

      <PageHeader
        title={category.name}
        description={`${category.description ? category.description + ' · ' : ''}${products.length} ${products.length === 1 ? 'product' : 'products'}`}
      />

      {/* Mobile filter toggle */}
      <div className="mb-4 lg:hidden">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setMobileFiltersOpen(!mobileFiltersOpen)}
        >
          <AdjustmentsHorizontalIcon className="h-4 w-4" />
          Filters
          {hasActiveFilters(filters) && (
            <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-xs text-white">
              {Object.keys(filters.attributeFilters).length +
                (filters.minPrice !== null || filters.maxPrice !== null ? 1 : 0) +
                (filters.inStockOnly ? 1 : 0)}
            </span>
          )}
        </Button>
      </div>

      <div className="flex gap-8">
        {/* Sidebar - desktop */}
        <div className="hidden w-64 shrink-0 lg:block">
          <Card className="sticky top-32">
            <CardContent>
            <FilterSidebar
              filters={filters}
              onFiltersChange={handleFiltersChange}
              filterOptions={filterOptions}
              categoryAttributes={categoryAttributes}
            />
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
              <FilterSidebar
                filters={filters}
                onFiltersChange={handleFiltersChange}
                filterOptions={filterOptions}
                categoryAttributes={categoryAttributes}
              />
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
