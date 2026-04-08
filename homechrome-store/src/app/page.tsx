import Link from 'next/link';

import CategoryCard from '@/components/catalog/CategoryCard';
import ProductCard from '@/components/catalog/ProductCard';
import { Container } from '@/components/ui/container';
import { Category, Product } from '@/types';

import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';

import HomePageTracker from './HomePageTracker';

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

export default async function HomePage() {
  const [categories, products] = await Promise.all([
    getCategories(),
    getFeaturedProducts(),
  ]);

  return (
    <div>
      <HomePageTracker />
      {/* Hero Section */}
      <section className="relative bg-foreground">
        <Container className="py-24 sm:py-32">
          <div className="text-center">
            <h1 className="text-4xl font-bold tracking-tight text-white sm:text-5xl lg:text-6xl">
              Handwoven with{' '}
              <span className="text-primary">tradition</span>
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-gray-300">
              Discover premium handloom textiles crafted with tradition
              across India. Each piece tells a story of heritage and
              craftsmanship.
            </p>
            <div className="mt-10 flex items-center justify-center gap-4">
              <Link
                href="/products"
                className="rounded-lg bg-primary px-8 py-3 text-base font-medium text-white transition-colors hover:bg-primary-dark"
              >
                Shop Now
              </Link>
              <Link
                href="/categories"
                className="rounded-lg border-2 border-white/30 px-8 py-3 text-base font-medium text-white transition-colors hover:border-white hover:bg-white/10"
              >
                Browse Categories
              </Link>
            </div>
          </div>
        </Container>
      </section>

      {/* Features Section */}
      <section className="bg-white py-16">
        <Container>
          <div className="grid grid-cols-1 gap-8 sm:grid-cols-3">
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                  className="h-6 w-6 text-primary"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-base font-semibold text-foreground">
                Handcrafted
              </h3>
              <p className="mt-2 text-sm text-muted">
                Each piece is handwoven by skilled weavers preserving
                centuries-old techniques.
              </p>
            </div>
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                  className="h-6 w-6 text-primary"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M8.25 18.75a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 0 1-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 0 0-3.213-9.193 2.056 2.056 0 0 0-1.58-.86H14.25M16.5 18.75h-2.25m0-11.177v-.958c0-.568-.422-1.048-.987-1.106a48.554 48.554 0 0 0-10.026 0 1.106 1.106 0 0 0-.987 1.106v7.635m12-6.677v6.677m0 4.5v-4.5m0 0h-12"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-base font-semibold text-foreground">
                Free Shipping
              </h3>
              <p className="mt-2 text-sm text-muted">
                Free delivery on all orders above Rs. 999 across India.
              </p>
            </div>
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                  className="h-6 w-6 text-primary"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-base font-semibold text-foreground">
                Quality Assured
              </h3>
              <p className="mt-2 text-sm text-muted">
                Authenticity guaranteed with GI-tagged handloom products.
              </p>
            </div>
          </div>
        </Container>
      </section>

      {/* Featured Categories */}
      {categories.length > 0 && (
        <section className="py-16">
          <Container>
            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-bold text-foreground sm:text-3xl">
                Shop by Category
              </h2>
              <Link
                href="/categories"
                className="text-sm font-medium text-primary transition-colors hover:text-primary-dark"
              >
                View All
              </Link>
            </div>
            <div className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {categories.slice(0, 8).map((category) => (
                <CategoryCard key={category.id} category={category} />
              ))}
            </div>
          </Container>
        </section>
      )}

      {/* Featured Products / New Arrivals */}
      {products.length > 0 && (
        <section className="bg-white py-16">
          <Container>
            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-bold text-foreground sm:text-3xl">
                New Arrivals
              </h2>
              <Link
                href="/products"
                className="text-sm font-medium text-primary transition-colors hover:text-primary-dark"
              >
                View All
              </Link>
            </div>
            <div className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {products.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          </Container>
        </section>
      )}
    </div>
  );
}
