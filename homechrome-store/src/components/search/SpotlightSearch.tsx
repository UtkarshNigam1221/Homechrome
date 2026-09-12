'use client';

import { ArrowRightIcon, ClockIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { Box, Divider, Flex, Group, Stack, Text } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { Spotlight, spotlight } from '@mantine/spotlight';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

import { AssetImage } from '@/components/ui/asset-image';
import { DiscountBadge } from '@/components/ui/discount-badge';
import HCLoader from '@/components/ui/HCLoader';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import { buildSearchURL, fetchProductsPage } from '@/lib/api';
import { SUPPORT_WHATSAPP } from '@/lib/constants';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
import { Category, Product } from '@/types';

import { displayFont } from '@/app/fonts';

import '@mantine/spotlight/styles.css';

const RECENT_KEY = 'hc_recent_searches';
const MAX_RECENT = 4;

function readRecent(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    return raw ? (JSON.parse(raw) as string[]).slice(0, MAX_RECENT) : [];
  } catch {
    return [];
  }
}

function pushRecent(term: string) {
  try {
    const next = [term, ...readRecent().filter((t) => t !== term)].slice(0, MAX_RECENT);
    localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    /* private mode — recents are a convenience, not state we depend on */
  }
}

interface SpotlightSearchProps {
  categories: Category[];
}

