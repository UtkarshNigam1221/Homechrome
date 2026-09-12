'use client';

import { Box, Card, Group, SimpleGrid, Stack, Text } from '@mantine/core';

import { ASSURANCES } from '@/lib/assurances';

interface AssuranceRowProps {
  /** `cards` sits each on its own surface; `plain` lays them on the section. */
  variant?: 'plain' | 'cards';
  /** `short` is the one-line form for tight rows. */
  copy?: 'body' | 'short';
  columns?: number;
  /** Tile-style icon chip, as the product page uses. */
  withTile?: boolean;
}

export function AssuranceRow({
  variant = 'plain',
  copy = 'body',
  columns = 4,
  withTile = false,
}: AssuranceRowProps) {
  const items = ASSURANCES.map(({ key, icon: Icon, title, body, short }) => {
    const inner = (
      <Group gap="sm" wrap="nowrap" align="flex-start">
        {withTile ? (
          <Box
            w={40}
            h={40}
            bg="brand.1"
            style={{
              borderRadius: 'var(--mantine-radius-md)',
              display: 'grid',
              placeItems: 'center',
              flexShrink: 0,
            }}
          >
            <Icon width={19} height={19} color="var(--mantine-color-brand-6)" />
          </Box>
        ) : (
          <Icon
            width={20}
            height={20}
            color="var(--mantine-color-brand-6)"
            style={{ flexShrink: 0, marginTop: 2 }}
          />
        )}
        <Stack gap={2}>
          <Text fz="sm" fw={600} c="navy.9" lh={1.3}>
            {title}
          </Text>
          <Text fz="xs" c="navy.6" lh={1.5}>
            {copy === 'short' ? short : body}
          </Text>
        </Stack>
      </Group>
    );

    return variant === 'cards' ? (
      <Card key={key} shadow="sm" radius="lg" padding="md" withBorder={false}>
        {inner}
      </Card>
    ) : (
      <Box key={key}>{inner}</Box>
    );
  });

  return (
    <SimpleGrid cols={{ base: 1, xs: 2, lg: columns }} spacing="md">
      {items}
    </SimpleGrid>
  );
}
