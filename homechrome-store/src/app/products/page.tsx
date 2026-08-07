import type { Metadata } from 'next';

import ProductsView from './ProductsView';

import { API_BASE, PRODUCTS_PAGE_SIZE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Product } from '@/types';

export const metadata: Metadata = {
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

export default async function ProductsPage({ searchParams }: PageProps) {
  const sp = await searchParams;
  const search = typeof sp.search === 'string' ? sp.search : '';
  const { products, nextCursor } = await getProducts(search);

  return <ProductsView products={products} initialCursor={nextCursor} initialSearch={search} />;
}
