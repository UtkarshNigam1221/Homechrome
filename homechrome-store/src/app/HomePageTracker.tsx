'use client';

import { usePageView } from '@/hooks/usePageView';
import { useScrollDepth } from '@/hooks/useScrollDepth';

export default function HomePageTracker() {
  usePageView({ page_type: 'home' });
  useScrollDepth('home');
  return null;
}
