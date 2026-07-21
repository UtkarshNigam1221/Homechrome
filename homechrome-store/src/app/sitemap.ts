import type { MetadataRoute } from 'next';

import { API_BASE, IS_INDEXABLE, SITE_URL } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category, Product } from '@/types';

async function getAllCategories(): Promise<Category[]> {
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

async function getAllProducts(): Promise<Product[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.PRODUCTS}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // Non-prod hosts are disallowed in robots; don't expose their URLs here either.
  if (!IS_INDEXABLE) return [];

  const [categories, products] = await Promise.all([getAllCategories(), getAllProducts()]);

  const staticRoutes: MetadataRoute.Sitemap = [
    {
      url: SITE_URL,
      lastModified: new Date(),
      changeFrequency: 'daily',
      priority: 1,
    },
    {
      url: `${SITE_URL}/products`,
      lastModified: new Date(),
      changeFrequency: 'daily',
      priority: 0.9,
    },
    {
      url: `${SITE_URL}/categories`,
      lastModified: new Date(),
      changeFrequency: 'daily',
      priority: 0.8,
    },
    {
      url: `${SITE_URL}/contact`,
      lastModified: new Date(),
      changeFrequency: 'monthly',
      priority: 0.3,
    },
    // Legal/policy pages — low priority, rarely change.
    ...['privacy-policy', 'terms', 'refund-policy', 'shipping-policy'].map((slug) => ({
      url: `${SITE_URL}/${slug}`,
      lastModified: new Date(),
      changeFrequency: 'yearly' as const,
      priority: 0.2,
    })),
  ];

  const categoryRoutes: MetadataRoute.Sitemap = categories.map((category) => ({
    url: `${SITE_URL}/c/${category.slug}`,
    lastModified: new Date(),
    changeFrequency: 'weekly' as const,
    priority: 0.7,
  }));

  const productRoutes: MetadataRoute.Sitemap = products.map((product) => ({
    url: `${SITE_URL}/p/${product.slug}`,
    lastModified: new Date(),
    changeFrequency: 'weekly' as const,
    priority: 0.6,
  }));

  return [...staticRoutes, ...categoryRoutes, ...productRoutes];
}
