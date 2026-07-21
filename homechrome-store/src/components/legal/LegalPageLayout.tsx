import { Container, Stack, Text, Title } from '@mantine/core';

import { LEGAL_LAST_UPDATED } from '@/lib/constants';

// Shared shell for the static legal pages: constrained prose width, one h1,
// last-updated stamp, consistent section typography. Server component — same
// no-'use client' Mantine pattern as app/contact/page.tsx.
export function LegalPageLayout({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Container size="sm" py="xl">
      <Stack gap="xl">
        <div>
          <Title order={1} c="navy.7">
            {title}
          </Title>
          <Text size="sm" c="dimmed" mt={4}>
            Last updated: {LEGAL_LAST_UPDATED}
          </Text>
        </div>
        {children}
      </Stack>
    </Container>
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
      <Title order={2} size="h4" c="navy.7" mb="xs">
        {heading}
      </Title>
      <Stack gap="sm">{children}</Stack>
    </section>
  );
}
