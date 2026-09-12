'use client';

import { Box, Stack, Text, Title } from '@mantine/core';

import { displayFont } from '@/app/fonts';

interface PageHeaderProps {
  /** Small letterspaced label above the title. */
  eyebrow?: string;
  title: string;
  description?: string;
  children?: React.ReactNode;
  className?: string;
}

export function PageHeader({
  eyebrow,
  title,
  description,
  children,
  className,
}: PageHeaderProps) {
  return (
    <Box mb="xl" className={className}>
      <Stack gap={6}>
        {eyebrow && (
          <Text fz={12} fw={700} c="brand.5" style={{ letterSpacing: '0.14em' }}>
            {eyebrow.toUpperCase()}
          </Text>
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
        {description && (
          <Text size="sm" c="navy.6" maw={620} lh={1.6}>
            {description}
          </Text>
        )}
      </Stack>
      {children}
    </Box>
  );
}
