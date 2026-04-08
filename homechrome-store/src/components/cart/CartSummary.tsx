'use client';

import Link from 'next/link';

import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { formatPrice } from '@/lib/utils';

interface CartSummaryProps {
  subtotal: number;
  itemCount: number;
  isAuthenticated: boolean;
}

const FREE_SHIPPING_THRESHOLD = 99900; // 999 INR in paise

export default function CartSummary({ subtotal, itemCount, isAuthenticated }: CartSummaryProps) {
  const shippingEstimate = subtotal >= FREE_SHIPPING_THRESHOLD ? 0 : 7900; // 79 INR flat rate
  const total = subtotal + shippingEstimate;
  const amountToFreeShipping = FREE_SHIPPING_THRESHOLD - subtotal;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Order Summary</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="space-y-3">
          <div className="flex justify-between text-sm">
            <dt className="text-muted-foreground">
              Subtotal ({itemCount} {itemCount === 1 ? 'item' : 'items'})
            </dt>
            <dd className="font-medium text-foreground">{formatPrice(subtotal)}</dd>
          </div>

          <div className="flex justify-between text-sm">
            <dt className="text-muted-foreground">Shipping estimate</dt>
            <dd className="font-medium text-foreground">
              {shippingEstimate === 0 ? (
                <span className="text-green-600">Free</span>
              ) : (
                formatPrice(shippingEstimate)
              )}
            </dd>
          </div>

          <Separator />

          <div className="flex justify-between">
            <dt className="text-base font-semibold text-foreground">Total</dt>
            <dd className="text-base font-bold text-foreground">{formatPrice(total)}</dd>
          </div>
        </dl>

        {subtotal > 0 && amountToFreeShipping > 0 && (
          <p className="mt-4 text-xs text-muted-foreground">
            Add {formatPrice(amountToFreeShipping)} more for free shipping!
          </p>
        )}

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
