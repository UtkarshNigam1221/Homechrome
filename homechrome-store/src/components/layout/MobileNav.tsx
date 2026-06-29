'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import {
  Anchor,
  AspectRatio,
  Box,
  Button,
  Center,
  Divider,
  Drawer,
  Group,
  ScrollArea,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import { AssetImage } from '@/components/ui/asset-image';
import logo80 from '@/assets/logo-80.png';
import { SUPPORT_WHATSAPP } from '@/lib/constants';
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

interface MobileNavProps {
  isOpen: boolean;
  onClose: () => void;
  categories: Category[];
}

export default function MobileNav({ isOpen, onClose, categories }: MobileNavProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const contactHref = `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`;

  return (
    <Drawer
      opened={isOpen}
      onClose={onClose}
      position="left"
      size="xs"
      closeButtonProps={{ 'aria-label': 'Close menu' }}
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
          <Title order={3} size="xs" tt="uppercase" fw={600} c="dimmed" mb="xs" px="xs" style={{ letterSpacing: '0.05em' }}>
            Categories
          </Title>
          <SimpleGrid cols={3} spacing="xs" px="xs">
            {categories.map((category) => (
              <CategoryTile key={category.id} category={category} onClose={onClose} />
            ))}
          </SimpleGrid>

          <Box mt="sm">
            <NavItem href="/products" onClose={onClose}>
              All Products
            </NavItem>
          </Box>
        </Box>
      </ScrollArea>

      <Divider />

      <Box p="sm">
        <Stack gap="xs">
          {isAuthenticated && (
            <>
              <NavItem href="/account" onClose={onClose}>
                Account
              </NavItem>
              <NavItem href="/account/orders" onClose={onClose}>
                Orders
              </NavItem>
            </>
          )}
          {!isAuthenticated && (
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
          <NavItem href={contactHref} onClose={onClose} external>
            Contact us
          </NavItem>
        </Stack>
      </Box>
    </Drawer>
  );
}

interface NavItemProps {
  href: string;
  onClose: () => void;
  children: React.ReactNode;
  external?: boolean;
}

function NavItem({ href, onClose, children, external }: NavItemProps) {
  if (external) {
    return (
      <Anchor
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        onClick={onClose}
        underline="never"
        c="navy.7"
        size="md"
        fw={500}
        px="xs"
        py={10}
        lineClamp={2}
        style={{ borderRadius: 'var(--mantine-radius-md)' }}
      >
        {children}
      </Anchor>
    );
  }

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
      lineClamp={2}
      style={{ borderRadius: 'var(--mantine-radius-md)' }}
    >
      {children}
    </Anchor>
  );
}

interface CategoryTileProps {
  category: Category;
  onClose: () => void;
}

function CategoryTile({ category, onClose }: CategoryTileProps) {
  return (
    <Anchor
      component={Link}
      href={`/c/${category.slug}`}
      onClick={onClose}
      underline="never"
      c="navy.7"
    >
      <AspectRatio
        ratio={1}
        bg="gray.1"
        style={{ borderRadius: 'var(--mantine-radius-md)', overflow: 'hidden' }}
      >
        {category.image_url ? (
          <AssetImage
            src={category.image_url}
            alt={category.name}
            sizes="120px"
            width={120}
            height={120}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          <Center bg="brand.1" h="100%">
            <PhotoIcon width={24} height={24} color="var(--mantine-color-brand-5)" opacity={0.4} />
          </Center>
        )}
      </AspectRatio>
      <Text size="xs" fw={500} mt={4} lineClamp={2} style={{ lineHeight: 1.2 }}>
        {category.name}
      </Text>
    </Anchor>
  );
}
