import type { Metadata } from 'next';
import { cache } from 'react';

import ProductDetailView from './ProductDetailView';

import { API_BASE, SITE_URL } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Product } from '@/types';

export const revalidate = 3600;

interface PageProps {
  params: Promise<{ slug: string }>;
}

const getProduct = cache(async function getProduct(slug: string): Promise<Product | null> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.PRODUCT(slug)}`, {
      next: { revalidate: 3600 },
    });
    if (res.status === 404) return null;
    if (!res.ok) throw new Error(`Failed to load product (${res.status})`);
    const json = await res.json();
    return json.data || null;
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Failed to load')) {
      throw error;
    }
    return null;
  }
});

export async function generateStaticParams() {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.PRODUCTS}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return (json.data || []).map((p: Product) => ({ slug: p.slug }));
  } catch {
    return [];
  }
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { slug } = await params;
  const product = await getProduct(slug);
  if (!product) {
    return { title: 'Product Not Found | Homechrome' };
  }

  const ogImages = product.images?.[0]?.url ? [{ url: product.images[0].url }] : [];

  return {
    title: `${product.name} | Homechrome`,
    description: product.description || `Shop ${product.name} at Homechrome.`,
    openGraph: {
      title: `${product.name} | Homechrome`,
      description: product.description || `Shop ${product.name} at Homechrome.`,
      url: `${SITE_URL}/p/${slug}`,
      images: ogImages,
      type: 'website',
    },
  };
}

function ProductJsonLd({ product }: { product: Product }) {
  const priceInRupees = (product.selling_price / 100).toFixed(2);
  const imageUrls = product.images?.map((img) => img.url) || [];

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Product',
    name: product.name,
    description: product.description,
    image: imageUrls,
    sku: product.sku,
    brand: {
      '@type': 'Brand',
      name: 'Homechrome',
    },
    offers: {
      '@type': 'Offer',
      url: `${SITE_URL}/p/${product.slug}`,
      priceCurrency: 'INR',
      price: priceInRupees,
      availability: product.in_stock
        ? 'https://schema.org/InStock'
        : 'https://schema.org/OutOfStock',
    },
  };

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
    />
  );
}

export default async function ProductPage({ params }: PageProps) {
  const { slug } = await params;
  const product = await getProduct(slug);

  if (!product) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-foreground">Product Not Found</h1>
        <p className="mt-2 text-muted-foreground">The product you are looking for does not exist.</p>
      </div>
    );
  }

  return (
    <>
      <ProductJsonLd product={product} />
      <ProductDetailView product={product} />
    </>
  );
}
