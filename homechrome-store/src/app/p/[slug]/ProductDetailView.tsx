'use client';

import { Box, Container, SimpleGrid } from '@mantine/core';

import { Breadcrumb } from '@/components/ui/breadcrumb';
import { useProductGallery } from '@/hooks/useProductGallery';
import { useProductTracking } from '@/hooks/useProductTracking';
import { Product } from '@/types';

import { ProductGallery } from './ProductGallery';
import { ProductInfo } from './ProductInfo';

interface ProductDetailViewProps {
  product: Product;
}

export default function ProductDetailView({ product }: ProductDetailViewProps) {
  useProductTracking(product);
  const { items, selectedIndex, selectedItem, setSelectedIndex } = useProductGallery(product);

  return (
    <Container size="xl" py="xl">
      <Breadcrumb
        items={[
          { label: 'Home', href: '/' },
          { label: 'Products', href: '/products' },
          { label: product.name },
        ]}
      />

      <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="xl">
        {/* Sticky left column: gallery pins while the taller right column scrolls,
            then releases at the row end so both scroll together. */}
        <Box
          pos={{ base: 'static', lg: 'sticky' }}
          top="calc(var(--app-header-h) + var(--mantine-spacing-md))"
          style={{ alignSelf: 'start' }}
        >
          <ProductGallery
            productName={product.name}
            items={items}
            selectedIndex={selectedIndex}
            selectedItem={selectedItem}
            onSelect={setSelectedIndex}
          />
        </Box>
        <ProductInfo product={product} />
      </SimpleGrid>
    </Container>
  );
}
