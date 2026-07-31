'use client';

import { AdjustmentsHorizontalIcon } from '@heroicons/react/24/outline';
import {
  Badge,
  Box,
  Button,
  Card,
  Drawer,
  Flex,
  Group,
  ScrollArea,
} from '@mantine/core';
import { ReactNode, useEffect, useState } from 'react';

import ProductGrid from '@/components/catalog/ProductGrid';
import ProductGridSkeleton from '@/components/skeleton/ProductGridSkeleton';
import { Product } from '@/types';

interface ProductsBrowserProps {
  products: Product[];
  loading: boolean;
  filtersSidebar: ReactNode;
  activeFilterCount?: number;
  skeletonCount?: number;
  /** Infinite scroll — omit both for a static list. */
  hasMore?: boolean;
  onLoadMore?: () => void;
}

export function ProductsBrowser({
  products,
  loading,
  filtersSidebar,
  activeFilterCount = 0,
  skeletonCount = 8,
  hasMore = false,
  onLoadMore,
}: ProductsBrowserProps) {
  const [mobileOpen, setMobileOpen] = useState(false);
  // State, not a ref, so the observer effect re-runs when the sentinel mounts
  // (it only exists while `hasMore`, i.e. after the effect would have run).
  const [sentinel, setSentinel] = useState<HTMLDivElement | null>(null);

  // Load the next page when the sentinel nears the viewport. Re-created per page
  // (products.length): a still-mounted target reports no *change* in
  // intersection, and a fresh observer re-reports it, so paging continues.
  useEffect(() => {
    if (!sentinel || !hasMore || !onLoadMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) onLoadMore();
      },
      { rootMargin: '400px' }, // start fetching before the user hits the end
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [sentinel, hasMore, onLoadMore, products.length]);

  return (
    <>
      <Box mb="md" hiddenFrom="lg">
        <Button
          variant="outline"
          color="navy"
          size="sm"
          onClick={() => setMobileOpen(true)}
          leftSection={<AdjustmentsHorizontalIcon width={16} height={16} />}
        >
          <Group gap="xs" wrap="nowrap">
            <span>Filters</span>
            {activeFilterCount > 0 && (
              <Badge size="xs" color="brand" circle>
                {activeFilterCount}
              </Badge>
            )}
          </Group>
        </Button>
      </Box>

      <Flex gap="xl">
        <Box w={256} flex="none" visibleFrom="lg">
          <Card shadow="sm" padding="md" radius="lg" pos="sticky" top={128}>
            {filtersSidebar}
          </Card>
        </Box>

        <Drawer
          opened={mobileOpen}
          onClose={() => setMobileOpen(false)}
          position="right"
          size="xs"
          title="Filters"
          scrollAreaComponent={ScrollArea.Autosize}
        >
          {filtersSidebar}
        </Drawer>

        <Box flex={1}>
          {loading ? (
            <ProductGridSkeleton count={skeletonCount} />
          ) : (
            <>
              <ProductGrid products={products} />
              {/* mih: a 0-height target is unreliable to observe. */}
              {hasMore && <Box ref={setSentinel} mt="xl" mih={1} />}
            </>
          )}
        </Box>
      </Flex>
    </>
  );
}
