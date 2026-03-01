import type { Metadata } from 'next';

import CategoryCard from '@/components/catalog/CategoryCard';
import { Category } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

export const metadata: Metadata = {
  title: 'All Categories | Homechrome',
  description: 'Browse our collection of handloom textile categories.',
};

async function getCategories(): Promise<Category[]> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/store/catalog/categories`, {
      next: { revalidate: 60 },
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
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-foreground">All Categories</h1>
        <p className="mt-2 text-muted">
          Explore our curated collections of handloom textiles.
        </p>
      </div>

      {categories.length > 0 ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {categories.map((category) => (
            <CategoryCard key={category.id} category={category} />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-lg text-muted">No categories available at the moment.</p>
        </div>
      )}
    </div>
  );
}
