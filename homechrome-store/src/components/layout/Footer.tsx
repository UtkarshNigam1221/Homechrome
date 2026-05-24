'use client';

import {
  Anchor,
  Box,
  Container,
  Divider,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Image from 'next/image';
import Link from 'next/link';

import logo80 from '@/assets/logo-80.png';

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
          </Stack>

          <FooterSection title="Shop">
            <FooterLink href="/products">All Products</FooterLink>
            <FooterLink href="/categories">Categories</FooterLink>
          </FooterSection>

          <FooterSection title="Customer">
            <FooterLink href="/track">Track Order</FooterLink>
            <FooterLink href="/account">My Account</FooterLink>
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
