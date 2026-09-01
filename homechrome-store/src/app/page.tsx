import type { Metadata } from 'next';

import { Category, Product, PublicCoupon } from '@/types';

import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';

import HomeView from './HomeView';

export const metadata: Metadata = {
  alternates: { canonical: '/' },
};

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

async function getFeaturedProducts(): Promise<Product[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.PRODUCTS}?limit=8`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

async function getPublicCoupons(): Promise<PublicCoupon[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.COUPONS}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function HomePage() {
  const [categories, products, coupons] = await Promise.all([
    getCategories(),
    getFeaturedProducts(),
    getPublicCoupons(),
  ]);

  return <HomeView categories={categories} products={products} coupons={coupons} />;
}
