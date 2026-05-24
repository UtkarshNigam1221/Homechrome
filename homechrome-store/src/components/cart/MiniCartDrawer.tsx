'use client';

import { PhotoIcon, ShoppingBagIcon } from '@heroicons/react/24/outline';
import {
  ActionIcon,
  AspectRatio,
  Box,
  Button,
  Center,
  Divider,
  Drawer,
  Group,
  ScrollArea,
  Skeleton,
  Stack,
  Text,
  ThemeIcon,
} from '@mantine/core';
import Link from 'next/link';

import { AssetImage } from '@/components/ui/asset-image';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useCart } from '@/hooks/useCart';
import { formatPrice } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/uiStore';

export function MiniCartDrawer() {
  const isOpen = useUIStore((s) => s.miniCartOpen);
  const close = useUIStore((s) => s.closeMiniCart);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { cart, loading, updateQuantity, removeItem } = useCart();
  const items = cart?.items ?? [];
  const subtotal = cart?.cart.subtotal ?? 0;
  const itemCount = cart?.cart.item_count ?? 0;
  const showSkeleton = loading && !cart;

  return (
    <Drawer
      opened={isOpen}
      onClose={close}
      position="right"
      size={420}
      title={
        <Group gap="xs">
          <ThemeIcon size="md" radius="xl" variant="light" color="brand">
            <ShoppingBagIcon width={16} height={16} />
          </ThemeIcon>
          <Text fw={600}>Your Cart</Text>
          {itemCount > 0 && (
            <Text size="sm" c="dimmed">
              ({itemCount} {itemCount === 1 ? 'item' : 'items'})
            </Text>
          )}
        </Group>
      }
      padding="md"
      scrollAreaComponent={ScrollArea.Autosize}
    >
      {showSkeleton ? (
        <Stack gap="md">
          {[1, 2, 3].map((i) => (
            <Group key={i} gap="sm" wrap="nowrap" align="flex-start">
              <Skeleton h={64} w={64} radius="md" />
              <Stack gap={4} flex={1}>
                <Skeleton h={14} w="75%" />
                <Skeleton h={12} w="40%" />
                <Skeleton h={28} mt="xs" />
              </Stack>
            </Group>
          ))}
        </Stack>
      ) : items.length === 0 ? (
        <Stack align="center" justify="center" gap="md" py="xl">
          <ThemeIcon size={64} radius="xl" variant="light" color="brand">
            <ShoppingBagIcon width={32} height={32} />
          </ThemeIcon>
          <Text c="dimmed" ta="center">
            Your cart is empty.
          </Text>
          <Button component={Link} href="/products" color="brand" onClick={close}>
            Start Shopping
          </Button>
        </Stack>
      ) : (
        <Stack gap="md">
          {items.map((item) => (
            <Box key={item.product_id}>
              <Group gap="sm" wrap="nowrap" align="flex-start">
                <Box w={64} flex="none">
                  <AspectRatio ratio={1}>
                    <Box
                      bg="gray.1"
                      style={{
                        overflow: 'hidden',
                        borderRadius: 'var(--mantine-radius-md)',
                      }}
                    >
                      {item.product_image ? (
                        <AssetImage
                          src={item.product_image}
                          alt={item.product_name}
                          sizes="64px"
                          width={64}
                          height={64}
                          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                        />
                      ) : (
                        <Center bg="brand.1" h="100%">
                          <PhotoIcon width={24} height={24} color="var(--mantine-color-brand-5)" opacity={0.4} />
                        </Center>
                      )}
                    </Box>
                  </AspectRatio>
                </Box>
                <Stack gap={4} flex={1}>
                  <Text size="sm" fw={500} c="navy.7" lineClamp={2}>
                    {item.product_name}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {formatPrice(item.unit_price)} each
                  </Text>
                  <Group justify="space-between" align="center">
                    <QuantityStepper
                      value={item.quantity}
                      onIncrement={async () => {
                        try {
                          await updateQuantity(item.product_id, item.quantity + 1);
                        } catch {
                          /* useCart shows error toast */
                        }
                      }}
                      onDecrement={async () => {
                        try {
                          await updateQuantity(item.product_id, item.quantity - 1);
                        } catch {
                          /* useCart shows error toast */
                        }
                      }}
                      disableDecrement={item.quantity <= 1}
                      size="sm"
                    />
                    <Text size="sm" fw={700} c="navy.7">
                      {formatPrice(item.total_price)}
                    </Text>
                  </Group>
                </Stack>
                <ActionIcon
                  variant="subtle"
                  color="red"
                  size="sm"
                  aria-label={`Remove ${item.product_name}`}
                  onClick={async () => {
                    try {
                      await removeItem(item.product_id);
                    } catch {
                      /* useCart shows error toast */
                    }
                  }}
                >
                  <Text size="xs" aria-hidden>
                    ×
                  </Text>
                </ActionIcon>
              </Group>
              <Divider mt="md" />
            </Box>
          ))}

          <Group justify="space-between">
            <Text c="dimmed">Subtotal</Text>
            <Text fw={700} c="navy.7">
              {formatPrice(subtotal)}
            </Text>
          </Group>

          <Stack gap="xs">
            <Button
              component={Link}
              href={isAuthenticated ? '/checkout' : '/login?redirect=/cart'}
              color="brand"
              size="md"
              fullWidth
              onClick={close}
            >
              {isAuthenticated ? 'Checkout' : 'Sign in to Checkout'}
            </Button>
            <Button
              component={Link}
              href="/cart"
              variant="subtle"
              color="navy"
              size="sm"
              fullWidth
              onClick={close}
            >
              View Cart
            </Button>
          </Stack>
        </Stack>
      )}
    </Drawer>
  );
}
