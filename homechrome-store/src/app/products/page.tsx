import type { Metadata } from 'next';

import ProductsView from './ProductsView';

import { API_BASE, PRODUCTS_PAGE_SIZE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category, CategoryAttribute, Product } from '@/types';

export const metadata: Metadata = {
  alternates: { canonical: '/products' },
  title: 'Shop All Products | Homechrome',
  description: 'Browse and search our complete collection of handloom textiles.',
};

export const revalidate = 300;

interface PageProps {
  searchParams: Promise<{ search?: string | string[] }>;
}

async function getProducts(search: string): Promise<{ products: Product[]; nextCursor?: string }> {
  try {
    // /search handles both filtered + filter-only listings now — empty q just
    // returns all active products in sort_order.
    const qs = new URLSearchParams({ limit: String(PRODUCTS_PAGE_SIZE) });
    if (search) qs.set('q', search);
    const url = `${API_BASE}${ROUTES.CATALOG.SEARCH}?${qs.toString()}`;
    const res = await fetch(url, { next: { revalidate: 300 } });
    if (!res.ok) return { products: [] };
    const json = await res.json();
    // next_cursor hands the client the offset to continue infinite scroll from.
    return { products: json.data || [], nextCursor: json.meta?.next_cursor || undefined };
  } catch {
    return { products: [] };
  }
}

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

// Attribute options are published per category, and this page spans all of
// them, so merge the sets. /search honours af_* filters without a category, so
// the merged list filters correctly even where an attribute belongs to one
// collection only.
async function getMergedFilterOptions(
  categories: Category[],
): Promise<Record<string, string[]>> {
  const perCategory = await Promise.all(
    categories.map(async (c) => {
      try {
        const res = await fetch(
          `${API_BASE}${ROUTES.CATALOG.FILTER_OPTIONS(c.id)}`,
          { next: { revalidate: 3600 } },
        );
        if (!res.ok) return {};
        const json = await res.json();
        return (json.data || {}) as Record<string, string[]>;
      } catch {
        return {};
      }
    }),
  );

  const merged: Record<string, Set<string>> = {};
  for (const options of perCategory) {
    for (const [name, values] of Object.entries(options)) {
      merged[name] ??= new Set();
      for (const v of values) merged[name].add(v);
    }
  }
  return Object.fromEntries(
    Object.entries(merged).map(([name, values]) => [name, [...values].sort()]),
  );
}

/** The catalogue publishes options, not definitions, so derive the headings. */
function attributesFromOptions(options: Record<string, string[]>): CategoryAttribute[] {
  return Object.keys(options)
    .sort()
    .map((name, i) => ({
      name,
      label: name.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
      type: 'text',
      required: false,
      searchable: true,
      display_order: i,
    }));
}

export default async function ProductsPage({ searchParams }: PageProps) {
  const sp = await searchParams;
  const search = typeof sp.search === 'string' ? sp.search : '';
  const [{ products, nextCursor }, categories] = await Promise.all([
    getProducts(search),
    getCategories(),
  ]);
  const filterOptions = await getMergedFilterOptions(categories);

  return (
    <ProductsView
      products={products}
      initialCursor={nextCursor}
      initialSearch={search}
      categories={categories}
      filterOptions={filterOptions}
      categoryAttributes={attributesFromOptions(filterOptions)}
    />
  );
}
