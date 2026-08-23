import type { MetadataRoute } from 'next';

import { API_BASE, IS_INDEXABLE, SITE_URL } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category, Product } from '@/types';

// Cursor-paged catalog fetch. The API defaults to limit=20 and caps it at
// 100, so a single un-paged call silently truncates the sitemap.
async function fetchAllPages<T>(path: string): Promise<T[]> {
  const out: T[] = [];
  let cursor: string | undefined;
  try {
    // Bounded so a repeating next_cursor cannot spin the build forever.
    for (let page = 0; page < 100; page++) {
      const qs = new URLSearchParams({ limit: '100' });
      if (cursor) qs.set('cursor', cursor);
      const res = await fetch(`${API_BASE}${path}?${qs}`, {
        next: { revalidate: 3600 },
      });
      if (!res.ok) break;
      const json = await res.json();
      out.push(...(json.data ?? []));
      if (!json.meta?.has_more || !json.meta?.next_cursor) break;
      cursor = json.meta.next_cursor as string;
    }
  } catch {
    // Partial sitemap beats an empty one; static routes still ship.
  }
  return out;
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // Non-prod hosts are disallowed in robots; don't expose their URLs here either.
  if (!IS_INDEXABLE) return [];

  const [categories, products] = await Promise.all([
    fetchAllPages<Category>(ROUTES.CATALOG.CATEGORIES),
    fetchAllPages<Product>(ROUTES.CATALOG.PRODUCTS),
  ]);

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
