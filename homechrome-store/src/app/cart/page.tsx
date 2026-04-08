'use client';

import { ShoppingBagIcon } from '@heroicons/react/24/outline';

import CartItemComponent from '@/components/cart/CartItem';
import CartSummary from '@/components/cart/CartSummary';
import CartSkeleton from '@/components/skeleton/CartSkeleton';
import { Container } from '@/components/ui/container';
import { EmptyState } from '@/components/ui/empty-state';
import { PageHeader } from '@/components/ui/page-header';
import { useCart } from '@/hooks/useCart';
import { useAuthStore } from '@/stores/auth';

export default function CartPage() {
  const { cart, loading, updateQuantity, removeItem } = useCart();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAuthLoading = useAuthStore((s) => s.isLoading);

  if (isAuthLoading || loading) {
    return <CartSkeleton />;
  }

  const items = cart?.items || [];

  if (items.length === 0) {
    return (
      <Container className="py-10">
        <PageHeader title="Shopping Cart" />
        <EmptyState
          icon={<ShoppingBagIcon strokeWidth={1} className="h-16 w-16 text-muted-foreground/50" />}
          title="Your cart is empty"
          description="Browse our collection and add some beautiful textiles."
          actionLabel="Start Shopping"
          actionHref="/products"
        />
      </Container>
    );
  }

  return (
    <Container className="py-10">
      <PageHeader title="Shopping Cart" />

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
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
    </Container>
  );
}
