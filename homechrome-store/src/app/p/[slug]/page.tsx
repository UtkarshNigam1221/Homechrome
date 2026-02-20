import type { Metadata } from 'next';

import ProductDetailView from './ProductDetailView';

import { Product } from '@/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface PageProps {
  params: Promise<{ slug: string }>;
}

async function getProduct(slug: string): Promise<Product | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/store/catalog/products/${slug}`, {
      next: { revalidate: 120 },
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
  return {
    title: `${product.name} | Homechrome`,
    description: product.description || `Shop ${product.name} at Homechrome.`,
  };
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

  return <ProductDetailView product={product} />;
}
