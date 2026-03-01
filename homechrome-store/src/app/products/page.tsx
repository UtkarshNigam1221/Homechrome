import type { Metadata } from 'next';

import ProductsView from './ProductsView';

import { Product } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

export const metadata: Metadata = {
  title: 'Shop All Products | Homechrome',
  description: 'Browse and search our complete collection of handloom textiles.',
};

interface PageProps {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}

async function searchProducts(
  query: string,
  filterParams: Record<string, string>,
): Promise<Product[]> {
  try {
    const url = new URL(`${API_BASE}/api/v1/store/catalog/products`);
    if (query) url.searchParams.set('search', query);
    for (const [key, value] of Object.entries(filterParams)) {
      if (value) url.searchParams.set(key, value);
    }
    const res = await fetch(url.toString(), {
      next: { revalidate: 60 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function ProductsPage({ searchParams }: PageProps) {
  const params = await searchParams;
  const search = typeof params.search === 'string' ? params.search : '';

  const filterParams: Record<string, string> = {};
  if (typeof params.min_price === 'string') filterParams.min_price = params.min_price;
  if (typeof params.max_price === 'string') filterParams.max_price = params.max_price;
  if (typeof params.in_stock === 'string') filterParams.in_stock = params.in_stock;

  const products = await searchProducts(search, filterParams);

  return <ProductsView products={products} initialSearch={search} />;
}
