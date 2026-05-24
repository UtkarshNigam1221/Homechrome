'use client';

import { ArrowUpIcon, ShoppingBagIcon } from '@heroicons/react/24/outline';
import { ActionIcon, Affix, Box, Button, Indicator, Stack, Transition } from '@mantine/core';
import { useWindowScroll } from '@mantine/hooks';

import { useCartStore } from '@/stores/cart';
import { useUIStore } from '@/stores/uiStore';

export function FloatingActions() {
  const [scroll, scrollTo] = useWindowScroll();
  const cartCount = useCartStore((s) => s.itemCount);
  const openMiniCart = useUIStore((s) => s.openMiniCart);
  const visible = scroll.y > 400;

  return (
    <Affix position={{ bottom: 24, right: 24 }} zIndex={30}>
      <Stack gap="sm" align="flex-end">
        <Box hiddenFrom="md">
          <Transition transition="slide-up" mounted={cartCount > 0}>
            {(styles) => (
              <Indicator label={cartCount} size={18} color="brand" offset={6} style={styles}>
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
            )}
          </Transition>
        </Box>

        <Transition transition="slide-up" mounted={visible}>
          {(styles) => (
            <ActionIcon
              style={styles}
              onClick={() => scrollTo({ y: 0 })}
              variant="filled"
              color="navy"
              radius="xl"
              size="xl"
              aria-label="Back to top"
            >
              <ArrowUpIcon width={20} height={20} />
            </ActionIcon>
          )}
        </Transition>
      </Stack>
    </Affix>
  );
}
