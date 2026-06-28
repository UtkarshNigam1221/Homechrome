'use client';

import dynamic from 'next/dynamic';

import { Category } from '@/types';

const SpotlightSearch = dynamic(
  () => import('@/components/search/SpotlightSearch').then((m) => m.SpotlightSearch),
  { ssr: false },
);

interface SpotlightSearchLoaderProps {
  categories: Category[];
}

export function SpotlightSearchLoader({ categories }: SpotlightSearchLoaderProps) {
  return <SpotlightSearch categories={categories} />;
}
