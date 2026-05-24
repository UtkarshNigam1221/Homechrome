import type { Metadata } from 'next';

import ProductsView from './ProductsView';

import { API_BASE } from '@/lib/constants';
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

async function getProducts(search: string): Promise<Product[]> {
  try {
    const url = new URL(`${API_BASE}${ROUTES.CATALOG.PRODUCTS}`);
    if (search) url.searchParams.set('search', search);
    const res = await fetch(url.toString(), { next: { revalidate: 300 } });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function ProductsPage({ searchParams }: PageProps) {
  const sp = await searchParams;
  const search = typeof sp.search === 'string' ? sp.search : '';
  const products = await getProducts(search);

  return <ProductsView products={products} initialSearch={search} />;
}
