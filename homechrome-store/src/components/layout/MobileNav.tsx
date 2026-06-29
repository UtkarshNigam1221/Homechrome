'use client';

import {
  ChatBubbleLeftRightIcon,
  ChevronRightIcon,
  ShoppingBagIcon,
  Squares2X2Icon,
  UserIcon,
} from '@heroicons/react/24/outline';
import {
  Anchor,
  AspectRatio,
  Box,
  Button,
  Center,
  Divider,
  Drawer,
  Group,
  NavLink,
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
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

interface MobileNavProps {
  isOpen: boolean;
  onClose: () => void;
  categories: Category[];
}

export default function MobileNav({ isOpen, onClose, categories }: MobileNavProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

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
          <Eyebrow>Categories</Eyebrow>
          <SimpleGrid cols={3} spacing="xs" px="xs">
            {categories.map((category) => (
              <CategoryTile key={category.id} category={category} onClose={onClose} />
            ))}
          </SimpleGrid>

          <Box mt="xs">
            <DrawerRow
              href="/products"
              label="All products"
              icon={<Squares2X2Icon width={20} height={20} />}
              onClose={onClose}
            />
          </Box>
        </Box>
      </ScrollArea>

      <Divider />

      <Box p="sm">
        {isAuthenticated ? (
          <>
            <Eyebrow>Account</Eyebrow>
            <DrawerRow
              href="/account"
              label="Account"
              icon={<UserIcon width={20} height={20} />}
              onClose={onClose}
            />
            <DrawerRow
              href="/account/orders"
              label="Orders"
              icon={<ShoppingBagIcon width={20} height={20} />}
              onClose={onClose}
            />
            <DrawerRow
              href="/contact"
              label="Contact us"
              icon={<ChatBubbleLeftRightIcon width={20} height={20} />}
              onClose={onClose}
            />
          </>
        ) : (
          <Stack gap="xs">
            <Button component={Link} href="/login" onClick={onClose} fullWidth color="brand">
              Sign In
            </Button>
            <DrawerRow
              href="/contact"
              label="Contact us"
              icon={<ChatBubbleLeftRightIcon width={20} height={20} />}
              onClose={onClose}
            />
          </Stack>
        )}
      </Box>
    </Drawer>
  );
}

function Eyebrow({ children }: { children: React.ReactNode }) {
  return (
    <Title
      order={3}
      size="xs"
      tt="uppercase"
      fw={600}
      c="dimmed"
      mb="xs"
      px="xs"
      style={{ letterSpacing: '0.05em' }}
    >
      {children}
    </Title>
  );
}

interface DrawerRowProps {
  href: string;
  label: string;
  icon: React.ReactNode;
  onClose: () => void;
}

function DrawerRow({ href, label, icon, onClose }: DrawerRowProps) {
  return (
    <NavLink
      component={Link}
      href={href}
      onClick={onClose}
      label={label}
      leftSection={icon}
      rightSection={<ChevronRightIcon width={16} height={16} style={{ opacity: 0.4 }} />}
      color="brand"
      c="navy.7"
      style={{ borderRadius: 'var(--mantine-radius-md)' }}
    />
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
            <Text fw={600} fz={22} c="brand.6">
              {category.name.charAt(0).toUpperCase()}
            </Text>
          </Center>
        )}
      </AspectRatio>
      <Text size="xs" fw={500} mt={4} lineClamp={2} style={{ lineHeight: 1.2 }}>
        {category.name}
      </Text>
    </Anchor>
  );
}
