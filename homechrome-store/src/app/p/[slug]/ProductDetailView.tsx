'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useEffect, useState } from 'react';

import Button from '@/components/common/Button';
import { useCart } from '@/hooks/useCart';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import { formatPrice } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { Product, ProductImage } from '@/types';

interface ProductDetailViewProps {
  product: Product;
}

export default function ProductDetailView({ product }: ProductDetailViewProps) {
  const sortedImages = [...(product.images || [])].sort((a, b) => {
    if (a.is_primary && !b.is_primary) return -1;
    if (!a.is_primary && b.is_primary) return 1;
    return a.sort_order - b.sort_order;
  });

  const [selectedImage, setSelectedImage] = useState<ProductImage | null>(
    sortedImages[0] || null,
  );
  const [quantity, setQuantity] = useState(1);
  const [adding, setAdding] = useState(false);
  const { addItem } = useCart();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  useEffect(() => {
    track('product_viewed', {
      product_id: product.id,
      product_name: product.name,
      category_id: product.category_id,
      price: product.selling_price,
    });
  }, [product.id, product.name, product.category_id, product.selling_price]);

  useScrollDepth('product');

  const hasDiscount = product.mrp > product.selling_price;
  const discountPercent = hasDiscount
    ? Math.round(((product.mrp - product.selling_price) / product.mrp) * 100)
    : 0;

  const handleAddToCart = async () => {
    if (!isAuthenticated) {
      window.location.href = '/login?redirect=' + encodeURIComponent(`/p/${product.slug}`);
      return;
    }
    setAdding(true);
    try {
      await addItem(product.id, quantity);
    } catch {
      // ignore
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav className="mb-6 text-sm text-muted">
        <ol className="flex items-center gap-2">
          <li>
            <Link href="/" className="hover:text-primary">
              Home
            </Link>
          </li>
          <li>/</li>
          <li>
            <Link href="/products" className="hover:text-primary">
              Products
            </Link>
          </li>
          <li>/</li>
          <li className="truncate text-foreground">{product.name}</li>
        </ol>
      </nav>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-2">
        {/* Image Gallery */}
        <div>
          {/* Main Image */}
          <div className="relative aspect-square overflow-hidden rounded-xl bg-gray-100">
            {selectedImage ? (
              <Image
                src={selectedImage.url}
                alt={selectedImage.alt_text || product.name}
                fill
                sizes="(max-width: 1024px) 100vw, 50vw"
                className="object-cover"
                priority
              />
            ) : (
              <div className="flex h-full items-center justify-center bg-primary-light/30">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1}
                  stroke="currentColor"
                  className="h-16 w-16 text-primary/40"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z"
                  />
                </svg>
              </div>
            )}
          </div>

          {/* Thumbnails */}
          {sortedImages.length > 1 && (
            <div className="mt-4 flex gap-3 overflow-x-auto pb-2">
              {sortedImages.map((img, index) => (
                <button
                  key={index}
                  type="button"
                  onClick={() => setSelectedImage(img)}
                  className={`relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-lg border-2 transition-colors ${
                    selectedImage === img
                      ? 'border-primary'
                      : 'border-transparent hover:border-border'
                  }`}
                >
                  <Image
                    src={img.url}
                    alt={img.alt_text || `${product.name} ${index + 1}`}
                    fill
                    sizes="80px"
                    className="object-cover"
                  />
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Product Info */}
        <div>
          <h1 className="text-2xl font-bold text-foreground sm:text-3xl">
            {product.name}
          </h1>

          {product.sku && (
            <p className="mt-1 text-sm text-muted">SKU: {product.sku}</p>
          )}

          {/* Price */}
          <div className="mt-4 flex items-baseline gap-3">
            <span className="text-3xl font-bold text-foreground">
              {formatPrice(product.selling_price)}
            </span>
            {hasDiscount && (
              <>
                <span className="text-lg text-muted line-through">
                  {formatPrice(product.mrp)}
                </span>
                <span className="rounded-md bg-red-100 px-2 py-0.5 text-sm font-semibold text-red-600">
                  -{discountPercent}%
                </span>
              </>
            )}
          </div>

          {/* Stock status */}
          <div className="mt-4">
            {product.in_stock ? (
              <span className="inline-flex items-center gap-1.5 text-sm font-medium text-green-600">
                <span className="h-2 w-2 rounded-full bg-green-500" />
                In Stock
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 text-sm font-medium text-red-500">
                <span className="h-2 w-2 rounded-full bg-red-500" />
                Out of Stock
              </span>
            )}
          </div>

          {/* Description */}
          {product.description && (
            <div className="mt-6">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground">
                Description
              </h2>
              <p className="mt-2 leading-relaxed text-muted">{product.description}</p>
            </div>
          )}

          {/* Attributes */}
          {product.attributes && Object.keys(product.attributes).length > 0 && (
            <div className="mt-6">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground">
                Details
              </h2>
              <dl className="mt-3 space-y-2">
                {Object.entries(product.attributes).map(([key, value]) => (
                  <div key={key} className="flex gap-4 text-sm">
                    <dt className="w-32 flex-shrink-0 font-medium capitalize text-foreground">
                      {key.replace(/_/g, ' ')}
                    </dt>
                    <dd className="text-muted">{value}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}

          {/* Quantity + Add to Cart */}
          <div className="mt-8 border-t border-border pt-6">
            <div className="flex items-center gap-4">
              {/* Quantity selector */}
              <div className="flex items-center rounded-lg border border-border">
                <button
                  type="button"
                  onClick={() => setQuantity((q) => Math.max(1, q - 1))}
                  disabled={quantity <= 1}
                  className="px-3 py-2 text-foreground transition-colors hover:bg-background disabled:opacity-40"
                  aria-label="Decrease quantity"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                    className="h-4 w-4"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14" />
                  </svg>
                </button>
                <span className="w-12 text-center text-sm font-medium text-foreground">
                  {quantity}
                </span>
                <button
                  type="button"
                  onClick={() => setQuantity((q) => q + 1)}
                  className="px-3 py-2 text-foreground transition-colors hover:bg-background"
                  aria-label="Increase quantity"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                    className="h-4 w-4"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M12 4.5v15m7.5-7.5h-15"
                    />
                  </svg>
                </button>
              </div>

              <Button
                variant="primary"
                size="lg"
                className="flex-1"
                onClick={handleAddToCart}
                loading={adding}
                disabled={!product.in_stock}
              >
                {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
