'use client';

import Link from 'next/link';

import CartItemComponent from '@/components/cart/CartItem';
import CartSummary from '@/components/cart/CartSummary';
import { useCart } from '@/hooks/useCart';
import { useAuthStore } from '@/stores/auth';

export default function CartPage() {
  const { cart, loading, updateQuantity, removeItem } = useCart();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAuthLoading = useAuthStore((s) => s.isLoading);

  // Loading state
  if (isAuthLoading || loading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>
        <div className="mt-8 flex items-center justify-center py-16">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      </div>
    );
  }

  // Empty cart
  const items = cart?.items || [];
  if (items.length === 0) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>
        <div className="mt-8 flex flex-col items-center justify-center py-16 text-center">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1}
            stroke="currentColor"
            className="h-16 w-16 text-muted/50"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
            />
          </svg>
          <h2 className="mt-4 text-lg font-medium text-foreground">
            Your cart is empty
          </h2>
          <p className="mt-2 text-sm text-muted">
            Browse our collection and add some beautiful textiles.
          </p>
          <Link
            href="/products"
            className="mt-6 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            Start Shopping
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Cart items */}
        <div className="space-y-4 lg:col-span-2">
          {items.map((item) => (
            <CartItemComponent
              key={item.product_id}
              item={item}
              onUpdateQuantity={async (productId, qty) => {
                await updateQuantity(productId, qty);
              }}
              onRemove={async (productId) => {
                await removeItem(productId);
              }}
            />
          ))}
        </div>

        {/* Summary */}
        <div>
          <div className="sticky top-32">
            <CartSummary
              subtotal={cart?.cart.subtotal || 0}
              itemCount={cart?.cart.item_count || 0}
              isAuthenticated={isAuthenticated}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
