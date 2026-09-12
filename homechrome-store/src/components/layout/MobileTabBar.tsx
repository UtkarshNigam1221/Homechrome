'use client';

import {
  HomeIcon,
  ShoppingBagIcon,
  Squares2X2Icon,
  SwatchIcon,
} from '@heroicons/react/24/outline';
import { Box, Group, Indicator, Stack, Text, UnstyledButton } from '@mantine/core';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { useCartStore } from '@/stores/cart';
import { useUIStore } from '@/stores/uiStore';

const TABS = [
  { href: '/', label: 'Home', icon: HomeIcon },
  { href: '/categories', label: 'Categories', icon: Squares2X2Icon },
  { href: '/products', label: 'Shop', icon: SwatchIcon },
];

/** Phone-only bottom navigation. The cart opens the drawer rather than routing. */
export function MobileTabBar() {
  const pathname = usePathname();
  const cartCount = useCartStore((s) => s.itemCount);
  const openMiniCart = useUIStore((s) => s.openMiniCart);

  return (
    <Box
      component="nav"
      aria-label="Primary"
      pos="fixed"
      bottom={0}
      left={0}
      right={0}
      hiddenFrom="md"
      bg="var(--mantine-color-body)"
      style={{ zIndex: 30, borderTop: '1px solid var(--mantine-color-navy-2)' }}
    >
      <Group gap={0} grow px={4} py={6}>
        {TABS.map(({ href, label, icon: Icon }) => {
          const active = pathname === href;
          return (
            <UnstyledButton
              key={href}
              component={Link}
              href={href}
              aria-current={active ? 'page' : undefined}
            >
              <Stack gap={3} align="center">
                <Icon width={22} height={22} color={`var(--mantine-color-${active ? 'brand-5' : 'navy-6'})`} />
                <Text fz={10} fw={active ? 700 : 500} c={active ? 'brand.5' : 'navy.6'}>
                  {label}
                </Text>
              </Stack>
            </UnstyledButton>
          );
        })}

        <UnstyledButton onClick={openMiniCart} aria-label={`Cart, ${cartCount} items`}>
          <Stack gap={3} align="center">
            <Indicator
              label={cartCount > 0 ? cartCount : undefined}
              disabled={cartCount === 0}
              size={15}
              offset={2}
              color="brand"
            >
              <ShoppingBagIcon width={22} height={22} color="var(--mantine-color-navy-6)" />
            </Indicator>
            <Text fz={10} fw={500} c="navy.6">
              Cart
            </Text>
          </Stack>
        </UnstyledButton>
      </Group>
    </Box>
  );
}
