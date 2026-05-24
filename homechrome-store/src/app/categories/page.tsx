import { Center, Container, SimpleGrid, Text } from '@mantine/core';
import type { Metadata } from 'next';

import CategoryCard from '@/components/catalog/CategoryCard';
import { PageHeader } from '@/components/ui/page-header';
import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category } from '@/types';

export const metadata: Metadata = {
  title: 'All Categories | Homechrome',
  description: 'Browse our collection of handloom textile categories.',
};

async function getCategories(): Promise<Category[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.CATEGORIES}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function CategoriesPage() {
  const categories = await getCategories();

  return (
    <Container size="xl" py="xl">
      <PageHeader
        title="All Categories"
        description="Explore our curated collections of handloom textiles."
      />

      {categories.length > 0 ? (
        <SimpleGrid cols={{ base: 2, sm: 3, lg: 4 }} spacing="md">
          {categories.map((category) => (
            <CategoryCard key={category.id} category={category} />
          ))}
        </SimpleGrid>
      ) : (
        <Center py="xl">
          <Text size="lg" c="dimmed">No categories available at the moment.</Text>
        </Center>
      )}
    </Container>
  );
}
