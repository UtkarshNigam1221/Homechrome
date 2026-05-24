'use client';

import { Bars3Icon, ShoppingBagIcon, UserIcon } from '@heroicons/react/24/outline';
import {
  ActionIcon,
  Anchor,
  Box,
  Container,
  Divider,
  Group,
  Indicator,
  Text,
  Tooltip,
} from '@mantine/core';
import dynamic from 'next/dynamic';
import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

import logo80 from '@/assets/logo-80.png';

import { SearchInput } from '@/components/ui/search-input';
import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';
import { Category } from '@/types';

const MobileNav = dynamic(() => import('./MobileNav'), { ssr: false });

interface HeaderProps {
  categories: Category[];
}

export default function Header({ categories }: HeaderProps) {
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const cartCount = useCartStore((s) => s.itemCount);
  const [searchQuery, setSearchQuery] = useState('');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (searchQuery.trim()) {
        router.push(`/products?search=${encodeURIComponent(searchQuery.trim())}`);
        setSearchQuery('');
      }
    },
    [searchQuery, router],
  );

  return (
    <>
      <Box
        component="header"
        pos="sticky"
        top={0}
        style={{
          zIndex: 40,
          borderBottom: '1px solid var(--mantine-color-default-border)',
          background: 'rgba(255,255,255,0.95)',
          backdropFilter: 'blur(8px)',
        }}
      >
        <Container size="xl">
          <Group justify="space-between" gap="md" py="xs">
            <Group gap="xs">
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
                    height={40}
                    style={{ height: 40, width: 'auto' }}
                    priority
                    unoptimized
                  />
                  <Text fw={700} c="navy.7" visibleFrom="sm" style={{ letterSpacing: '-0.01em' }}>
                    HOME<Text span c="brand">CHROME</Text>
                  </Text>
                </Group>
              </Anchor>
            </Group>

            <Box flex={1} maw={400} visibleFrom="lg">
              <SearchInput
                value={searchQuery}
                onChange={setSearchQuery}
                onSubmit={handleSearch}
                placeholder="Search handloom textiles..."
              />
            </Box>

            <Group gap="md" align="center">
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
                    visibleFrom="sm"
                    aria-label="My account"
                  >
                    <UserIcon width={22} height={22} />
                  </Anchor>
                </Tooltip>
              ) : (
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
              )}

              <Tooltip label={cartCount > 0 ? `Cart (${cartCount})` : 'Cart'} withArrow>
                <Indicator
                  label={cartCount > 0 ? cartCount : undefined}
                  disabled={cartCount === 0}
                  size={16}
                  offset={2}
                  color="brand"
                >
                  <Anchor component={Link} href="/cart" c="navy.7" underline="never" aria-label="Cart">
                    <ShoppingBagIcon width={22} height={22} />
                  </Anchor>
                </Indicator>
              </Tooltip>
            </Group>
          </Group>
        </Container>

        <Divider hiddenFrom="lg" />
        <Box hiddenFrom="lg" px="md" pb={6} pt={4}>
          <SearchInput
            value={searchQuery}
            onChange={setSearchQuery}
            onSubmit={handleSearch}
            placeholder="Search handloom textiles..."
          />
        </Box>
      </Box>

      <MobileNav
        isOpen={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        categories={categories}
      />
    </>
  );
}
