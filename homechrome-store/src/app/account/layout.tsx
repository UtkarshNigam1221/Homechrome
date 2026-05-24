'use client';

import { Box, Button, Card, Container, Grid, NavLink, Stack, Title } from '@mantine/core';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';

import { LoadingBlock } from '@/components/ui/loading-spinner';
import { useAuthStore } from '@/stores/auth';

const navItems = [
  { label: 'My Account', href: '/account' },
  { label: 'Orders', href: '/account/orders' },
  { label: 'Addresses', href: '/account/addresses' },
];

export default function AccountLayout({
  children,
}: Readonly<{
    children: React.ReactNode;
}>) {
  const pathname = usePathname();
  const router = useRouter();
  const { isAuthenticated, isLoading, logout } = useAuthStore();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace('/login?redirect=/account');
    }
  }, [isLoading, isAuthenticated, router]);

  const handleLogout = async () => {
    await logout();
    router.replace('/');
  };

  if (isLoading || !isAuthenticated) {
    return <LoadingBlock />;
  }

  return (
    <Container size="lg" py="lg">
      <Title order={1} mb="lg" size="h2">My Account</Title>

      <Grid gap="xl">
        <Grid.Col span={{ base: 12, lg: 3 }}>
          <Card component="aside" shadow="sm" radius="lg" padding="sm">
            <Stack gap={4}>
              {navItems.map((item) => {
                const isActive =
                  item.href === '/account'
                    ? pathname === '/account'
                    : pathname.startsWith(item.href);

                return (
                  <NavLink
                    key={item.href}
                    component={Link}
                    href={item.href}
                    label={item.label}
                    active={isActive}
                    variant="light"
                    color="brand"
                  />
                );
              })}
              <Box mt="xs">
                <Button
                  variant="light"
                  color="red"
                  fullWidth
                  justify="start"
                  onClick={handleLogout}
                >
                  Logout
                </Button>
              </Box>
            </Stack>
          </Card>
        </Grid.Col>

        <Grid.Col span={{ base: 12, lg: 9 }} component="main">
          {children}
        </Grid.Col>
      </Grid>
    </Container>
  );
}
