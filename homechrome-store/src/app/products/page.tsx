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

async function searchProducts(query: string): Promise<Product[]> {
  try {
    const url = new URL(`${API_BASE}/api/v1/store/catalog/products`);
    if (query) {
      url.searchParams.set('search', query);
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
  const products = await searchProducts(search);

  return <ProductsView products={products} initialSearch={search} />;
}