export function SpotlightSearch({ categories }: SpotlightSearchProps) {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [opened, setOpened] = useState(false);
  const [recent, setRecent] = useState<string[]>([]);
  const [debounced] = useDebouncedValue(query, 300);

  const term = debounced.trim();
  const active = term.length >= 2;

  const { data: products = [], isLoading } = useQuery<Product[]>({
    queryKey: ['spotlight-products', term],
    enabled: opened && active,
    staleTime: 60_000,
    queryFn: async () => {
      const { products } = await fetchProductsPage(buildSearchURL({ q: term, limit: 6 }));
      return products;
    },
  });

  const go = useCallback(
    (href: string, remember?: string) => {
      if (remember) pushRecent(remember);
      spotlight.close();
      router.push(href);
    },
    [router],
  );

  return (
    <Spotlight.Root
      // No client-side filtering: the embedder already ranked these by semantic
      // relevance, and this list is rendered from that result directly.
      shortcut="mod + K"
      onSpotlightOpen={() => {
        setOpened(true);
        // Reading on open, not in an effect: this is an event, not a sync.
        setRecent(readRecent());
      }}
      onSpotlightClose={() => {
        setOpened(false);
        setQuery('');
      }}
      query={query}
      onQueryChange={setQuery}
      size={900}
    >
      <Spotlight.Search
        placeholder="Search handloom textiles, categories..."
        leftSection={<MagnifyingGlassIcon width={18} height={18} />}
      />

      <Flex align="stretch" direction={{ base: 'column', md: 'row' }}>
        <Box flex={1} miw={0}>
          <Spotlight.ActionsList mah={440} style={{ overflowY: 'auto' }}>
            {active && isLoading && (
              <Group justify="center" py="xl">
                <HCLoader size="sm" label="Searching" />
              </Group>
            )}

            {active && !isLoading && products.length === 0 && (
              <Group justify="center" py="xl">
                <Text c="dimmed" size="sm">
                  No products match &ldquo;{term}&rdquo;
                </Text>
              </Group>
            )}

            {active &&
              !isLoading &&
              products.map((p) => {
                const image = p.images?.find((i) => i.is_primary) || p.images?.[0];
                const hasDiscount = p.base_price > p.selling_price;
                const meta = [p.material, p.weave_type].filter(Boolean).join(' · ');
                return (
                  <Spotlight.Action
                    key={p.id}
                    onClick={() => go(`/p/${p.slug}`, term)}
                    style={{ alignItems: 'stretch' }}
                  >
                    <Group gap="sm" wrap="nowrap" w="100%">
                      <Box
                        w={64}
                        h={64}
                        bg="navy.2"
                        style={{ borderRadius: 8, overflow: 'hidden', flexShrink: 0 }}
                      >
                        {image && (
                          <AssetImage
                            src={image.url}
                            alt={image.alt_text || p.name}
                            sizes="64px"
                            width={128}
                            height={128}
                            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                          />
                        )}
                      </Box>

                      <Stack gap={3} flex={1} miw={0}>
                        {meta && (
                          <Text fz={10} fw={700} c="brand.5" tt="uppercase" lineClamp={1}>
                            {meta}
                          </Text>
                        )}
                        <Text
                          fz="sm"
                          fw={600}
                          c="navy.9"
                          lineClamp={1}
                          style={{ fontFamily: displayFont.style.fontFamily }}
                        >
                          {p.name}
                        </Text>
                        <Group gap={7} wrap="nowrap">
                          <Text fz="sm" fw={700} c="navy.9">
                            {formatPrice(p.selling_price)}
                          </Text>
                          {hasDiscount && (
                            <>
                              <Text fz="xs" c="dimmed" td="line-through">
                                {formatPrice(p.base_price)}
                              </Text>
                              <DiscountBadge
                                percent={calculateDiscountPercent(p.base_price, p.selling_price)}
                              />
                            </>
                          )}
                          <Text fz={10} fw={600} c={p.in_stock ? '#2E4E37' : 'dimmed'}>
                            {p.in_stock ? 'In stock' : 'Out of stock'}
                          </Text>
                        </Group>
                      </Stack>

                      {/* The design puts an Add button on each row, but a
                          Spotlight action is itself a <button> and nesting one
                          is invalid — it broke hydration and ate the list.
                          Keyboard navigation is worth more than the shortcut. */}
                      <ArrowRightIcon
                        width={15}
                        height={15}
                        color="var(--mantine-color-navy-5)"
                        style={{ flexShrink: 0 }}
                      />
                    </Group>
                  </Spotlight.Action>
                );
              })}

            {active && !isLoading && products.length > 0 && (
              <Spotlight.Action onClick={() => go(`/products?search=${encodeURIComponent(term)}`, term)}>
                <Group justify="space-between" w="100%" wrap="nowrap">
                  <Text fz="sm" fw={600} c="brand.5">
                    View all results for &ldquo;{term}&rdquo;
                  </Text>
                  <ArrowRightIcon width={16} height={16} color="var(--mantine-color-brand-5)" />
                </Group>
              </Spotlight.Action>
            )}

            {!active &&
              categories.slice(0, 6).map((c) => (
                <Spotlight.Action key={c.id} onClick={() => go(`/c/${c.slug}`)}>
                  <Group justify="space-between" w="100%" wrap="nowrap">
                    <Text fz="sm" fw={500} c="navy.8">
                      {c.name}
                    </Text>
                    <Text fz="xs" c="navy.5">
                      {c.product_count} designs
                    </Text>
                  </Group>
                </Spotlight.Action>
              ))}
          </Spotlight.ActionsList>
        </Box>

        <Box
          w={280}
          flex="none"
          p="md"
          bg="navy.1"
          visibleFrom="md"
          style={{ borderLeft: '1px solid var(--mantine-color-navy-2)' }}
        >
          <Stack gap="lg">
            {recent.length > 0 && (
              <Stack gap={6}>
                <Text fz={10} fw={700} c="navy.5" style={{ letterSpacing: '0.1em' }}>
                  RECENT SEARCHES
                </Text>
                {recent.map((r) => (
                  <Group
                    key={r}
                    gap={7}
                    wrap="nowrap"
                    style={{ cursor: 'pointer' }}
                    onClick={() => setQuery(r)}
                  >
                    <ClockIcon width={13} height={13} color="var(--mantine-color-navy-5)" />
                    <Text fz="xs" c="navy.7" lineClamp={1}>
                      {r}
                    </Text>
                  </Group>
                ))}
                <Divider mt={4} />
              </Stack>
            )}

            <Stack gap={8}>
              <Text fz={10} fw={700} c="navy.5" style={{ letterSpacing: '0.1em' }}>
                POPULAR COLLECTIONS
              </Text>
              {categories.slice(0, 3).map((c) => (
                <Group
                  key={c.id}
                  gap="sm"
                  wrap="nowrap"
                  style={{ cursor: 'pointer' }}
                  onClick={() => go(`/c/${c.slug}`)}
                >
                  <Box
                    w={30}
                    h={30}
                    bg="brand.1"
                    style={{ borderRadius: 8, display: 'grid', placeItems: 'center', flexShrink: 0 }}
                  >
                    <Text fz="xs" fw={700} c="brand.6">
                      {c.name.charAt(0)}
                    </Text>
                  </Box>
                  <Stack gap={0} miw={0}>
                    <Text fz="xs" fw={600} c="navy.9" lineClamp={1}>
                      {c.name}
                    </Text>
                    <Text fz={10} c="navy.6">
                      {c.product_count} designs
                    </Text>
                  </Stack>
                </Group>
              ))}
            </Stack>

            <Box p="sm" bg="white" style={{ borderRadius: 'var(--mantine-radius-md)' }}>
              <Text
                fz="sm"
                fw={600}
                c="navy.9"
                mb={4}
                style={{ fontFamily: displayFont.style.fontFamily }}
              >
                Can&rsquo;t find it?
              </Text>
              <Text fz={10} c="navy.6" lh={1.55} mb="xs">
                Tell us the size or weave you are after and we will point you to it.
              </Text>
              <Group
                gap={6}
                justify="center"
                c="white"
                py={7}
                style={{
                  background: '#42634C',
                  borderRadius: 'var(--mantine-radius-sm)',
                  cursor: 'pointer',
                }}
                onClick={() =>
                  window.open(
                    `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I am looking for something specific')}`,
                    '_blank',
                    'noopener,noreferrer',
                  )
                }
              >
                <WhatsAppIcon size={14} />
                <Text fz={11} fw={600} c="white">
                  Ask on WhatsApp
                </Text>
              </Group>
            </Box>
          </Stack>
        </Box>
      </Flex>
    </Spotlight.Root>
  );
}
