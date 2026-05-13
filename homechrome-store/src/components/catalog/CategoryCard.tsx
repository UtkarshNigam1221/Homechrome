import { PhotoIcon } from '@heroicons/react/24/outline';
import Image from 'next/image';
import Link from 'next/link';

import { Category } from '@/types';

interface CategoryCardProps {
  category: Category;
}

export default function CategoryCard({ category }: CategoryCardProps) {
  return (
    <Link
      href={`/c/${category.slug}`}
      className="group block overflow-hidden rounded-xl bg-white shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="relative aspect-[4/3] overflow-hidden bg-gray-100">
        {category.image_url ? (
          <Image
            src={category.image_url}
            alt={category.name}
            fill
            sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 25vw"
            className="object-cover transition-transform duration-300 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full items-center justify-center bg-primary-light/30">
            <PhotoIcon className="h-12 w-12 text-primary/40" />
          </div>
        )}
      </div>
      <div className="p-4">
        <h3 className="text-base font-semibold text-foreground group-hover:text-primary">
          {category.name}
        </h3>
        <p className="mt-1 text-sm text-muted-foreground">
          {category.product_count} {category.product_count === 1 ? 'product' : 'products'}
        </p>
      </div>
    </Link>
  );
}
