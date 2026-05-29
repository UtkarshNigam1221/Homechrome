'use client';

import { ShoppingBagIcon } from '@heroicons/react/24/outline';
import { Box, Container, SimpleGrid, Stack } from '@mantine/core';

import CartItemComponent from '@/components/cart/CartItem';
import CartSummary from '@/components/cart/CartSummary';
import CartSkeleton from '@/components/skeleton/CartSkeleton';
import InlineLoaderOverlay from '@/components/ui/InlineLoaderOverlay';
import { EmptyState } from '@/components/ui/empty-state';
import { PageHeader } from '@/components/ui/page-header';
import { useCart } from '@/hooks/useCart';
import { useAuthStore } from '@/stores/auth';

export default function CartPage() {
  const { cart, loading, updateQuantity, removeItem, updatingItemId, removingItemId } = useCart();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAuthLoading = useAuthStore((s) => s.isLoading);

  if (isAuthLoading || loading) {
    return <CartSkeleton />;
  }

  const items = cart?.items || [];

  if (items.length === 0) {
    return (
      <Container size="xl" py="xl">
        <PageHeader title="Shopping Cart" />
        <EmptyState
          icon={<ShoppingBagIcon strokeWidth={1} width={64} height={64} color="var(--mantine-color-dimmed)" />}
          title="Your cart is empty"
          description="Browse our collection and add some beautiful textiles."
          actionLabel="Start Shopping"
          actionHref="/products"
        />
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <PageHeader title="Shopping Cart" />

      <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="xl">
        <Stack gap="md" style={{ gridColumn: 'span 2 / span 2' }}>
          {items.map((item) => (
            <Box key={item.product_id} pos="relative">
              <CartItemComponent
                item={item}
                onUpdateQuantity={async (productId, qty) => {
                  await updateQuantity(productId, qty);
                }}
                onRemove={async (productId) => {
                  await removeItem(productId);
                }}
              />
              <InlineLoaderOverlay
                visible={updatingItemId === item.product_id || removingItemId === item.product_id}
                label="Updating item"
              />
            </Box>
          ))}
        </Stack>

        <Box>
          <Box pos="sticky" top={128}>
            <CartSummary
              subtotal={cart?.cart.subtotal || 0}
              itemCount={cart?.cart.item_count || 0}
              isAuthenticated={isAuthenticated}
            />
          </Box>
        </Box>
      </SimpleGrid>
    </Container>
  );
}
