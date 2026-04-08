import type { Metadata } from 'next';

import CategoryCard from '@/components/catalog/CategoryCard';
import { Container } from '@/components/ui/container';
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
    <Container className="py-10">
      <PageHeader
        title="All Categories"
        description="Explore our curated collections of handloom textiles."
      />

      {categories.length > 0 ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {categories.map((category) => (
            <CategoryCard key={category.id} category={category} />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-lg text-muted-foreground">No categories available at the moment.</p>
        </div>
      )}
    </Container>
  );
}
