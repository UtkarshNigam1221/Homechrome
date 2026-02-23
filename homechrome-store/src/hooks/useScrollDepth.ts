'use client';

import { useEffect } from 'react';

import { track } from '@/lib/analytics';

export function useScrollDepth(pageType: string) {
  useEffect(() => {
    let maxDepth = 0;

    const handler = () => {
      const scrollHeight =
        document.documentElement.scrollHeight - window.innerHeight;
      if (scrollHeight > 0) {
        const depth = Math.round((window.scrollY / scrollHeight) * 100);
        maxDepth = Math.max(maxDepth, depth);
      }
    };

    window.addEventListener('scroll', handler, { passive: true });

    return () => {
      window.removeEventListener('scroll', handler);
      if (maxDepth > 0) {
        track('scroll_depth', {
          page_type: pageType,
          max_depth_percent: maxDepth,
        });
      }
    };
  }, [pageType]);
}
