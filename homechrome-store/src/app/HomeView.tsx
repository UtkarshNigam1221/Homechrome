'use client';

import {
  Anchor,
  Box,
  Button,
  Container,
  Group,
  SimpleGrid,
  Stack,
  Text,
  ThemeIcon,
  Title,
} from '@mantine/core';
import Link from 'next/link';

import CategoryCard from '@/components/catalog/CategoryCard';
import ProductCard from '@/components/catalog/ProductCard';
import { Category, Product } from '@/types';

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
            <Title order={1} ta="center" c="white" size="3rem" fw={700}>
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

      <Box component="section" bg="white" py="xl">
        <Container size="xl">
          <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="xl">
            <Feature
              title="Handcrafted"
              description="Each piece is handwoven by skilled weavers preserving centuries-old techniques."
              iconPath="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z"
            />
            <Feature
              title="Free Shipping"
              description="Free delivery on all orders above Rs. 999 across India."
              iconPath="M8.25 18.75a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 0 1-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 0 0-3.213-9.193 2.056 2.056 0 0 0-1.58-.86H14.25M16.5 18.75h-2.25m0-11.177v-.958c0-.568-.422-1.048-.987-1.106a48.554 48.554 0 0 0-10.026 0 1.106 1.106 0 0 0-.987 1.106v7.635m12-6.677v6.677m0 4.5v-4.5m0 0h-12"
            />
            <Feature
              title="Quality Assured"
              description="Authenticity guaranteed with GI-tagged handloom products."
              iconPath="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"
            />
          </SimpleGrid>
        </Container>
      </Box>

      {categories.length > 0 && (
        <Box component="section" py="xl">
          <Container size="xl">
            <Group justify="space-between" align="center">
              <Title order={2} size="h3">Shop by Category</Title>
              <Anchor component={Link} href="/categories" c="brand" fw={500} size="sm">
                View All
              </Anchor>
            </Group>
            <SimpleGrid cols={{ base: 2, sm: 3, lg: 4 }} spacing="md" mt="lg">
              {categories.slice(0, 8).map((category) => (
                <CategoryCard key={category.id} category={category} />
              ))}
            </SimpleGrid>
          </Container>
        </Box>
      )}

      {products.length > 0 && (
        <Box component="section" bg="white" py="xl">
          <Container size="xl">
            <Group justify="space-between" align="center">
              <Title order={2} size="h3">New Arrivals</Title>
              <Anchor component={Link} href="/products" c="brand" fw={500} size="sm">
                View All
              </Anchor>
            </Group>
            <SimpleGrid cols={{ base: 2, sm: 3, lg: 4 }} spacing="md" mt="lg">
              {products.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </SimpleGrid>
          </Container>
        </Box>
      )}
    </>
  );
}

interface FeatureProps {
  title: string;
  description: string;
  iconPath: string;
}

function Feature({ title, description, iconPath }: FeatureProps) {
  return (
    <Stack align="center" gap="xs" ta="center">
      <ThemeIcon size={48} radius="xl" variant="light" color="brand">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
          width={24}
          height={24}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d={iconPath} />
        </svg>
      </ThemeIcon>
      <Title order={3} size="md" mt="xs">{title}</Title>
      <Text size="sm" c="dimmed">{description}</Text>
    </Stack>
  );
}
