'use client';

import { PhoneIcon } from '@heroicons/react/24/outline';
import {
  Anchor,
  Box,
  Container,
  Divider,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import logo80 from '@/assets/logo-80.png';
import { INSTAGRAM_URL, SUPPORT_PHONE, SUPPORT_PHONE_DISPLAY, SUPPORT_WHATSAPP } from '@/lib/constants';

export default function Footer() {
  return (
    <Box component="footer" bg="white" style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}>
      <Container size="xl" py="xl">
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="xl">
          <Stack gap="sm">
            <Anchor component={Link} href="/" underline="never" display="inline-flex">
              <Box style={{ display: 'inline-flex', alignItems: 'center', gap: '0.625rem' }}>
                <Image src={logo80} alt="Homechrome" style={{ height: 40, width: 'auto' }} unoptimized />
                <Text fw={700} size="xl" c="navy.7" style={{ letterSpacing: '-0.01em' }}>
                  HOME<Text span c="brand">CHROME</Text>
                </Text>
              </Box>
            </Anchor>
            <Text size="sm" c="dimmed">
              Premium handloom textiles from across India. Celebrating the art of traditional weaving.
            </Text>
            <Anchor
              href={INSTAGRAM_URL}
              target="_blank"
              rel="noopener noreferrer"
              size="sm"
              c="navy.7"
              fw={500}
              underline="never"
              aria-label="Follow Homechrome on Instagram"
            >
              <Group component="span" gap={6} wrap="nowrap" align="center">
                <InstagramIcon />
                Follow us
              </Group>
            </Anchor>
          </Stack>

          <FooterSection title="Shop">
            <FooterLink href="/products">All Products</FooterLink>
            <FooterLink href="/categories">Categories</FooterLink>
          </FooterSection>

          <FooterSection title="Customer">
            <FooterLink href="/track">Track Order</FooterLink>
            <FooterLink href="/account">My Account</FooterLink>
          </FooterSection>

          <FooterSection title="Need Help?">
            <Anchor
              href={`https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`}
              target="_blank"
              rel="noopener noreferrer"
              size="sm"
              c="navy.7"
              fw={500}
              underline="never"
              aria-label="Chat with us on WhatsApp"
            >
              <Group component="span" gap={6} wrap="nowrap" align="center">
                <WhatsAppIcon />
                WhatsApp us
              </Group>
            </Anchor>
            <Anchor
              href={`tel:${SUPPORT_PHONE}`}
              size="sm"
              c="navy.7"
              fw={500}
              underline="never"
              aria-label="Call to order"
            >
              <Group component="span" gap={6} wrap="nowrap" align="center">
                <PhoneIcon width={16} height={16} />
                Call to order
              </Group>
            </Anchor>
            <Text size="xs" c="dimmed">
              {SUPPORT_PHONE_DISPLAY}
            </Text>
          </FooterSection>
        </SimpleGrid>

        <Divider my="lg" />

        <Text ta="center" size="xs" c="dimmed">
          &copy; {new Date().getFullYear()} Homechrome. All rights reserved.
        </Text>
      </Container>
    </Box>
  );
}

function FooterSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Stack gap="xs">
      <Title order={3} size="xs" tt="uppercase" fw={600} style={{ letterSpacing: '0.05em' }}>
        {title}
      </Title>
      <Stack gap={4}>{children}</Stack>
    </Stack>
  );
}

function FooterLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Anchor component={Link} href={href} size="sm" c="dimmed" underline="never">
      {children}
    </Anchor>
  );
}

function WhatsAppIcon() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M19.05 4.91A9.816 9.816 0 0 0 12.04 2c-5.46 0-9.91 4.45-9.91 9.91 0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38c1.45.79 3.08 1.21 4.74 1.21h.01c5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.91-7.01zm-7.01 15.24c-1.48 0-2.93-.4-4.2-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.264 8.264 0 0 1-1.26-4.38c0-4.54 3.7-8.24 8.25-8.24 2.2 0 4.27.86 5.82 2.42a8.183 8.183 0 0 1 2.41 5.83c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.12-.17.25-.64.81-.78.97-.14.17-.29.19-.54.06-.25-.12-1.05-.39-1.99-1.23-.74-.66-1.23-1.47-1.38-1.72-.14-.25-.02-.38.11-.51.11-.11.25-.29.37-.43.13-.14.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.34-.76-1.84-.2-.48-.41-.42-.56-.43-.14-.01-.31-.01-.48-.01-.17 0-.43.06-.66.31-.22.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z" />
    </svg>
  );
}

function InstagramIcon() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z" />
    </svg>
  );
}
