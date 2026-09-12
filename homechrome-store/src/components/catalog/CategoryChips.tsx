'use client';

import { Anchor, Box, Group, Text } from '@mantine/core';
import Link from 'next/link';

import { Category } from '@/types';

/** Chips navigate to each collection's own route rather than filtering in place. */
export function CategoryChips({ categories }: { categories: Category[] }) {
  return (
    <Group gap="xs" mb="xl" wrap="wrap">
      <Box px={16} py={8} bg="brand.5" style={{ borderRadius: 999 }}>
        <Text fz="sm" fw={600} c="white">
          All Categories ({categories.length})
        </Text>
      </Box>
      {categories.map((category) => (
        <Anchor key={category.id} component={Link} href={`/c/${category.slug}`} underline="never">
          <Box
            px={16}
            py={8}
            bg="white"
            style={{ borderRadius: 999, border: '1px solid var(--mantine-color-navy-2)' }}
          >
            <Text fz="sm" fw={500} c="navy.8">
              {category.name}{' '}
              <Text span c="navy.5" inherit>
                ({category.product_count})
              </Text>
            </Text>
          </Box>
        </Anchor>
      ))}
    </Group>
  );
}
