import { useEffect } from 'react';

import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import { Product } from '@/types';

export function useProductTracking(product: Product) {
  useEffect(() => {
    track('product_viewed', {
      product_id: product.id,
      product_name: product.name,
      category_id: product.category_id,
      price: product.selling_price,
    });
  }, [product.id, product.name, product.category_id, product.selling_price]);

  useScrollDepth('product');
}
