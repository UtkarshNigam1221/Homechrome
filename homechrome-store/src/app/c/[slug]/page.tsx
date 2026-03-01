import type { Metadata } from 'next';

import CategoryProductsView from './CategoryProductsView';

import { Category, Product } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || 'https://homechrome.lldlab.com';

interface PageProps {
  params: Promise<{ slug: string }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}

async function getCategory(slug: string): Promise<Category | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/store/catalog/categories/${slug}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  } catch {
    return null;
  }
}

async function getCategoryProducts(
  categoryId: string,
  filterParams: Record<string, string>,
): Promise<Product[]> {
  try {
    const url = new URL(`${API_BASE}/api/v1/store/catalog/products`);
    url.searchParams.set('category_id', categoryId);
    for (const [key, value] of Object.entries(filterParams)) {
      if (value) url.searchParams.set(key, value);
    }
    const res = await fetch(url.toString(), { next: { revalidate: 60 } });
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
      dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
    />
  );
}

export default async function CategoryPage({ params, searchParams }: PageProps) {
  const { slug } = await params;
  const sp = await searchParams;
  const category = await getCategory(slug);

  if (!category) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-foreground">Category Not Found</h1>
        <p className="mt-2 text-muted">The category you are looking for does not exist.</p>
      </div>
    );
  }

  // Forward filter params to the server fetch
  const filterParams: Record<string, string> = {};
  if (typeof sp.min_price === 'string') filterParams.min_price = sp.min_price;
  if (typeof sp.max_price === 'string') filterParams.max_price = sp.max_price;
  if (typeof sp.in_stock === 'string') filterParams.in_stock = sp.in_stock;
  // Forward af_* attribute filter params
  for (const [key, value] of Object.entries(sp)) {
    if (key.startsWith('af_') && typeof value === 'string') {
      filterParams[key] = value;
    }
  }

  const products = await getCategoryProducts(category.id, filterParams);

  return (
    <>
      <BreadcrumbJsonLd category={category} />
      <CategoryProductsView category={category} products={products} />
    </>
  );
}
