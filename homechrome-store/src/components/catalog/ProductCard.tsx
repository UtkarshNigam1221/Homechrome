'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import Image from 'next/image';
import Link from 'next/link';
import { useState } from 'react';

import { toast } from 'sonner';

import Button from '@/components/ui/button';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useCart } from '@/hooks/useCart';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
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
  const discountPercent = calculateDiscountPercent(product.mrp, product.selling_price);

  const handleAddToCart = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setLoading(true);
    try {
      await addItem(product.id, 1);
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  const handleIncrement = async () => {
    setLoading(true);
    try {
      await updateQuantity(product.id, cartQty + 1);
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  const handleDecrement = async () => {
    setLoading(true);
    try {
      if (cartQty <= 1) {
        await removeItem(product.id);
      } else {
        await updateQuantity(product.id, cartQty - 1);
      }
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  return (
    <article className="group flex flex-col overflow-hidden rounded-xl bg-card shadow-sm transition-shadow hover:shadow-md">
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
            <PhotoIcon className="h-12 w-12 text-primary/40" />
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
            <span className="text-sm text-muted-foreground line-through">
              {formatPrice(product.mrp)}
            </span>
          )}
        </div>

        <div className="mt-auto pt-3">
          {cartQty > 0 ? (
            <QuantityStepper
              value={cartQty}
              onIncrement={handleIncrement}
              onDecrement={handleDecrement}
              disabled={loading}
              variant="primary"
              className="w-full justify-between"
            />
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
    </article>
  );
}
