'use client';

import { ArrowRightIcon, PhotoIcon } from '@heroicons/react/24/outline';
import { Anchor, Box, Center, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import Link from 'next/link';

import { AssetImage } from '@/components/ui/asset-image';
import { Category } from '@/types';

import { stripMarkdown } from '@/lib/utils';

import { displayFont } from '@/app/fonts';

interface CategoryFeatureRowProps {
  category: Category;
  index: number;
}

/** Image and copy swap sides on alternate rows — only once side by side, so
 *  the stacked phone layout always leads with the image. */
export function CategoryFeatureRow({ category, index }: CategoryFeatureRowProps) {
  const flipped = index % 2 === 1;

  const visual = (
    <Box
      pos="relative"
      style={{
        borderRadius: 'var(--mantine-radius-lg)',
        overflow: 'hidden',
        aspectRatio: '4 / 3',
      }}
    >
      {category.image_url ? (
        <AssetImage
          src={category.image_url}
          alt={category.name}
          sizes="(max-width: 767px) 100vw, 50vw"
          width={900}
          height={675}
          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
        />
      ) : (
        <Center bg="brand.1" h="100%">
          <PhotoIcon width={48} height={48} color="var(--mantine-color-brand-5)" opacity={0.4} />
        </Center>
      )}

      {category.product_count > 0 && (
        <Box
          pos="absolute"
          top={14}
          left={14}
          px={10}
          py={5}
          bg="rgba(252,249,244,0.93)"
          style={{ borderRadius: 4, backdropFilter: 'blur(4px)' }}
        >
          <Text fz={11} fw={700} c="navy.8" style={{ letterSpacing: '0.06em' }}>
            {category.product_count} {category.product_count === 1 ? 'DESIGN' : 'DESIGNS'}
          </Text>
        </Box>
      )}
    </Box>
  );

  const copy = (
    <Stack gap="md" justify="center">
      <Text fz={11} fw={700} c="navy.5" style={{ letterSpacing: '0.14em' }}>
        CATEGORY {String(index + 1).padStart(2, '0')}
      </Text>

      <Title
        order={2}
        fz={{ base: '1.75rem', sm: '2.25rem' }}
        fw={600}
        lh={1.15}
        c="navy.9"
        style={{ fontFamily: displayFont.style.fontFamily }}
      >
        {category.name}
      </Title>

      {category.description && (
        <Text c="navy.6" lh={1.65} lineClamp={5}>
          {stripMarkdown(category.description)}
        </Text>
      )}

      <Anchor
        component={Link}
        href={`/c/${category.slug}`}
        underline="never"
        c="brand.5"
        fw={600}
        w="fit-content"
        mt={4}
      >
        <Group gap={8} wrap="nowrap" align="center">
          Explore {category.name}
          <ArrowRightIcon width={16} height={16} />
        </Group>
      </Anchor>
    </Stack>
  );

  return (
    <SimpleGrid
      data-feature-flip={flipped || undefined}
      cols={{ base: 1, md: 2 }}
      spacing={{ base: 'lg', md: 48 }}
      p={{ base: 'md', md: 'xl' }}
      bg="white"
      style={{ borderRadius: 'var(--mantine-radius-lg)', alignItems: 'center' }}
    >
      {visual}
      {copy}
    </SimpleGrid>
  );
}
