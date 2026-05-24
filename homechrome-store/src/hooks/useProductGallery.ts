import { useMemo, useState } from 'react';

import { Product, ProductImage } from '@/types';

export type GalleryItem =
  | { kind: 'video'; url: string; poster?: string }
  | { kind: 'image'; image: ProductImage };

export function useProductGallery(product: Product) {
  const items = useMemo<GalleryItem[]>(() => {
    const sortedImages = [...(product.images || [])].sort((a, b) => {
      if (a.is_primary && !b.is_primary) return -1;
      if (!a.is_primary && b.is_primary) return 1;
      return a.sort_order - b.sort_order;
    });

    return [
      ...(product.video_url
        ? [{ kind: 'video' as const, url: product.video_url, poster: product.video_poster_url }]
        : []),
      ...sortedImages.map((image) => ({ kind: 'image' as const, image })),
    ];
  }, [product.images, product.video_url, product.video_poster_url]);

  const [selectedIndex, setSelectedIndex] = useState(0);
  const selectedItem: GalleryItem | null = items[selectedIndex] ?? null;

  return { items, selectedIndex, selectedItem, setSelectedIndex };
}
