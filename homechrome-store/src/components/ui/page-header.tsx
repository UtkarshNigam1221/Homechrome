'use client';

import { Box, Stack, Text, Title } from '@mantine/core';

interface PageHeaderProps {
  title: string;
  description?: string;
  children?: React.ReactNode;
  className?: string;
}

export function PageHeader({ title, description, children, className }: PageHeaderProps) {
  return (
    <Box mb="xl" className={className}>
      <Stack gap={4}>
        <Title order={1} size="h2">{title}</Title>
        {description && (
          <Text size="sm" c="dimmed">{description}</Text>
        )}
      </Stack>
      {children}
    </Box>
  );
}
