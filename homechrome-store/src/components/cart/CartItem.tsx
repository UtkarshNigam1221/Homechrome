'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import Image from 'next/image';
import { useState } from 'react';

import Button from '@/components/ui/button';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
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
    <article className="flex gap-4 rounded-xl bg-card p-4 shadow-sm">
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
            <PhotoIcon className="h-8 w-8 text-primary/40" />
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
            <p className="mt-0.5 text-xs text-muted-foreground">SKU: {item.product_sku}</p>
          </div>
          <p className="text-sm font-bold text-foreground sm:text-base">
            {formatPrice(item.total_price)}
          </p>
        </div>

        <div className="mt-auto flex items-center justify-between pt-3">
          <QuantityStepper
            value={item.quantity}
            onIncrement={() => handleQuantityChange(item.quantity + 1)}
            onDecrement={() => handleQuantityChange(item.quantity - 1)}
            disableDecrement={item.quantity <= 1 || updating}
            disabled={updating}
            loading={updating}
            size="sm"
          />

          <p className="hidden text-sm text-muted-foreground sm:block">
            {formatPrice(item.unit_price)} each
          </p>

          <Button
            variant="destructive"
            size="sm"
            onClick={handleRemove}
            loading={removing}
          >
            Remove
          </Button>
        </div>
      </div>
    </article>
  );
}
