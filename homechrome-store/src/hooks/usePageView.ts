'use client';

import { useEffect } from 'react';

import { track } from '@/lib/analytics';

export function usePageView(properties: Record<string, unknown> = {}) {
  useEffect(() => {
    track('page_view', properties);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
}
