'use client';

import {
  ArrowRightIcon,
  ShieldCheckIcon,
  SparklesIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';
import { Carousel } from '@mantine/carousel';
import {
  Box,
  Button,
  Card,
  Container,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import Link from 'next/link';

import { AssetImage } from '@/components/ui/asset-image';
import { AssuranceRow } from '@/components/ui/assurance-row';
import { SectionHeading } from '@/components/ui/section-heading';
import CategoryCard from '@/components/catalog/CategoryCard';
import ProductCard from '@/components/catalog/ProductCard';
import { Category, Product } from '@/types';


import { displayFont } from './fonts';
import HomePageTracker from './HomePageTracker';

function HeroVisual({ product }: { product?: Product }) {
  const image = product?.images?.find((i) => i.is_primary) ?? product?.images?.[0];
  if (!image) return null;

  // The provenance card reads from the product's own origin/craft fields, so it
  // never states a cluster the catalogue does not claim.
  const provenance = [product?.craft_type, product?.origin].filter(Boolean).join(' · ');

  return (
    <Box pos="relative">
      <Box
        style={{
          borderRadius: 'var(--mantine-radius-lg)',
          overflow: 'hidden',
          aspectRatio: '4 / 3.4',
        }}
      >
        <AssetImage
          src={image.url}
          alt={image.alt_text || product?.name || 'Handloom textile'}
          sizes="(max-width: 767px) 100vw, 50vw"
          width={900}
          height={760}
          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
        />
      </Box>

      {product && (
        <Card
          pos="absolute"
          bottom={16}
          left={16}
          right={16}
          radius="md"
          p="sm"
          bg="rgba(252,249,244,0.94)"
          style={{ backdropFilter: 'blur(6px)' }}
        >
          <Group gap="sm" wrap="nowrap" justify="space-between">
            <Group gap="sm" wrap="nowrap">
              <Box
                w={36}
                h={36}
                bg="brand.1"
                style={{
                  borderRadius: 'var(--mantine-radius-sm)',
                  display: 'grid',
                  placeItems: 'center',
                  flexShrink: 0,
                }}
              >
                <SparklesIcon width={18} height={18} color="var(--mantine-color-brand-6)" />
              </Box>
              <Stack gap={0} style={{ minWidth: 0 }}>
                <Text
                  fz="sm"
                  fw={600}
                  c="navy.9"
                  truncate
                  style={{ fontFamily: displayFont.style.fontFamily }}
                >
                  {product.name}
                </Text>
                {provenance && (
                  <Text fz="xs" c="navy.6" truncate>
                    {provenance}
                  </Text>
                )}
              </Stack>
            </Group>
          </Group>
        </Card>
      )}
    </Box>
  );
}

interface HomeViewProps {
  categories: Category[];
  products: Product[];
}

export default function HomeView({ categories, products }: HomeViewProps) {
  return (
    <>
      <HomePageTracker />

      <Box component="section" bg="navy.0">
        <Container size="xl" py={{ base: 48, sm: 72 }}>
          <SimpleGrid
            cols={{ base: 1, md: 2 }}
            spacing={{ base: 40, md: 64 }}
            style={{ alignItems: 'center' }}
          >
            <Box>
              <Stack gap="lg">
                <Group
                  gap={8}
                  w="fit-content"
                  px={12}
                  py={6}
                  bg="brand.1"
                  style={{ borderRadius: 999 }}
                >
                  <SparklesIcon width={14} height={14} color="var(--mantine-color-brand-6)" />
                  <Text fz={11} fw={700} c="brand.6" style={{ letterSpacing: '0.1em' }}>
                    HANDLOOM TEXTILES FROM INDIA
                  </Text>
                </Group>

                <Title
                  order={1}
                  fz={{ base: '2.25rem', sm: '3rem', md: '3.4rem' }}
                  fw={600}
                  lh={1.08}
                  c="navy.9"
                >
                  Handwoven with{' '}
                  <Text span inherit c="brand.5" fs="italic">
                    tradition,
                  </Text>{' '}
                  crafted for modern sanctuaries.
                </Title>

                <Text size="lg" c="navy.6" maw={520} lh={1.6}>
                  Premium handloom textiles from across India — bedsheets, dohars and cushions
                  woven by hand, printed by hand, and made for everyday living.
                </Text>

                <Group gap="md" mt="xs">
                  <Button
                    component={Link}
                    href="/products"
                    size="md"
                    color="brand"
                    radius="sm"
                    rightSection={<ArrowRightIcon width={16} height={16} />}
                  >
                    Explore Collections
                  </Button>
                  <Button
                    component={Link}
                    href="/categories"
                    size="md"
                    variant="white"
                    c="navy.9"
                    radius="sm"
                  >
                    Browse Categories
                  </Button>
                </Group>

                <Group gap="xl" mt="sm">
                  <Group gap={8} wrap="nowrap">
                    <TruckIcon width={17} height={17} color="var(--mantine-color-navy-5)" />
                    <Text fz="sm" c="navy.6">Free shipping, always</Text>
                  </Group>
                  <Group gap={8} wrap="nowrap">
                    <ShieldCheckIcon width={17} height={17} color="var(--mantine-color-navy-5)" />
                    <Text fz="sm" c="navy.6">Damage replacement</Text>
                  </Group>
                </Group>
              </Stack>
            </Box>

            <HeroVisual product={products[0]} />
          </SimpleGrid>
        </Container>
      </Box>

      <Box component="section" bg="white" py={{ base: 32, sm: 40 }}>
        <Container size="xl">
          <AssuranceRow withTile />
        </Container>
      </Box>

      {categories.length > 0 && (
        <Box component="section" py={{ base: 48, sm: 72 }} aria-labelledby="home-categories-heading">
          <Container size="xl">
            <SectionHeading
              id="home-categories-heading"
              eyebrow="Handcrafted disciplines"
              title="Curated Handloom Living"
              linkHref="/categories"
              linkLabel="View All Categories"
            />
            <SimpleGrid cols={{ base: 1, xs: 2, lg: 3 }} spacing="lg">
              {categories.slice(0, 6).map((category) => (
                <CategoryCard key={category.id} category={category} />
              ))}
            </SimpleGrid>
          </Container>
        </Box>
      )}

      {products.length > 0 && (
        <Box
          component="section"
          bg="white"
          py={{ base: 48, sm: 72 }}
          aria-labelledby="home-new-arrivals-heading"
        >
          <Container size="xl">
            <SectionHeading
              id="home-new-arrivals-heading"
              eyebrow="Seasonal wefts"
              title="New Arrivals"
              linkHref="/products"
              linkLabel="View All"
            />
            <Carousel
              slideSize={{ base: '80%', xs: '50%', sm: '33.333%', lg: '25%' }}
              slideGap="md"
              emblaOptions={{ align: 'start', dragFree: true }}
              withControls
              controlSize={36}
              aria-label="New arrivals"
            >
              {products.map((product) => (
                <Carousel.Slide key={product.id}>
                  <ProductCard product={product} />
                </Carousel.Slide>
              ))}
            </Carousel>

            <Group justify="center" mt="xl">
              <Button
                component={Link}
                href="/products"
                size="md"
                variant="default"
                radius="sm"
                rightSection={<ArrowRightIcon width={16} height={16} />}
              >
                Explore All Handcrafted Pieces
              </Button>
            </Group>
          </Container>
        </Box>
      )}
    </>
  );
}
