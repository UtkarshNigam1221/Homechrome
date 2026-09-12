'use client';

import { Bars3Icon, MagnifyingGlassIcon, ShoppingBagIcon, UserIcon } from '@heroicons/react/24/outline';
import {
  ActionIcon,
  Anchor,
  Box,
  Container,
  Group,
  Indicator,
  Kbd,
  Stack,
  Text,
  Tooltip,
  UnstyledButton,
} from '@mantine/core';
import { spotlight } from '@mantine/spotlight';
import dynamic from 'next/dynamic';
import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState } from 'react';

import logo80 from '@/assets/logo-80.webp';

import { displayFont } from '@/app/fonts';
import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';
import { useUIStore } from '@/stores/uiStore';
import { Category } from '@/types';

const MobileNav = dynamic(() => import('./MobileNav'), { ssr: false });

interface HeaderProps {
  categories: Category[];
}

/** "All Products" then the first three categories, then the index page. */
function navLinks(categories: Category[]) {
  return [
    { label: 'All Products', href: '/products' },
    ...categories.slice(0, 3).map((c) => ({ label: c.name, href: `/c/${c.slug}` })),
    { label: 'Categories', href: '/categories' },
  ];
}

export default function Header({ categories }: HeaderProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const cartCount = useCartStore((s) => s.itemCount);
  const openMiniCart = useUIStore((s) => s.openMiniCart);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const pathname = usePathname();

  return (
    <>
      <Box
        component="header"
        pos="sticky"
        top={0}
        mih="var(--app-header-h)"
        style={{
          zIndex: 40,
          borderBottom: '1px solid var(--mantine-color-navy-2)',
          background: 'rgba(252,249,244,0.95)',
          backdropFilter: 'blur(8px)',
        }}
      >
        <Container size="xl">
          <Group justify="space-between" gap="sm" py={10} wrap="nowrap">
            <Group gap="xs" wrap="nowrap">
              <ActionIcon
                variant="subtle"
                color="navy"
                size="lg"
                hiddenFrom="lg"
                onClick={() => setMobileMenuOpen(true)}
                aria-label="Open menu"
              >
                <Bars3Icon width={20} height={20} />
              </ActionIcon>

              <Anchor component={Link} href="/" underline="never">
                <Group gap={10} wrap="nowrap" align="center">
                  <Image
                    src={logo80}
                    alt="Homechrome"
                    height={36}
                    style={{ height: 36, width: 'auto' }}
                    unoptimized
                  />
                  <Stack gap={0} visibleFrom="md">
                    <Text
                      fz={20}
                      fw={600}
                      c="navy.9"
                      lh={1.1}
                      style={{ fontFamily: displayFont.style.fontFamily }}
                    >
                      HOME<Text span c="brand.5" inherit>CHROME</Text>
                    </Text>
                    <Text fz={9} fw={600} c="navy.5" style={{ letterSpacing: '0.12em' }}>
                      HANDWOVEN COMFORT &amp; LIVING
                    </Text>
                  </Stack>
                </Group>
              </Anchor>
            </Group>

            <Group gap={28} visibleFrom="lg" wrap="nowrap">
              {navLinks(categories).map((link) => {
                const active = pathname === link.href;
                return (
                  <Anchor
                    key={link.href}
                    component={Link}
                    href={link.href}
                    size="sm"
                    fw={active ? 600 : 500}
                    c={active ? 'brand.5' : 'navy.7'}
                    underline="never"
                  >
                    {link.label}
                  </Anchor>
                );
              })}
            </Group>

            <Group gap="sm" align="center" wrap="nowrap">
              <Box flex={1} miw={{ base: 0, lg: 200 }} maw={{ base: '100%', lg: 240 }}>
                <SpotlightTrigger />
              </Box>

              <Tooltip label={cartCount > 0 ? `Cart (${cartCount})` : 'Cart'} withArrow>
                <Indicator
                  label={cartCount > 0 ? cartCount : undefined}
                  disabled={cartCount === 0}
                  size={16}
                  offset={2}
                  color="brand"
                >
                  <ActionIcon
                    component={Link}
                    href="/cart"
                    variant="subtle"
                    color="navy"
                    size="lg"
                    aria-label="Open cart"
                    onClick={(e) => {
                      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) return;
                      e.preventDefault();
                      openMiniCart();
                    }}
                  >
                    <ShoppingBagIcon width={22} height={22} />
                  </ActionIcon>
                </Indicator>
              </Tooltip>

              <Tooltip
                label={isAuthenticated ? customer?.first_name || 'Account' : 'Login'}
                withArrow
              >
                <ActionIcon
                  component={Link}
                  href={isAuthenticated ? '/account' : '/login'}
                  variant="filled"
                  color="brand"
                  radius="xl"
                  size="lg"
                  aria-label={isAuthenticated ? 'My account' : 'Login'}
                >
                  <UserIcon width={20} height={20} />
                </ActionIcon>
              </Tooltip>
            </Group>
          </Group>
        </Container>
      </Box>

      <MobileNav
        isOpen={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        categories={categories}
      />
    </>
  );
}

function SpotlightTrigger() {
  return (
    <UnstyledButton
      onClick={() => spotlight.open()}
      aria-label="Search"
      w="100%"
      h={38}
      px="md"
      bg="navy.1"
      style={{
        borderRadius: 'var(--mantine-radius-xl)',
        border: '1px solid var(--mantine-color-navy-2)',
      }}
    >
      <Group gap="xs" wrap="nowrap" align="center" h="100%">
        <MagnifyingGlassIcon width={16} height={16} color="var(--mantine-color-dimmed)" />
        <Text size="sm" c="dimmed" visibleFrom="sm" style={{ flex: 1, textAlign: 'left' }} truncate>
          Search craft...
        </Text>
        <Text size="sm" c="dimmed" hiddenFrom="sm" style={{ flex: 1, textAlign: 'left' }}>
          Search
        </Text>
        <Kbd size="xs" visibleFrom="xl">
          ⌘K
        </Kbd>
      </Group>
    </UnstyledButton>
  );
}
