'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useState } from 'react';

import Button from '@/components/common/Button';
import { useCart } from '@/hooks/useCart';
import { formatPrice } from '@/lib/utils';
import { useCartStore } from '@/stores/cart';
import { Product } from '@/types';

interface ProductCardProps {
  product: Product;
}

export default function ProductCard({ product }: ProductCardProps) {
  const [loading, setLoading] = useState(false);
  const { addItem, updateQuantity, removeItem } = useCart();
  const cartQty = useCartStore((s) => s.getQuantity(product.id));

  const primaryImage = product.images?.find((img) => img.is_primary) || product.images?.[0];
  const hasDiscount = product.mrp > product.selling_price;
  const discountPercent = hasDiscount
    ? Math.round(((product.mrp - product.selling_price) / product.mrp) * 100)
    : 0;

  const handleAddToCart = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setLoading(true);
    try {
      await addItem(product.id, 1);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  const handleIncrement = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setLoading(true);
    try {
      await updateQuantity(product.id, cartQty + 1);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  const handleDecrement = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setLoading(true);
    try {
      if (cartQty <= 1) {
        await removeItem(product.id);
      } else {
        await updateQuantity(product.id, cartQty - 1);
      }
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="group flex flex-col overflow-hidden rounded-xl bg-white shadow-sm transition-shadow hover:shadow-md">
      <Link href={`/p/${product.slug}`} className="relative aspect-square overflow-hidden bg-gray-100">
        {primaryImage ? (
          <Image
            src={primaryImage.url}
            alt={primaryImage.alt_text || product.name}
            fill
            sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
            className="object-cover transition-transform duration-300 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full items-center justify-center bg-primary-light/30">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1}
              stroke="currentColor"
              className="h-12 w-12 text-primary/40"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z"
              />
            </svg>
          </div>
        )}
        {hasDiscount && (
          <span className="absolute left-2 top-2 rounded-md bg-red-500 px-2 py-0.5 text-xs font-semibold text-white">
            -{discountPercent}%
          </span>
        )}
        {!product.in_stock && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/40">
            <span className="rounded-md bg-white px-3 py-1 text-sm font-medium text-foreground">
              Out of Stock
            </span>
          </div>
        )}
      </Link>

      <div className="flex flex-1 flex-col p-4">
        <Link href={`/p/${product.slug}`}>
          <h3 className="line-clamp-2 text-sm font-medium text-foreground group-hover:text-primary">
            {product.name}
          </h3>
        </Link>

        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-lg font-bold text-foreground">
            {formatPrice(product.selling_price)}
          </span>
          {hasDiscount && (
            <span className="text-sm text-muted line-through">
              {formatPrice(product.mrp)}
            </span>
          )}
        </div>

        <div className="mt-auto pt-3">
          {cartQty > 0 ? (
            <div className="flex items-center justify-between rounded-lg border border-primary bg-primary/5">
              <button
                type="button"
                onClick={handleDecrement}
                disabled={loading}
                className="px-3 py-2 text-primary transition-colors hover:bg-primary/10 disabled:opacity-40"
                aria-label="Decrease quantity"
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" className="h-4 w-4">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14" />
                </svg>
              </button>
              <span className="text-sm font-semibold text-primary">{cartQty}</span>
              <button
                type="button"
                onClick={handleIncrement}
                disabled={loading}
                className="px-3 py-2 text-primary transition-colors hover:bg-primary/10 disabled:opacity-40"
                aria-label="Increase quantity"
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" className="h-4 w-4">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                </svg>
              </button>
            </div>
          ) : (
            <Button
              variant="primary"
              size="sm"
              className="w-full"
              onClick={handleAddToCart}
              loading={loading}
              disabled={!product.in_stock}
            >
              {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
