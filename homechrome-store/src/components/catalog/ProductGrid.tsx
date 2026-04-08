import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';

import ProductCard from '@/components/catalog/ProductCard';
import { Product } from '@/types';

interface ProductGridProps {
  products: Product[];
}

export default function ProductGrid({ products }: ProductGridProps) {
  if (products.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <MagnifyingGlassIcon strokeWidth={1} className="h-16 w-16 text-muted/50" />
        <h3 className="mt-4 text-lg font-medium text-foreground">No products found</h3>
        <p className="mt-2 text-sm text-muted">
          Try adjusting your filters or browse our categories.
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {products.map((product) => (
        <ProductCard key={product.id} product={product} />
      ))}
    </div>
  );
}
