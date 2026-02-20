import type { Metadata } from 'next';

import CategoryProductsView from './CategoryProductsView';

import { Category, Product } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface PageProps {
  params: Promise<{ slug: string }>;
}

async function getCategory(slug: string): Promise<Category | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/store/catalog/categories/${slug}`, {
      next: { revalidate: 300 },
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  } catch {
    return null;
  }
}

async function getCategoryProducts(categoryId: string): Promise<Product[]> {
  try {
    const res = await fetch(
      `${API_BASE}/api/v1/store/catalog/products?category_id=${categoryId}`,
      { next: { revalidate: 300 } },
    );
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { slug } = await params;
  const category = await getCategory(slug);
  if (!category) {
    return { title: 'Category Not Found | Homechrome' };
  }
  return {
    title: `${category.name} | Homechrome`,
    description: category.description || `Browse ${category.name} handloom textiles at Homechrome.`,
  };
}

export default async function CategoryPage({ params }: PageProps) {
  const { slug } = await params;
  const category = await getCategory(slug);

  if (!category) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-foreground">Category Not Found</h1>
        <p className="mt-2 text-muted">The category you are looking for does not exist.</p>
      </div>
    );
  }

  const products = await getCategoryProducts(category.id);

  return <CategoryProductsView category={category} products={products} />;
}
