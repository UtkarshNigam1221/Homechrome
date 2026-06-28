'use client';

import { MagnifyingGlassIcon, TagIcon } from '@heroicons/react/24/outline';
import { Avatar, Group, Text, ThemeIcon } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { Spotlight, SpotlightActionData, SpotlightActionGroupData } from '@mantine/spotlight';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useMemo, useState } from 'react';

import HCLoader from '@/components/ui/HCLoader';
import { searchProducts } from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import { Category, Product } from '@/types';

import '@mantine/spotlight/styles.css';

interface SpotlightSearchProps {
  categories: Category[];
}

export function SpotlightSearch({ categories }: SpotlightSearchProps) {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [opened, setOpened] = useState(false);
  const [debounced] = useDebouncedValue(query, 300);

  const { data: products = [], isLoading } = useQuery<Product[]>({
    queryKey: ['spotlight-products', debounced],
    enabled: opened && debounced.trim().length >= 2,
    staleTime: 60_000,
    queryFn: async () => {
      const res = await searchProducts({ q: debounced.trim(), limit: 8 });
      return res.data;
    },
  });

  const actions = useMemo<(SpotlightActionData | SpotlightActionGroupData)[]>(() => {
    const productActions: SpotlightActionData[] = products.map((p) => ({
      id: `product-${p.id}`,
      label: p.name,
      description: formatPrice(p.selling_price),
      onClick: () => router.push(`/p/${p.slug}`),
      leftSection: (
        <Avatar
          src={p.images?.find((i) => i.is_primary)?.url || p.images?.[0]?.url}
          radius="md"
          size={36}
          alt={p.name}
        />
      ),
    }));

    const categoryActions: SpotlightActionData[] = categories.slice(0, 8).map((c) => ({
      id: `category-${c.id}`,
      label: c.name,
      description: 'Category',
      onClick: () => router.push(`/c/${c.slug}`),
      leftSection: (
        <ThemeIcon size={36} radius="md" variant="light" color="brand">
          <TagIcon width={18} height={18} />
        </ThemeIcon>
      ),
    }));

    if (debounced.trim().length >= 2) {
      // While the query is in flight, suppress the "view-all" action so the
      // actions list is empty and Mantine renders `nothingFound` (the loader).
      // Once the fetch settles, "view-all" is always appended so the list is
      // never empty when a real query is active.
      if (isLoading) {
        return [...productActions];
      }
      return [
        ...productActions,
        {
          id: 'view-all',
          label: `View all results for "${debounced.trim()}"`,
          description: 'See full product list',
          onClick: () => router.push(`/products?search=${encodeURIComponent(debounced.trim())}`),
          leftSection: (
            <ThemeIcon size={36} radius="md" variant="light" color="navy">
              <MagnifyingGlassIcon width={18} height={18} />
            </ThemeIcon>
          ),
        },
      ];
    }

    const groups: SpotlightActionGroupData[] = [];
    if (categoryActions.length > 0) {
      groups.push({ group: 'Browse categories', actions: categoryActions });
    }
    groups.push({
      group: 'Quick links',
      actions: [
        {
          id: 'all-products',
          label: 'All products',
          description: 'Browse the full catalog',
          onClick: () => router.push('/products'),
        },
        {
          id: 'cart',
          label: 'Cart',
          description: 'View your cart',
          onClick: () => router.push('/cart'),
        },
        {
          id: 'orders',
          label: 'My orders',
          description: 'Track your orders',
          onClick: () => router.push('/account/orders'),
        },
      ],
    });
    return groups;
  }, [products, categories, debounced, isLoading, router]);

  return (
    <Spotlight
      actions={actions}
      // Disable Mantine's client-side label/keyword filter — the embedder
      // already ranked these by semantic + tsvector + trigram relevance, so
      // re-filtering by exact substring match drops valid semantic matches
      // (e.g. query "wedding" returning "Bridal Lehenga").
      filter={(_query, actions) => actions}
      shortcut="mod + K"
      onSpotlightOpen={() => setOpened(true)}
      onSpotlightClose={() => {
        setOpened(false);
        setQuery('');
      }}
      nothingFound={
        isLoading && debounced.trim().length > 0 ? (
          <div className="flex w-full items-center justify-center py-6">
            <HCLoader size="sm" label="Searching" />
          </div>
        ) : (
          <Group justify="center" py="lg">
            <Text c="dimmed" size="sm">
              No products match &ldquo;{debounced.trim()}&rdquo;
            </Text>
          </Group>
        )
      }
      highlightQuery
      query={query}
      onQueryChange={setQuery}
      searchProps={{
        leftSection: <MagnifyingGlassIcon width={18} height={18} />,
        placeholder: 'Search handloom textiles, categories...',
      }}
      scrollable
      maxHeight={420}
    />
  );
}
