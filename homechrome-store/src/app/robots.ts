import type { MetadataRoute } from 'next';

import { IS_INDEXABLE, SITE_URL } from '@/lib/constants';

export default function robots(): MetadataRoute.Robots {
  // Non-production hosts (dev-store, preview, localhost) disallow everything so
  // search engines never index them as duplicate content against prod.
  if (!IS_INDEXABLE) {
    return { rules: [{ userAgent: '*', disallow: '/' }] };
  }

  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
  };
}
