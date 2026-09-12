'use client';

import {
  ChatBubbleLeftRightIcon,
  ChevronRightIcon,
  RectangleGroupIcon,
  Squares2X2Icon,
  TruckIcon,
  UserIcon,
} from '@heroicons/react/24/outline';
import {
  Anchor,
  Box,
  Button,
  Divider,
  Drawer,
  Group,
  NavLink,
  ScrollArea,
  Stack,
  Text,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import logo80 from '@/assets/logo-80.webp';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import { SUPPORT_WHATSAPP } from '@/lib/constants';
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

import { displayFont } from '@/app/fonts';

const LEAF = '#42634C';

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
      size="80%"
      closeButtonProps={{ 'aria-label': 'Close menu' }}
      padding={0}
      title={
        <Anchor component={Link} href="/" onClick={onClose} underline="never">
          <Group gap={8} align="center" wrap="nowrap">
            <Image src={logo80} alt="Homechrome" style={{ height: 30, width: 'auto' }} unoptimized />
            <Stack gap={0}>
              <Text
                fz={18}
                fw={600}
                c="navy.9"
                lh={1.1}
                style={{ fontFamily: displayFont.style.fontFamily }}
              >
                Homechrome
              </Text>
              <Text fz={9} fw={600} c="navy.5" style={{ letterSpacing: '0.12em' }}>
                ARTISANAL HANDLOOM
              </Text>
            </Stack>
          </Group>
        </Anchor>
      }
      styles={{
        header: {
          borderBottom: '1px solid var(--mantine-color-navy-2)',
          padding: 'var(--mantine-spacing-md)',
        },
        title: { flex: 1 },
        content: { display: 'flex', flexDirection: 'column', maxWidth: 320 },
        body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0 },
      }}
    >
      <ScrollArea style={{ flex: 1 }}>
        <Box px="sm" py="md">
          <Eyebrow>Collections &amp; Shop</Eyebrow>
          <DrawerRow
            href="/products"
            label="All Products"
            icon={<Squares2X2Icon width={20} height={20} aria-hidden="true" />}
            onClose={onClose}
          />
          {categories.map((category) => (
            <DrawerRow
              key={category.id}
              href={`/c/${category.slug}`}
              label={category.name}
              icon={<RectangleGroupIcon width={20} height={20} aria-hidden="true" />}
              onClose={onClose}
            />
          ))}
          <DrawerRow
            href="/categories"
            label="Categories"
            icon={<Squares2X2Icon width={20} height={20} aria-hidden="true" />}
            onClose={onClose}
          />
        </Box>

        <Divider mx="sm" />

        <Box px="sm" py="md">
          <Eyebrow>Your Orders</Eyebrow>
          <DrawerRow
            href="/track"
            label="Track Order"
            icon={<TruckIcon width={20} height={20} aria-hidden="true" />}
            onClose={onClose}
          />
          {isAuthenticated && (
            <DrawerRow
              href="/account"
              label="My Account"
              icon={<UserIcon width={20} height={20} aria-hidden="true" />}
              onClose={onClose}
            />
          )}
          <DrawerRow
            href="/contact"
            label="Contact Us"
            icon={<ChatBubbleLeftRightIcon width={20} height={20} aria-hidden="true" />}
            onClose={onClose}
          />
          {/* Sits after the links, not between them. */}
          {!isAuthenticated && (
            <Button component={Link} href="/login" onClick={onClose} fullWidth color="brand" mt="sm">
              Sign In
            </Button>
          )}
        </Box>
      </ScrollArea>

      <Box p="md" bg="navy.1" style={{ borderTop: '1px solid var(--mantine-color-navy-2)' }}>
        <Anchor
          href={`https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`}
          target="_blank"
          rel="noopener noreferrer"
          underline="never"
          onClick={onClose}
        >
          <Group
            gap={8}
            justify="center"
            align="center"
            c="white"
            py={11}
            style={{ background: LEAF, borderRadius: 'var(--mantine-radius-lg)' }}
          >
            <WhatsAppIcon size={18} />
            <Text fz="sm" fw={600} c="white">
              WhatsApp Concierge
            </Text>
          </Group>
        </Anchor>
        <Text ta="center" fz="xs" c="navy.5" mt={8}>
          Free shipping on every order
        </Text>
      </Box>
    </Drawer>
  );
}

function Eyebrow({ children }: { children: React.ReactNode }) {
  return (
    <Text fz={11} fw={700} c="navy.5" mb={6} px={10} style={{ letterSpacing: '0.12em' }}>
      {children?.toString().toUpperCase()}
    </Text>
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
      rightSection={
        <ChevronRightIcon width={16} height={16} aria-hidden="true" style={{ opacity: 0.4 }} />
      }
      color="brand"
      c="navy.8"
      style={{ borderRadius: 'var(--mantine-radius-md)' }}
    />
  );
}
