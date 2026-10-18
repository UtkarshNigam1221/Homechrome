'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { AspectRatio, Box, Card, Center, Stack, Text } from '@mantine/core';
import Link from 'next/link';

import { AssetImage } from '@/components/ui/asset-image';
import { stripMarkdown } from '@/lib/utils';
import { Category } from '@/types';

interface CategoryCardProps {
  category: Category;
  /** Stretch to the grid cell instead of a fixed 4:5 image, for the mixed-size homepage grid. */
  fill?: boolean;
}

export default function CategoryCard({ category, fill = false }: CategoryCardProps) {
  const image = category.image_url ? (
    <AssetImage
      src={category.image_url}
      alt={category.name}
      sizes={fill ? '(max-width: 61.99em) 100vw, 66vw' : '(max-width: 767px) 50vw, (max-width: 1199px) 33vw, 25vw'}
      width={fill ? 1200 : 640}
      height={fill ? 900 : 480}
      style={{ width: '100%', height: '100%', objectFit: 'cover' }}
    />
  ) : (
    <Center bg="brand.1" h="100%">
      <PhotoIcon width={48} height={48} color="var(--mantine-color-brand-7)" opacity={0.4} />
    </Center>
  );

  return (
    <Card
      component={Link}
      href={`/c/${category.slug}`}
      shadow="sm"
      padding={0}
      radius="lg"
      withBorder={false}
      h={fill ? '100%' : undefined}
      style={{ textDecoration: 'none', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}
    >
      <Card.Section pos="relative" style={fill ? { flex: 1, minHeight: 220 } : undefined}>
        {fill ? (
          <Box pos="absolute" inset={0} bg="navy.1">
            {image}
          </Box>
        ) : (
          <AspectRatio ratio={4 / 5} bg="navy.1">
            {image}
          </AspectRatio>
        )}

        {category.product_count > 0 && (
          <Box
            pos="absolute"
            top={12}
            right={12}
            px={10}
            py={5}
            bg="rgba(245,240,235,0.92)"
            style={{ borderRadius: 999, backdropFilter: 'blur(4px)' }}
          >
            <Text fz={11} fw={600} c="navy.8">
              {category.product_count} {category.product_count === 1 ? 'design' : 'designs'}
            </Text>
          </Box>
        )}
      </Card.Section>

      <Stack p="md" gap={2}>
        <Text fz="xl" fw={600} c="navy.9" lineClamp={1}>
          {category.name}
        </Text>
        <Text size="sm" c="dimmed" lineClamp={1}>
          {category.description
            ? stripMarkdown(category.description)
            : `${category.product_count} handcrafted pieces`}
        </Text>
      </Stack>
    </Card>
  );
}
