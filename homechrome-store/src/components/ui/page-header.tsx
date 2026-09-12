'use client';

import { Box, Stack, Text, Title } from '@mantine/core';

import { Eyebrow } from '@/components/ui/eyebrow';


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
        {eyebrow && <Eyebrow size={12}>{eyebrow}</Eyebrow>}
        <Title
          order={1}
          fz={{ base: '1.75rem', sm: '2.25rem' }}
          fw={600}
          lh={1.22}
          c="navy.9"
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
