'use client';

import Link from 'next/link';

import ShippingLine from '@/components/checkout/ShippingLine';
import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { formatPrice } from '@/lib/utils';

interface CartSummaryProps {
  subtotal: number;
  itemCount: number;
  isAuthenticated: boolean;
  shippingCharge?: number; // backend-provided, in paise
}

export default function CartSummary({
  subtotal,
  itemCount,
  isAuthenticated,
  shippingCharge,
}: CartSummaryProps) {
  const total = subtotal + (shippingCharge ?? 0);
  const totalLabel =
    shippingCharge === undefined
      ? `${formatPrice(subtotal)} + shipping`
      : formatPrice(total);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Order Summary</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">
              Subtotal ({itemCount} {itemCount === 1 ? 'item' : 'items'})
            </span>
            <span className="font-medium text-foreground">{formatPrice(subtotal)}</span>
          </div>

          <ShippingLine chargePaise={shippingCharge} pendingLabel="Calculated at checkout" />

          <Separator />

          <div className="flex justify-between">
            <span className="text-base font-semibold text-foreground">Total</span>
            <span className="text-base font-bold text-foreground">{totalLabel}</span>
          </div>
        </div>

        <div className="mt-6">
          {isAuthenticated ? (
            <Link href="/checkout">
              <Button variant="primary" size="lg" className="w-full" disabled={itemCount === 0}>
                Proceed to Checkout
              </Button>
            </Link>
          ) : (
            <Link href="/login?redirect=/cart">
              <Button variant="primary" size="lg" className="w-full" disabled={itemCount === 0}>
                Sign in to Checkout
              </Button>
            </Link>
          )}
        </div>

        <div className="mt-4 text-center">
          <Link
            href="/products"
            className="text-sm text-primary transition-colors hover:text-primary-dark"
          >
            Continue Shopping
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}
