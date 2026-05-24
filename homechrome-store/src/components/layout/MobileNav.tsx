'use client';

import {
  Anchor,
  Box,
  Button,
  Divider,
  Drawer,
  Group,
  ScrollArea,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import logo80 from '@/assets/logo-80.png';
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

interface MobileNavProps {
  isOpen: boolean;
  onClose: () => void;
  categories: Category[];
}

export default function MobileNav({ isOpen, onClose, categories }: MobileNavProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const logout = useAuthStore((s) => s.logout);

  return (
    <Drawer
      opened={isOpen}
      onClose={onClose}
      position="left"
      size="xs"
      withCloseButton={false}
      padding={0}
      title={
        <Anchor component={Link} href="/" onClick={onClose} underline="never">
          <Group gap="xs" align="center">
            <Image src={logo80} alt="Homechrome" style={{ height: 28, width: 'auto' }} unoptimized />
            <Text fw={700} size="lg" c="navy.7" style={{ letterSpacing: '-0.01em' }}>
              HOME<Text span c="brand">CHROME</Text>
            </Text>
          </Group>
        </Anchor>
      }
      styles={{
        header: {
          borderBottom: '1px solid var(--mantine-color-default-border)',
          padding: 'var(--mantine-spacing-md)',
        },
        title: { flex: 1 },
        content: { display: 'flex', flexDirection: 'column' },
        body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0 },
      }}
    >
      <ScrollArea style={{ flex: 1 }}>
        <Box p="md">
          <NavItem href="/products" onClose={onClose}>
            All Products
          </NavItem>
          <NavItem href="/cart" onClose={onClose}>
            My Cart
          </NavItem>

          <Title order={3} size="xs" tt="uppercase" fw={600} c="dimmed" mt="lg" mb="xs" px="xs" style={{ letterSpacing: '0.05em' }}>
            Categories
          </Title>
          <Stack gap={4}>
            {categories.map((category) => (
              <NavItem key={category.id} href={`/c/${category.slug}`} onClose={onClose}>
                {category.name}
              </NavItem>
            ))}
          </Stack>
        </Box>
      </ScrollArea>

      <Divider />

      <Box p="md">
        {isAuthenticated ? (
          <Stack gap="sm">
            <Text size="sm" c="dimmed">
              Hello, {customer?.first_name || 'there'}
            </Text>
            <NavItem href="/account" onClose={onClose}>
              My Account
            </NavItem>
            <NavItem href="/account/orders" onClose={onClose}>
              My Orders
            </NavItem>
            <Button
              variant="light"
              color="red"
              fullWidth
              justify="start"
              onClick={() => {
                logout();
                onClose();
              }}
            >
              Sign Out
            </Button>
          </Stack>
        ) : (
          <Button
            component={Link}
            href="/login"
            onClick={onClose}
            fullWidth
            color="brand"
          >
            Sign In
          </Button>
        )}
      </Box>
    </Drawer>
  );
}

interface NavItemProps {
  href: string;
  onClose: () => void;
  children: React.ReactNode;
}

function NavItem({ href, onClose, children }: NavItemProps) {
  return (
    <Anchor
      component={Link}
      href={href}
      onClick={onClose}
      underline="never"
      c="navy.7"
      size="md"
      fw={500}
      px="xs"
      py={10}
      display="block"
      style={{ borderRadius: 'var(--mantine-radius-md)' }}
    >
      {children}
    </Anchor>
  );
}
