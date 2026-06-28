'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import {
  AspectRatio,
  Badge,
  Box,
  Button,
  Card,
  Center,
  Group,
  Stack,
  Text,
} from '@mantine/core';
import { AssetImage } from '@/components/ui/asset-image';
import Link from 'next/link';
import { useState } from 'react';

import { DiscountBadge } from '@/components/ui/discount-badge';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useCart } from '@/hooks/useCart';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
import { useCartStore } from '@/stores/cart';
import { Product } from '@/types';

interface ProductCardProps {
  product: Product;
}

export default function ProductCard({ product }: ProductCardProps) {
  const [loading, setLoading] = useState(false);
  const { addItem, updateQuantity, removeItem } = useCart();
  const cartQty = useCartStore((s) => s.getQuantity(product.id));

  const primaryImage = product.images?.find((img) => img.is_primary) || product.images?.[0];
  // API populates `base_price` (the MRP); `mrp` is a legacy alias the API
  // never sends, so reading it left the discount UI dead. Match the admin: use base_price.
  const hasDiscount = product.base_price > product.selling_price;
  const discountPercent = calculateDiscountPercent(product.base_price, product.selling_price);

  const handleAddToCart = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setLoading(true);
    try {
      await addItem(product.id, 1);
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const handleIncrement = async () => {
    setLoading(true);
    try {
      await updateQuantity(product.id, cartQty + 1);
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const handleDecrement = async () => {
    setLoading(true);
    try {
      if (cartQty <= 1) {
        await removeItem(product.id);
      } else {
        await updateQuantity(product.id, cartQty - 1);
      }
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const handleRemove = async () => {
    setLoading(true);
    try {
      await removeItem(product.id);
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card shadow="sm" padding={0} radius="lg" withBorder={false}>
      <Card.Section component={Link} href={`/p/${product.slug}`} pos="relative">
        <AspectRatio ratio={1} bg="gray.1">
          {primaryImage ? (
            <AssetImage
              src={primaryImage.url}
              alt={primaryImage.alt_text || product.name}
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
              width={640}
              height={640}
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            />
          ) : (
            <Center bg="brand.1" h="100%">
              <PhotoIcon width={48} height={48} color="var(--mantine-color-brand-5)" opacity={0.4} />
            </Center>
          )}
        </AspectRatio>
        {hasDiscount && (
          <Box pos="absolute" top={8} left={8}>
            <DiscountBadge percent={discountPercent} variant="solid" />
          </Box>
        )}
        {!product.in_stock && (
          <Center pos="absolute" inset={0} bg="rgba(28,41,81,0.4)">
            <Badge color="white" c="navy.7" radius="sm" size="md">
              Out of Stock
            </Badge>
          </Center>
        )}
      </Card.Section>

      <Stack p="md" gap="xs">
        <Link href={`/p/${product.slug}`} style={{ textDecoration: 'none' }}>
          <Text size="sm" fw={500} c="navy.7" lineClamp={2}>
            {product.name}
          </Text>
        </Link>

        <Group align="baseline" gap="xs">
          <Text size="lg" fw={700} c="navy.7">
            {formatPrice(product.selling_price)}
          </Text>
          {hasDiscount && (
            <Text size="sm" c="dimmed" td="line-through">
              {formatPrice(product.base_price)}
            </Text>
          )}
        </Group>

        <Box mt="auto" pt="xs">
          {cartQty > 0 ? (
            product.in_stock ? (
              <QuantityStepper
                value={cartQty}
                onIncrement={handleIncrement}
                onDecrement={handleDecrement}
                disabled={loading}
                variant="primary"
                fullWidth
              />
            ) : (
              <Button
                variant="light"
                color="red"
                size="sm"
                fullWidth
                onClick={handleRemove}
                loading={loading}
              >
                Remove (Out of Stock)
              </Button>
            )
          ) : (
            <Button
              variant="filled"
              color="brand"
              size="sm"
              fullWidth
              onClick={handleAddToCart}
              loading={loading}
              disabled={!product.in_stock}
            >
              {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
            </Button>
          )}
        </Box>
      </Stack>
    </Card>
  );
}
