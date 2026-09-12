'use client';

import { Anchor, Box, Group, ScrollArea, Text } from '@mantine/core';
import Link from 'next/link';

import { Category } from '@/types';

interface CategoryChipsProps {
  categories: Category[];
  /** Phone strips swipe sideways instead of wrapping onto three rows. */
  scrollable?: boolean;
}

/** Chips navigate to each collection's own route rather than filtering in place. */
export function CategoryChips({ categories, scrollable }: CategoryChipsProps) {
  const chips = (
    <Group gap="xs" mb={scrollable ? 0 : 'xl'} wrap={scrollable ? 'nowrap' : 'wrap'}>
      <Box px={16} py={8} bg="brand.5" style={{ borderRadius: 999, flexShrink: 0 }}>
        <Text fz="sm" fw={600} c="white" style={{ whiteSpace: 'nowrap' }}>
          All Categories ({categories.length})
        </Text>
      </Box>
      {categories.map((category) => (
        <Anchor
          key={category.id}
          component={Link}
          href={`/c/${category.slug}`}
          underline="never"
          style={{ flexShrink: 0 }}
        >
          <Box
            px={16}
            py={8}
            bg="white"
            style={{
              borderRadius: 999,
              border: '1px solid var(--mantine-color-navy-2)',
              flexShrink: 0,
            }}
          >
            <Text fz="sm" fw={500} c="navy.8" style={{ whiteSpace: 'nowrap' }}>
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

  if (!scrollable) return chips;
  return (
    <ScrollArea type="never" mb="lg" offsetScrollbars={false}>
      {chips}
    </ScrollArea>
  );
}
