'use client';

import { ArrowUpRightIcon, PhotoIcon } from '@heroicons/react/24/outline';
import { AspectRatio, Box, Card, Center, Group, Stack, Text } from '@mantine/core';
import { AssetImage } from '@/components/ui/asset-image';
import Link from 'next/link';

import { Category } from '@/types';

import { displayFont } from '@/app/fonts';

// Some catalogue descriptions carry markdown from the admin editor; the card
// shows one plain line, so strip the markers rather than print them.
const stripMarkdown = (text: string) => text.replace(/[*_`#]/g, '').trim();

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
        <AspectRatio ratio={4 / 5} bg="navy.1">
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

        {category.product_count > 0 && (
          <Box
            pos="absolute"
            top={12}
            right={12}
            px={10}
            py={5}
            bg="rgba(252,249,244,0.92)"
            style={{ borderRadius: 999, backdropFilter: 'blur(4px)' }}
          >
            <Text fz={11} fw={700} c="navy.8">
              {category.product_count} {category.product_count === 1 ? 'Design' : 'Designs'}
            </Text>
          </Box>
        )}
      </Card.Section>

      <Group p="md" gap="sm" wrap="nowrap" justify="space-between" align="center">
        <Stack gap={2} style={{ minWidth: 0 }}>
          <Text
            fz="xl"
            fw={600}
            c="navy.9"
            lineClamp={1}
            style={{ fontFamily: displayFont.style.fontFamily }}
          >
            {category.name}
          </Text>
          <Text size="sm" c="dimmed" lineClamp={1}>
            {category.description
              ? stripMarkdown(category.description)
              : `${category.product_count} handcrafted pieces`}
          </Text>
        </Stack>
        <Box
          w={34}
          h={34}
          c="navy.8"
          style={{
            border: '1px solid var(--mantine-color-navy-3)',
            borderRadius: 999,
            display: 'grid',
            placeItems: 'center',
            flexShrink: 0,
          }}
        >
          <ArrowUpRightIcon width={16} height={16} />
        </Box>
      </Group>
    </Card>
  );
}
