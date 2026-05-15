'use client';

import Image from 'next/image';

import ShippingLine from '@/components/checkout/ShippingLine';
import { formatPrice } from '@/lib/utils';
import { CartItem, CourierOption } from '@/types';

interface OrderSummaryProps {
  items: CartItem[];
  subtotal: number;
  shippingCourier?: CourierOption | null;
}

export default function OrderSummary({
  items,
  subtotal,
  shippingCourier,
}: OrderSummaryProps) {
  const shippingCharge = shippingCourier ? shippingCourier.rate : undefined;
  const total = subtotal + (shippingCharge ?? 0);
  const totalLabel =
    shippingCharge === undefined
      ? `${formatPrice(subtotal)} + shipping`
      : formatPrice(total);

  return (
    <div className="rounded-lg border border-border bg-white p-5">
      <h3 className="mb-4 text-lg font-semibold text-foreground">Order Summary</h3>

      <div className="mb-4 max-h-64 space-y-3 overflow-y-auto">
        {items.map((item) => (
          <div key={item.product_id} className="flex gap-3">
            <div className="relative h-16 w-16 flex-shrink-0 overflow-hidden rounded-md bg-gray-100">
              {item.product_image ? (
                <Image
                  src={item.product_image}
                  alt={item.product_name}
                  fill
                  className="object-cover"
                  sizes="64px"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">
                  No image
                </div>
              )}
            </div>
            <div className="flex flex-1 justify-between">
              <div>
                <p className="text-sm font-medium text-foreground line-clamp-2">
                  {item.product_name}
                </p>
                <p className="text-xs text-muted-foreground">Qty: {item.quantity}</p>
              </div>
              <p className="text-sm font-medium text-foreground">
                {formatPrice(item.total_price)}
              </p>
            </div>
          </div>
        ))}
      </div>

      <div className="space-y-2 border-t border-border pt-4">
        <div className="flex justify-between text-sm">
          <span className="text-muted-foreground">Subtotal</span>
          <span className="text-foreground">{formatPrice(subtotal)}</span>
        </div>
        <ShippingLine chargePaise={shippingCharge} pendingLabel="Calculated next" />
        <div className="flex justify-between border-t border-border pt-2 text-base font-semibold">
          <span className="text-foreground">Total</span>
          <span className="text-foreground">{totalLabel}</span>
        </div>
      </div>
    </div>
  );
}
