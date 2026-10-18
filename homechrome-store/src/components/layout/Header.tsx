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
  Text,
  Tooltip,
  UnstyledButton,
} from '@mantine/core';
import { spotlight } from '@mantine/spotlight';
import dynamic from 'next/dynamic';
import Image from 'next/image';
import Link from 'next/link';
import { useState } from 'react';

import logo80 from '@/assets/logo-80.webp';

import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';
import { useUIStore } from '@/stores/uiStore';
import { Category } from '@/types';

const MobileNav = dynamic(() => import('./MobileNav'), { ssr: false });

interface HeaderProps {
  categories: Category[];
}

export default function Header({ categories }: HeaderProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const cartCount = useCartStore((s) => s.itemCount);
  const openMiniCart = useUIStore((s) => s.openMiniCart);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <>
      <Box
        component="header"
        pos="sticky"
        top={0}
        mih="var(--app-header-h)"
        style={{
          zIndex: 40,
          borderBottom: '1px solid var(--mantine-color-default-border)',
          background: 'rgba(255,255,255,0.95)',
          backdropFilter: 'blur(8px)',
        }}
      >
        <Container size="xl">
          <Group justify="space-between" gap="sm" py="xs" wrap="nowrap">
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
                <Group gap="xs" wrap="nowrap" align="center">
                  <Image
                    src={logo80}
                    alt="Homechrome"
                    height={36}
                    style={{ height: 36, width: 'auto' }}
                    unoptimized
                  />
                  <Text fw={700} c="navy.7" visibleFrom="md" style={{ letterSpacing: '-0.01em' }}>
                    HOME<Text span c="brand">CHROME</Text>
                  </Text>
                </Group>
              </Anchor>
            </Group>

            <Box flex={1} maw={{ base: '100%', lg: 460 }} mx={{ base: 'xs', sm: 'md' }}>
              <SpotlightTrigger />
            </Box>

            <Group gap="sm" align="center" wrap="nowrap">
              <Group gap="lg" visibleFrom="lg">
                <Anchor component={Link} href="/products" c="navy.7" fw={500} size="sm" underline="never">
                  Shop
                </Anchor>
                <Anchor component={Link} href="/categories" c="navy.7" fw={500} size="sm" underline="never">
                  Categories
                </Anchor>
              </Group>

              {isAuthenticated ? (
                <Tooltip label={customer?.first_name || 'Account'} withArrow>
                  <Anchor
                    component={Link}
                    href="/account"
                    c="navy.7"
                    underline="never"
                    aria-label="My account"
                  >
                    <UserIcon width={22} height={22} />
                  </Anchor>
                </Tooltip>
              ) : (
                <>
                  <Anchor
                    component={Link}
                    href="/login"
                    c="navy.7"
                    fw={500}
                    size="sm"
                    underline="never"
                    visibleFrom="sm"
                  >
                    Login
                  </Anchor>
                  <Tooltip label="Login" withArrow>
                    <Anchor
                      component={Link}
                      href="/login"
                      c="navy.7"
                      underline="never"
                      hiddenFrom="sm"
                      aria-label="Login"
                    >
                      <UserIcon width={22} height={22} />
                    </Anchor>
                  </Tooltip>
                </>
              )}

              <Box visibleFrom="md" style={{ display: 'flex', alignItems: 'center' }}>
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
              </Box>
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
      h={42}
      px="md"
      bg="gray.0"
      style={{
        borderRadius: 'var(--mantine-radius-xl)',
        border: '1px solid var(--mantine-color-default-border)',
      }}
    >
      <Group gap="sm" wrap="nowrap" align="center" h="100%">
        <MagnifyingGlassIcon width={18} height={18} color="var(--mantine-color-dimmed)" />
        <Text size="sm" c="dimmed" visibleFrom="sm" style={{ flex: 1, textAlign: 'left' }}>
          Search handloom textiles...
        </Text>
        <Text size="sm" c="dimmed" hiddenFrom="sm" style={{ flex: 1, textAlign: 'left' }}>
          Search
        </Text>
        <Kbd size="xs" visibleFrom="md">
          ⌘K
        </Kbd>
      </Group>
    </UnstyledButton>
  );
}
