import { Box, Card, Container, Flex, Group, Stack, Text, Title } from '@mantine/core';

import { PolicyNav } from '@/components/legal/PolicyNav';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { LEGAL_LAST_UPDATED } from '@/lib/constants';

import { displayFont } from '@/app/fonts';

// Shared shell for the static legal pages: chapter rail, one h1, last-updated
// stamp, consistent section typography. The prose itself is passed in and left
// exactly as written — these pages are binding, so presentation only.
export function LegalPageLayout({
  title,
  eyebrow,
  intro,
  children,
}: {
  title: string;
  eyebrow?: string;
  intro?: string;
  children: React.ReactNode;
}) {
  return (
    <>
      <Box bg="navy.1" py={{ base: 28, sm: 40 }}>
        <Container size="xl">
          <Breadcrumb
            items={[
              { label: 'Home', href: '/' },
              { label: 'Policies', href: '/policies' },
              { label: title },
            ]}
          />
          <Stack gap="sm" maw={720}>
            {eyebrow && (
              <Group gap={8} w="fit-content" px={12} py={6} bg="brand.1" style={{ borderRadius: 999 }}>
                <Text fz={11} fw={700} c="brand.6" style={{ letterSpacing: '0.12em' }}>
                  {eyebrow.toUpperCase()}
                </Text>
              </Group>
            )}
            <Title
              order={1}
              fz={{ base: '1.875rem', sm: '2.5rem' }}
              fw={600}
              lh={1.15}
              c="navy.9"
              style={{ fontFamily: displayFont.style.fontFamily }}
            >
              {title}
            </Title>
            {intro && (
              <Text c="navy.6" lh={1.65}>
                {intro}
              </Text>
            )}
            <Text size="sm" c="navy.5">
              Last updated: {LEGAL_LAST_UPDATED}
            </Text>
          </Stack>
        </Container>
      </Box>

      <Container size="xl" py="xl">
        <Flex gap="lg" align="flex-start" direction={{ base: 'column', lg: 'row' }}>
          <Box w={{ base: '100%', lg: 280 }} flex="none">
            <PolicyNav />
          </Box>

          <Card shadow="sm" radius="lg" padding="xl" withBorder={false} flex={1} w="100%">
            <Stack gap="xl">{children}</Stack>
          </Card>
        </Flex>
      </Container>
    </>
  );
}

export function LegalSection({
  heading,
  children,
}: {
  heading: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <Title
        order={2}
        fz={{ base: '1.25rem', sm: '1.5rem' }}
        fw={600}
        c="navy.9"
        mb="sm"
        style={{ fontFamily: displayFont.style.fontFamily }}
      >
        {heading}
      </Title>
      <Stack gap="sm">{children}</Stack>
    </section>
  );
}
