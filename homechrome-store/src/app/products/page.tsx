import type { Metadata } from 'next';

import ProductsView from './ProductsView';

import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Product } from '@/types';

export const metadata: Metadata = {
  title: 'Shop All Products | Homechrome',
  description: 'Browse and search our complete collection of handloom textiles.',
};

// Cache the default product list at the edge for 5 minutes.
// Filtered/search results are fetched client-side by ProductsView.
export const revalidate = 300;

async function getProducts(): Promise<Product[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.PRODUCTS}`, {
      next: { revalidate: 300 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function ProductsPage() {
  const products = await getProducts();

  return <ProductsView products={products} initialSearch="" />;
}
