'use client';

import { Text } from '@mantine/core';

interface EyebrowProps {
  children: string;
  /** Accent by default; `muted` for the quieter label above a field or list. */
  tone?: 'accent' | 'muted';
  size?: number;
}

/** The letterspaced caps label the design puts above almost every heading. */
export function Eyebrow({ children, tone = 'accent', size = 11 }: EyebrowProps) {
  return (
    <Text
      fz={size}
      fw={700}
      c={tone === 'accent' ? 'brand.5' : 'navy.5'}
      style={{ letterSpacing: '0.12em' }}
    >
      {children.toUpperCase()}
    </Text>
  );
}
