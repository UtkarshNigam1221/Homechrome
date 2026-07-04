'use client';

import {
  Anchor,
  Box,
  Button,
  Container,
  Divider,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import { Carousel } from '@mantine/carousel';
import Link from 'next/link';

import CategoryCard from '@/components/catalog/CategoryCard';
import ProductCard from '@/components/catalog/ProductCard';
import { Category, Product } from '@/types';

import { displayFont } from './fonts';
import HomePageTracker from './HomePageTracker';

interface HomeViewProps {
  categories: Category[];
  products: Product[];
}

export default function HomeView({ categories, products }: HomeViewProps) {
  return (
    <>
      <HomePageTracker />

      <Box component="section" bg="navy.7">
        <Container size="xl" py={{ base: 96, sm: 128 }}>
          <Stack align="center" gap="xl">
            <Title
              order={1}
              ta="center"
              c="white"
              fz={{ base: '2.25rem', sm: '2.75rem', md: '3.25rem' }}
              fw={700}
              lh={1.1}
              style={{ fontFamily: displayFont.style.fontFamily }}
            >
              Handwoven with <Text span inherit c="brand">tradition</Text>
            </Title>
            <Text size="lg" ta="center" c="white" opacity={0.8} maw={640}>
              Discover premium handloom textiles crafted with tradition across India.
              Each piece tells a story of heritage and craftsmanship.
            </Text>
            <Group justify="center" gap="md" mt="lg">
              <Button component={Link} href="/products" size="lg" color="brand">
                Shop Now
              </Button>
              <Button
                component={Link}
                href="/categories"
                size="lg"
                variant="outline"
                c="white"
                styles={{ root: { borderColor: 'rgba(255,255,255,0.3)' } }}
              >
                Browse Categories
              </Button>
            </Group>
          </Stack>
        </Container>
      </Box>

      {categories.length > 0 && (
        <Box component="section" py="xl" aria-labelledby="home-categories-heading">
          <Container size="xl">
            <Group justify="space-between" align="end" mb="lg">
              <Title id="home-categories-heading" order={2} size="h3">
                Shop by Category
              </Title>
              <Anchor component={Link} href="/categories" c="brand" fw={500} size="sm">
                View All
              </Anchor>
            </Group>
            <Divider mb="lg" />
            <SimpleGrid cols={{ base: 2, sm: 3, lg: 4 }} spacing="md">
              {categories.slice(0, 8).map((category) => (
                <CategoryCard key={category.id} category={category} />
              ))}
            </SimpleGrid>
          </Container>
        </Box>
      )}

      {products.length > 0 && (
        <Box component="section" bg="white" py="xl" aria-labelledby="home-new-arrivals-heading">
          <Container size="xl">
            <Group justify="space-between" align="end" mb="lg">
              <Title id="home-new-arrivals-heading" order={2} size="h3">
                New Arrivals
              </Title>
              <Anchor component={Link} href="/products" c="brand" fw={500} size="sm">
                View All
              </Anchor>
            </Group>
            <Divider mb="lg" />
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
          </Container>
        </Box>
      )}
    </>
  );
}

