'use client';

import { ChevronDownIcon } from '@heroicons/react/24/outline';
import { Accordion, Box, Button, Divider, Group, Stack, Text, Title } from '@mantine/core';

import { DiscountBadge } from '@/components/ui/discount-badge';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useProductCartActions } from '@/hooks/useProductCartActions';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
import { Product } from '@/types';

type Spec = { label: string; value: string };

const COMMON_ATTRS: [keyof Product, string][] = [
  ['material', 'Material'],
  // color intentionally absent: it is multi-valued and arrives via the
  // attributes map, rendered by the dynamic loop below.
  ['weave_type', 'Weave type'],
  ['origin', 'Origin'],
  ['craft_type', 'Craft type'],
];

function formatDimensions(d?: Product['dimensions']): string {
  if (!d) return '';
  const parts = [d.length, d.width, d.height].filter(
    (n): n is number => typeof n === 'number' && n > 0,
  );
  return parts.length ? `${parts.join(' × ')} ${d.unit}` : '';
}

// Combine backend's top-level common attrs + dimensions/weight + free-form attributes map.
// Commons live on top-level fields (excluded from the attributes map by the API), so they
// must be read explicitly — that's why they never rendered before.
function buildSpecs(p: Product): Spec[] {
  const out: Spec[] = [];
  for (const [key, label] of COMMON_ATTRS) {
    const v = p[key];
    if (typeof v === 'string' && v.trim()) out.push({ label, value: v });
  }
  const dim = formatDimensions(p.dimensions);
  if (dim) out.push({ label: 'Dimensions', value: dim });
  if (p.weight && p.weight > 0) out.push({ label: 'Weight', value: `${p.weight} g` });
  for (const [key, v] of Object.entries(p.attributes || {})) {
    const value = Array.isArray(v) ? v.filter(Boolean).join(', ') : v;
    if (value && value.trim()) out.push({ label: key.replace(/_/g, ' '), value });
  }
  return out;
}

interface ProductInfoProps {
  product: Product;
}

export function ProductInfo({ product }: ProductInfoProps) {
  const {
    cartQty,
    loading,
    handleAdd,
    handleBuyNow,
    handleIncrement,
    handleDecrement,
  } = useProductCartActions(product);

  // API populates `base_price` (the MRP); `mrp` is a legacy alias the API
  // never sends, so reading it left the discount UI dead. Match the admin: use base_price.
  const hasDiscount = product.base_price > product.selling_price;
  const discountPercent = calculateDiscountPercent(product.base_price, product.selling_price);
  const specs = buildSpecs(product);

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

      {hasDiscount && (
        <Text size="sm" fw={600} c="teal.7">
          You save {formatPrice(product.base_price - product.selling_price)} ({discountPercent}% off)
        </Text>
      )}

      <StockStatus inStock={product.in_stock} />

      {product.description && (
        <Section title="Description">
          <Text size="md" c="gray.7" maw="62ch" style={{ whiteSpace: 'pre-line', lineHeight: 1.7 }}>
            {product.description}
          </Text>
        </Section>
      )}

      {specs.length > 0 && (
        <Section title="Details">
          <Accordion
            multiple
            variant="separated"
            chevronPosition="right"
            chevron={<ChevronDownIcon width={18} height={18} strokeWidth={2} />}
            mt="xs"
          >
            {specs.map((s) => (
              <Accordion.Item key={s.label} value={s.label}>
                <Accordion.Control>
                  <Text size="sm" fw={500} c="navy.7" tt="capitalize">{s.label}</Text>
                </Accordion.Control>
                <Accordion.Panel>
                  <Text size="sm" c="dimmed">{s.value}</Text>
                </Accordion.Panel>
              </Accordion.Item>
            ))}
          </Accordion>
        </Section>
      )}

      <Divider my="md" />

      <Group grow align="stretch" gap="sm">
        {cartQty > 0 ? (
          <QuantityStepper
            value={cartQty}
            onIncrement={handleIncrement}
            onDecrement={handleDecrement}
            disabled={loading}
            variant="primary"
            size="lg"
            fullWidth
          />
        ) : (
          <Button
            variant="light"
            color="brand"
            size="lg"
            onClick={handleAdd}
            loading={loading}
            disabled={!product.in_stock}
          >
            {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
          </Button>
        )}
        <Button
          color="brand"
          size="lg"
          onClick={handleBuyNow}
          loading={loading}
          disabled={!product.in_stock}
        >
          Buy Now
        </Button>
      </Group>
    </Stack>
  );
}

function StockStatus({ inStock }: { inStock: boolean }) {
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
    </Group>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Stack gap="sm">
      <Title order={2} size="h4" c="navy.7">
        {title}
      </Title>
      {children}
    </Stack>
  );
}
