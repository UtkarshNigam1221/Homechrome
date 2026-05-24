'use client';

import { Anchor, Breadcrumbs, Text } from '@mantine/core';
import Link from 'next/link';

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
  className?: string;
}

export function Breadcrumb({ items, className }: BreadcrumbProps) {
  return (
    <Breadcrumbs
      mb="md"
      className={className}
      aria-label="Breadcrumb"
      separator="/"
      separatorMargin="xs"
    >
      {items.map((item, idx) => {
        const isLast = idx === items.length - 1;
        if (item.href && !isLast) {
          return (
            <Anchor
              key={`${item.label}-${idx}`}
              component={Link}
              href={item.href}
              size="sm"
              c="dimmed"
              underline="never"
            >
              {item.label}
            </Anchor>
          );
        }
        return (
          <Text key={`${item.label}-${idx}`} size="sm" c="navy.7" truncate>
            {item.label}
          </Text>
        );
      })}
    </Breadcrumbs>
  );
}
