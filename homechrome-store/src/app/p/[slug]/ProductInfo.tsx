'use client';

import { Box, Button, Divider, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core';

import { DiscountBadge } from '@/components/ui/discount-badge';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useProductCartActions } from '@/hooks/useProductCartActions';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
import { Product } from '@/types';

interface ProductInfoProps {
  product: Product;
}

export function ProductInfo({ product }: ProductInfoProps) {
  const {
    quantity,
    cartQty,
    loading,
    incrementQuantity,
    decrementQuantity,
    handleAdd,
    handleIncrement,
    handleDecrement,
  } = useProductCartActions(product);

  // API populates `base_price` (the MRP); `mrp` is a legacy alias the API
  // never sends, so reading it left the discount UI dead. Match the admin: use base_price.
  const hasDiscount = product.base_price > product.selling_price;
  const discountPercent = calculateDiscountPercent(product.base_price, product.selling_price);

  return (
    <Stack gap="md">
      <Stack gap={4}>
        <Title order={1} size="h2">{product.name}</Title>
        {product.sku && (
          <Text size="sm" c="dimmed">SKU: {product.sku}</Text>
        )}
      </Stack>

      <Group align="baseline" gap="sm">
        <Text size="2rem" fw={700} c="navy.7">
          {formatPrice(product.selling_price)}
        </Text>
        {hasDiscount && (
          <>
            <Text size="lg" c="dimmed" td="line-through">
              {formatPrice(product.base_price)}
            </Text>
            <DiscountBadge percent={discountPercent} variant="soft" />
          </>
        )}
      </Group>

      <StockStatus inStock={product.in_stock} />

      {product.description && (
        <Section title="Description">
          <Text c="dimmed" style={{ whiteSpace: 'pre-line', lineHeight: 1.6 }}>
            {product.description}
          </Text>
        </Section>
      )}

      {product.attributes && Object.keys(product.attributes).length > 0 && (
        <Section title="Details">
          <Stack gap="xs" mt="xs">
            {Object.entries(product.attributes).map(([key, value]) => (
              <SimpleGrid key={key} cols={2} spacing="md">
                <Text size="sm" fw={500} c="navy.7" tt="capitalize">
                  {key.replace(/_/g, ' ')}
                </Text>
                <Text size="sm" c="dimmed">{value}</Text>
              </SimpleGrid>
            ))}
          </Stack>
        </Section>
      )}

      <Divider my="md" />

      {cartQty > 0 ? (
        <Group align="center" gap="md">
          <QuantityStepper
            value={cartQty}
            onIncrement={handleIncrement}
            onDecrement={handleDecrement}
            disabled={loading}
            variant="primary"
            size="lg"
          />
          <Text size="sm" c="dimmed">in your cart</Text>
        </Group>
      ) : (
        <Group align="center" gap="md">
          <QuantityStepper
            value={quantity}
            onIncrement={incrementQuantity}
            onDecrement={decrementQuantity}
            disableDecrement={quantity <= 1}
          />
          <Button
            color="brand"
            size="lg"
            flex={1}
            onClick={handleAdd}
            loading={loading}
            disabled={!product.in_stock}
          >
            {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
          </Button>
        </Group>
      )}
    </Stack>
  );
}

function StockStatus({ inStock }: { inStock: boolean }) {
  const color = inStock ? 'success' : 'destructive';
  return (
    <Group gap={6} align="center">
      <Box
        w={8}
        h={8}
        style={{ borderRadius: '50%', backgroundColor: `var(--mantine-color-${inStock ? 'teal-7' : 'red-7'})` }}
      />
      <Text size="sm" fw={500} c={inStock ? 'teal.7' : 'red.7'}>
        {inStock ? 'In Stock' : 'Out of Stock'}
      </Text>
      <span style={{ display: 'none' }}>{color}</span>
    </Group>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Stack gap="xs">
      <Text size="xs" fw={600} tt="uppercase" c="navy.7" style={{ letterSpacing: '0.05em' }}>
        {title}
      </Text>
      {children}
    </Stack>
  );
}
