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
import { ReactNode, useState } from 'react';

import ProductGrid from '@/components/catalog/ProductGrid';
import ProductGridSkeleton from '@/components/skeleton/ProductGridSkeleton';
import { Product } from '@/types';

interface ProductsBrowserProps {
  products: Product[];
  loading: boolean;
  filtersSidebar: ReactNode;
  activeFilterCount?: number;
  skeletonCount?: number;
}

export function ProductsBrowser({
  products,
  loading,
  filtersSidebar,
  activeFilterCount = 0,
  skeletonCount = 8,
}: ProductsBrowserProps) {
  const [mobileOpen, setMobileOpen] = useState(false);

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
            <ProductGrid products={products} />
          )}
        </Box>
      </Flex>
    </>
  );
}
