'use client';

import { PhotoIcon, ShoppingBagIcon } from '@heroicons/react/24/outline';
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
import { displayFont } from '@/app/fonts';
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
  // The mockup's cluster chip, sourced from the product's own fields so it
  // never asserts a provenance the catalogue does not record.
  const provenance = product.craft_type || product.weave_type || product.material;

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
        {(hasDiscount || provenance) && (
          <Stack pos="absolute" top={8} left={8} gap={6} align="flex-start">
            {hasDiscount && <DiscountBadge percent={discountPercent} variant="solid" />}
            {provenance && (
              <Box px={8} py={3} bg="rgba(252,249,244,0.93)" style={{ borderRadius: 4 }}>
                <Text fz={10} fw={700} c="navy.8" tt="capitalize" lineClamp={1}>
                  {provenance.toLowerCase()}
                </Text>
              </Box>
            )}
          </Stack>
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
          <Text
            fz="md"
            fw={600}
            c="navy.9"
            lineClamp={2}
            style={{ minHeight: '2lh', fontFamily: displayFont.style.fontFamily }}
          >
            {product.name}
          </Text>
        </Link>

        {product.description && (
          <Text fz="xs" c="navy.6" lineClamp={1} mt={-4}>
            {product.description.replace(/[*_`#]/g, '').trim()}
          </Text>
        )}

        <Group align="baseline" gap="xs" wrap="nowrap">
          <Text fz="xl" fw={700} c="navy.9">
            {formatPrice(product.selling_price)}
          </Text>
          {hasDiscount && (
            <>
              <Text size="sm" c="dimmed" td="line-through">
                {formatPrice(product.base_price)}
              </Text>
              <Text size="xs" fw={600} c="brand.6" style={{ whiteSpace: 'nowrap' }}>
                Save {formatPrice(product.base_price - product.selling_price)}
              </Text>
            </>
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
              radius="sm"
              fullWidth
              leftSection={
                product.in_stock ? <ShoppingBagIcon width={16} height={16} /> : undefined
              }
              onClick={handleAddToCart}
              loading={loading}
              disabled={!product.in_stock}
            >
              {product.in_stock ? 'Add to Bag' : 'Out of Stock'}
            </Button>
          )}
        </Box>
      </Stack>
    </Card>
  );
}
