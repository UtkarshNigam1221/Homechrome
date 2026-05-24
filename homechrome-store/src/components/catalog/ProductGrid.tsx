'use client';

import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { SimpleGrid, Stack, Text, ThemeIcon, Title } from '@mantine/core';

import ProductCard from '@/components/catalog/ProductCard';
import { Product } from '@/types';

interface ProductGridProps {
  products: Product[];
}

export default function ProductGrid({ products }: ProductGridProps) {
  if (products.length === 0) {
    return (
      <Stack align="center" gap="xs" py="xl" ta="center">
        <ThemeIcon size={64} radius="xl" color="gray" variant="light">
          <MagnifyingGlassIcon width={32} height={32} strokeWidth={1} />
        </ThemeIcon>
        <Title order={3} size="md" mt="xs">No products found</Title>
        <Text size="sm" c="dimmed">
          Try adjusting your filters or browse our categories.
        </Text>
      </Stack>
    );
  }

  return (
    <SimpleGrid cols={{ base: 2, sm: 2, lg: 3 }} spacing="md">
      {products.map((product) => (
        <ProductCard key={product.id} product={product} />
      ))}
    </SimpleGrid>
  );
}
