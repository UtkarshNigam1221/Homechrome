'use client';

import {
  ArrowRightIcon,
  ChevronDownIcon,
  ShoppingBagIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';
import {
  Accordion,
  Box,
  Button,
  Group,
  Stack,
  Text,
  Title,
} from '@mantine/core';

import { AssuranceRow } from '@/components/ui/assurance-row';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useProductCartActions } from '@/hooks/useProductCartActions';
import { DISPATCH_DAYS } from '@/lib/constants';
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
      <Group justify="space-between" align="center" gap="sm" wrap="nowrap">
        <Text fz="xs" fw={600} c="navy.6" tt="capitalize" lineClamp={1}>
          {[product.craft_type?.toLowerCase(), product.origin].filter(Boolean).join(' · ')}
          {product.sku ? `${product.craft_type || product.origin ? ' · ' : ''}SKU: ${product.sku}` : ''}
        </Text>
        <Group gap={5} wrap="nowrap" style={{ flexShrink: 0 }}>
          <TruckIcon width={14} height={14} color="var(--mantine-color-navy-6)" />
          <Text fz="xs" fw={600} c="navy.6" style={{ letterSpacing: '0.05em' }}>
            FAST DISPATCH
          </Text>
        </Group>
      </Group>

      <Title
        order={1}
        fz={{ base: '1.75rem', sm: '2.25rem' }}
        fw={600}
        lh={1.15}
        c="navy.9"
      >
        {product.name}
      </Title>

      <Box bg="navy.1" p="md" style={{ borderRadius: 'var(--mantine-radius-md)' }}>
        <Group align="baseline" gap="sm" wrap="wrap">
          <Text fz="2.25rem" fw={700} c="brand.5" lh={1}>
            {formatPrice(product.selling_price)}
          </Text>
          {hasDiscount && (
            <>
              <Text size="lg" c="dimmed" td="line-through">
                MRP {formatPrice(product.base_price)}
              </Text>
              <Box px={9} py={3} bg="brand.1" style={{ borderRadius: 4 }}>
                <Text fz="xs" fw={700} c="brand.6">
                  SAVE {formatPrice(product.base_price - product.selling_price)} ({discountPercent}% OFF)
                </Text>
              </Box>
            </>
          )}
        </Group>

        <Group gap="xs" mt={10} wrap="wrap">
          <StockStatus inStock={product.in_stock} />
          <Text size="sm" c="navy.6">
            · Dispatched in {DISPATCH_DAYS} business days · Inclusive of all taxes
          </Text>
        </Group>
      </Box>

      {product.description && (
        <Section title="The Craft Story">
          <Text size="md" c="gray.7" maw="62ch" style={{ whiteSpace: 'pre-line', lineHeight: 1.7 }}>
            {product.description}
          </Text>
        </Section>
      )}

      {specs.length > 0 && (
        <Section title="Specifications & Dimensions">
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

      <Group grow align="stretch" gap="sm" mt="xs">
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
            variant="default"
            size="lg"
            radius="sm"
            leftSection={<ShoppingBagIcon width={18} height={18} />}
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
          radius="sm"
          rightSection={<ArrowRightIcon width={18} height={18} />}
          onClick={handleBuyNow}
          loading={loading}
          disabled={!product.in_stock}
        >
          Buy Now
        </Button>
      </Group>

      <Box mt="xs">
        <AssuranceRow variant="cards" copy="short" columns={2} />
      </Box>
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
      <Title
        order={2}
        fz="1.25rem"
        fw={600}
        c="navy.9"
      >
        {title}
      </Title>
      {children}
    </Stack>
  );
}
