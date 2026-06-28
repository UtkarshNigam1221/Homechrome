'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { AspectRatio, Card, Center, Stack, Text } from '@mantine/core';
import { AssetImage } from '@/components/ui/asset-image';
import Link from 'next/link';

import { Category } from '@/types';

interface CategoryCardProps {
  category: Category;
}

export default function CategoryCard({ category }: CategoryCardProps) {
  return (
    <Card
      component={Link}
      href={`/c/${category.slug}`}
      shadow="sm"
      padding={0}
      radius="lg"
      withBorder={false}
      style={{ textDecoration: 'none', overflow: 'hidden' }}
    >
      <Card.Section pos="relative">
        <AspectRatio ratio={4 / 3} bg="gray.1">
          {category.image_url ? (
            <AssetImage
              src={category.image_url}
              alt={category.name}
              sizes="(max-width: 767px) 50vw, (max-width: 1199px) 33vw, 25vw"
              width={640}
              height={480}
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            />
          ) : (
            <Center bg="brand.1" h="100%">
              <PhotoIcon width={48} height={48} color="var(--mantine-color-brand-5)" opacity={0.4} />
            </Center>
          )}
        </AspectRatio>
      </Card.Section>

      <Stack p="md" gap={4}>
        <Text fw={600} c="navy.7">
          {category.name}
        </Text>
        <Text size="sm" c="dimmed">
          {category.product_count} {category.product_count === 1 ? 'product' : 'products'}
        </Text>
      </Stack>
    </Card>
  );
}
