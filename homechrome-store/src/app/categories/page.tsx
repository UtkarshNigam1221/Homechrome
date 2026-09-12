import { Box, Center, Container, Stack, Text } from '@mantine/core';
import type { Metadata } from 'next';

import { CategoryChips } from '@/components/catalog/CategoryChips';
import { CategoryFeatureRow } from '@/components/catalog/CategoryFeatureRow';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { PageHeader } from '@/components/ui/page-header';
import { WhatsAppButton } from '@/components/ui/whatsapp-button';
import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category } from '@/types';

export const metadata: Metadata = {
  alternates: { canonical: '/categories' },
  title: 'All Categories | Homechrome',
  description: 'Browse our collection of handloom textile categories.',
};

async function getCategories(): Promise<Category[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.CATEGORIES}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export default async function CategoriesPage() {
  const categories = await getCategories();
  const total = categories.reduce((sum, c) => sum + (c.product_count || 0), 0);

  return (
    <Container size="xl" py="xl">
      <Breadcrumb items={[{ label: 'Home', href: '/' }, { label: 'Categories' }]} />

      <PageHeader
        eyebrow="The archive"
        title="Curated Handloom Categories"
        description={`${total} handcrafted pieces across ${categories.length} collections — bedsheets, dohars and cushions woven and printed across India.`}
      />

      {categories.length > 0 ? (
        <>
          <CategoryChips categories={categories} />

          <Stack gap="lg">
            {categories.map((category, index) => (
              <CategoryFeatureRow key={category.id} category={category} index={index} />
            ))}
          </Stack>

          <Box mt={48} p={{ base: 'lg', md: 48 }} bg="white" style={{ borderRadius: 'var(--mantine-radius-lg)' }}>
            <Stack align="center" gap="md">
              <PageHeader title="Unsure about dimensions or drape?" />
              <Text c="navy.6" ta="center" maw={520} mt={-24}>
                Send us the bed size or room you are buying for and we will point you to the right
                weave.
              </Text>
              <WhatsAppButton
                message="Hi, I need help choosing a size"
                label="Ask us on WhatsApp"
                fullWidth={false}
              />
            </Stack>
          </Box>
        </>
      ) : (
        <Center py="xl">
          <Text size="lg" c="dimmed">No categories available at the moment.</Text>
        </Center>
      )}
    </Container>
  );
}
