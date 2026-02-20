'use client';

import Image from 'next/image';
import { useState } from 'react';

import { formatPrice } from '@/lib/utils';
import { CartItem as CartItemType } from '@/types';

interface CartItemProps {
  item: CartItemType;
  onUpdateQuantity: (productId: string, quantity: number) => Promise<void>;
  onRemove: (productId: string) => Promise<void>;
}

export default function CartItem({ item, onUpdateQuantity, onRemove }: CartItemProps) {
  const [updating, setUpdating] = useState(false);
  const [removing, setRemoving] = useState(false);

  const handleQuantityChange = async (newQuantity: number) => {
    if (newQuantity < 1) return;
    setUpdating(true);
    try {
      await onUpdateQuantity(item.product_id, newQuantity);
    } finally {
      setUpdating(false);
    }
  };

  const handleRemove = async () => {
    setRemoving(true);
    try {
      await onRemove(item.product_id);
    } finally {
      setRemoving(false);
    }
  };

  return (
    <div className="flex gap-4 rounded-xl bg-white p-4 shadow-sm">
      {/* Product image */}
      <div className="relative h-24 w-24 flex-shrink-0 overflow-hidden rounded-lg bg-gray-100 sm:h-32 sm:w-32">
        {item.product_image ? (
          <Image
            src={item.product_image}
            alt={item.product_name}
            fill
            sizes="128px"
            className="object-cover"
          />
        ) : (
          <div className="flex h-full items-center justify-center bg-primary-light/30">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1}
              stroke="currentColor"
              className="h-8 w-8 text-primary/40"
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

      {/* Product details */}
      <div className="flex flex-1 flex-col">
        <div className="flex justify-between">
          <div>
            <h3 className="text-sm font-medium text-foreground sm:text-base">
              {item.product_name}
            </h3>
            <p className="mt-0.5 text-xs text-muted">SKU: {item.product_sku}</p>
          </div>
          <p className="text-sm font-bold text-foreground sm:text-base">
            {formatPrice(item.total_price)}
          </p>
        </div>

        <div className="mt-auto flex items-center justify-between pt-3">
          {/* Quantity controls */}
          <div className="flex items-center rounded-lg border border-border">
            <button
              type="button"
              onClick={() => handleQuantityChange(item.quantity - 1)}
              disabled={item.quantity <= 1 || updating}
              className="px-2.5 py-1.5 text-foreground transition-colors hover:bg-background disabled:opacity-40"
              aria-label="Decrease quantity"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={2}
                stroke="currentColor"
                className="h-3.5 w-3.5"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14" />
              </svg>
            </button>
            <span className="w-10 text-center text-sm font-medium text-foreground">
              {updating ? '...' : item.quantity}
            </span>
            <button
              type="button"
              onClick={() => handleQuantityChange(item.quantity + 1)}
              disabled={updating}
              className="px-2.5 py-1.5 text-foreground transition-colors hover:bg-background disabled:opacity-40"
              aria-label="Increase quantity"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={2}
                stroke="currentColor"
                className="h-3.5 w-3.5"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 4.5v15m7.5-7.5h-15"
                />
              </svg>
            </button>
          </div>

          {/* Unit price */}
          <p className="hidden text-sm text-muted sm:block">
            {formatPrice(item.unit_price)} each
          </p>

          {/* Remove button */}
          <button
            type="button"
            onClick={handleRemove}
            disabled={removing}
            className="text-sm text-red-500 transition-colors hover:text-red-600 disabled:opacity-50"
          >
            {removing ? 'Removing...' : 'Remove'}
          </button>
        </div>
      </div>
    </div>
  );
}
