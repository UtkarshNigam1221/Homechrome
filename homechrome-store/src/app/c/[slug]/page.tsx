import type { Metadata } from 'next';
import { cache, Suspense } from 'react';

import CategoryProductsView from './CategoryProductsView';

import { API_BASE, SITE_URL } from '@/lib/constants';
import { safeJsonLd } from '@/lib/jsonld';
import { ROUTES } from '@/lib/routes';
import { Category, Product } from '@/types';

export const revalidate = 3600;

interface PageProps {
  params: Promise<{ slug: string }>;
}

const getCategory = cache(async function getCategory(slug: string): Promise<Category | null> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.CATEGORY(slug)}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  } catch {
    return null;
  }
});

const getCategoryProducts = cache(async function getCategoryProducts(categoryId: string): Promise<Product[]> {
  try {
    const res = await fetch(
      `${API_BASE}${ROUTES.CATALOG.PRODUCTS}?category_id=${categoryId}`,
      { next: { revalidate: 3600 } },
    );
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
});

export async function generateStaticParams() {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.CATEGORIES}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return (json.data || []).map((c: Category) => ({ slug: c.slug }));
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
    openGraph: {
      title: `${category.name} | Homechrome`,
      description:
        category.description || `Browse ${category.name} handloom textiles at Homechrome.`,
      url: `${SITE_URL}/c/${slug}`,
      type: 'website',
    },
  };
}

function BreadcrumbJsonLd({ category }: { category: Category }) {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Home',
        item: SITE_URL,
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Categories',
        item: `${SITE_URL}/categories`,
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: category.name,
        item: `${SITE_URL}/c/${category.slug}`,
      },
    ],
  };

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: safeJsonLd(jsonLd) }}
    />
  );
}

export default async function CategoryPage({ params }: PageProps) {
  const { slug } = await params;
  const category = await getCategory(slug);

  if (!category) {
    return (
      <div style={{ maxWidth: 1280, margin: '0 auto', padding: '64px 16px', textAlign: 'center' }}>
        <h1 style={{ fontSize: 24, fontWeight: 700 }}>Category Not Found</h1>
        <p style={{ marginTop: 8, color: 'var(--mantine-color-dimmed)' }}>
          The category you are looking for does not exist.
        </p>
      </div>
    );
  }

  const products = await getCategoryProducts(category.id);

  return (
    <>
      <BreadcrumbJsonLd category={category} />
      <Suspense>
        <CategoryProductsView category={category} products={products} />
      </Suspense>
    </>
  );
}
