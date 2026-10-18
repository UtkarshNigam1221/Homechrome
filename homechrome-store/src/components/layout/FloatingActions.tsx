'use client';

import { ShoppingBagIcon } from '@heroicons/react/24/outline';
import { Affix, Box, Button, Indicator } from '@mantine/core';

import { useCartStore } from '@/stores/cart';
import { useUIStore } from '@/stores/uiStore';

export function FloatingActions() {
  const cartCount = useCartStore((s) => s.itemCount);
  const openMiniCart = useUIStore((s) => s.openMiniCart);

  // Mobile-only: header cart icon is hidden below md, so this is the cart entry point.
  return (
    <Affix position={{ bottom: 24, right: 24 }} zIndex={30}>
      <Box hiddenFrom="md">
        <Indicator label={cartCount} size={18} color="brand" offset={6} disabled={cartCount === 0}>
          <Button
            onClick={openMiniCart}
            color="brand"
            radius="xl"
            size="md"
            leftSection={<ShoppingBagIcon width={18} height={18} />}
          >
            Cart
          </Button>
        </Indicator>
      </Box>
    </Affix>
  );
}
