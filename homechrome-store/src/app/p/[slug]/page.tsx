import type { Metadata } from 'next';

import ProductDetailView from './ProductDetailView';

import { Product } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || 'https://homechrome.lldlab.com';

interface PageProps {
  params: Promise<{ slug: string }>;
}

async function getProduct(slug: string): Promise<Product | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/store/catalog/products/${slug}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  } catch {
    return null;
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
        <p className="mt-2 text-muted">The product you are looking for does not exist.</p>
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
