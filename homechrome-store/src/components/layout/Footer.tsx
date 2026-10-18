'use client';

import { EnvelopeIcon, PhoneIcon } from '@heroicons/react/24/outline';
import {
  ActionIcon,
  Anchor,
  Box,
  Container,
  Flex,
  Group,
  SimpleGrid,
  Stack,
  Text,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import logo80 from '@/assets/logo-80.webp';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import {
  INSTAGRAM_URL,
  SUPPORT_EMAIL,
  SUPPORT_PHONE,
  SUPPORT_PHONE_TEL,
  SUPPORT_WHATSAPP,
} from '@/lib/constants';

export default function Footer() {
  return (
    <Box
      component="footer"
      bg="gray.0"
      mt={64}
      style={{ borderTop: '1px solid var(--mantine-color-gray-2)' }}
    >
      <Container size="xl">
        {/* Top: brand left, link groups right */}
        <Flex
          py={48}
          gap="xl"
          direction={{ base: 'column', sm: 'row' }}
          justify="space-between"
          align={{ base: 'center', sm: 'flex-start' }}
        >
          <Flex direction="column" gap={6} maw={280} align={{ base: 'center', sm: 'flex-start' }}>
            <Anchor component={Link} href="/" underline="never" w="fit-content">
              <Group gap={10} align="center" wrap="nowrap">
                <Image src={logo80} alt="Homechrome" style={{ height: 36, width: 'auto' }} unoptimized />
                <Text fw={700} size="lg" c="navy.7" style={{ letterSpacing: '-0.01em' }}>
                  HOME<Text span c="brand">CHROME</Text>
                </Text>
              </Group>
            </Anchor>
            <Text size="sm" c="dimmed" ta={{ base: 'center', sm: 'left' }}>
              Premium handloom textiles from across India. Celebrating the art of traditional weaving.
            </Text>
          </Flex>

          {/* Links render on every breakpoint: legal/policy pages must be
              reachable on mobile (E-Commerce Rules disclosure requirement),
              and most buyers are on phones. */}
          <SimpleGrid
            cols={{ base: 2, sm: 4 }}
            spacing={{ base: 24, sm: 48 }}
            verticalSpacing="lg"
          >
            <FooterColumn title="Shop">
              <FooterLink href="/products">All Products</FooterLink>
              <FooterLink href="/categories">Categories</FooterLink>
            </FooterColumn>

            <FooterColumn title="Customer">
              <FooterLink href="/track">Track Order</FooterLink>
              <FooterLink href="/account">My Account</FooterLink>
            </FooterColumn>

            <FooterColumn title="Policies">
              <FooterLink href="/privacy-policy">Privacy Policy</FooterLink>
              <FooterLink href="/terms">Terms &amp; Conditions</FooterLink>
              <FooterLink href="/refund-policy">Refund &amp; Replacement</FooterLink>
              <FooterLink href="/shipping-policy">Shipping &amp; Delivery</FooterLink>
              <FooterLink href="/contact#grievance">Grievance Redressal</FooterLink>
            </FooterColumn>

            <FooterColumn title="Need Help?">
              <FooterLink
                href={`https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`}
                icon={<WhatsAppIcon size={16} />}
              >
                WhatsApp us
              </FooterLink>
              <FooterLink href={`mailto:${SUPPORT_EMAIL}`} icon={<EnvelopeIcon width={15} height={15} />}>
                Email us
              </FooterLink>
              {/* The number itself is the label: payment-gateway review wants a
                  contact number visible on the page, not behind a "Call us". */}
              <FooterLink href={`tel:${SUPPORT_PHONE_TEL}`} icon={<PhoneIcon width={15} height={15} />}>
                {SUPPORT_PHONE}
              </FooterLink>
            </FooterColumn>
          </SimpleGrid>
        </Flex>

        {/* Bottom bar: copyright + social */}
        <Flex
          py="xl"
          gap="sm"
          direction={{ base: 'column', sm: 'row' }}
          justify="space-between"
          align="center"
          style={{ borderTop: '1px solid var(--mantine-color-gray-2)' }}
        >
          <Text size="sm" c="dimmed">
            &copy; {new Date().getFullYear()} Homechrome. All rights reserved.
          </Text>
          <ActionIcon
            component="a"
            href={INSTAGRAM_URL}
            target="_blank"
            rel="noopener noreferrer"
            size="lg"
            color="gray"
            variant="subtle"
            aria-label="Follow Homechrome on Instagram"
          >
            <InstagramIcon />
          </ActionIcon>
        </Flex>
      </Container>
    </Box>
  );
}

function FooterColumn({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Stack gap="xs">
      <Text fw={500} size="lg" c="navy.7">
        {title}
      </Text>
      <Stack gap={6}>{children}</Stack>
    </Stack>
  );
}

// Dimmed links with hover-underline — the FooterLinks finesse; the group title
// carries the contrast, links stay quiet until hovered.
const LINK_PROPS = { size: 'sm', c: 'dimmed', underline: 'hover' } as const;

// One footer link helper: internal routes use Next Link; raw hrefs (mailto:,
// wa.me) use a plain anchor and open http(s) in a new tab. Pass `icon` for the
// icon+label row.
function FooterLink({
  href,
  children,
  icon,
}: {
  href: string;
  children: React.ReactNode;
  icon?: React.ReactNode;
}) {
  const content = icon ? (
    <Group component="span" gap={8} wrap="nowrap" align="center">
      {icon}
      {children}
    </Group>
  ) : (
    children
  );

  if (href.startsWith('/')) {
    return <Anchor component={Link} href={href} {...LINK_PROPS}>{content}</Anchor>;
  }
  const newTab = href.startsWith('http') ? { target: '_blank', rel: 'noopener noreferrer' } : {};
  return <Anchor href={href} {...newTab} {...LINK_PROPS}>{content}</Anchor>;
}

function InstagramIcon() {
  return (
    <svg width={20} height={20} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z" />
    </svg>
  );
}
