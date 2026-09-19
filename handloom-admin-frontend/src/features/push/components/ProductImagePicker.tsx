import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

import { productsApi } from '@/features/products/api';
import { Input, Modal } from '@/shared/components/ui';

interface ProductImagePickerProps {
  opened: boolean;
  onClose: () => void;
  onSelect: (url: string) => void;
}

/** Every image already on a product, flattened — the primary one first. */
export function ProductImagePicker({ opened, onClose, onSelect }: ProductImagePickerProps) {
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['push-product-images', search],
    queryFn: () => productsApi.list({ search: search || undefined, limit: 40 }),
    enabled: opened,
  });

  const images = (data?.items ?? []).flatMap((product) =>
    (product.images ?? [])
      .slice()
      .sort((a, b) => Number(b.is_primary ?? false) - Number(a.is_primary ?? false))
      .map((image) => ({ url: image.url, name: product.name }))
  );

  return (
    <Modal isOpen={opened} onClose={onClose} title="Choose a product image">
      <Input
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Search products"
      />

      {isLoading ? (
        <p className="py-8 text-center text-sm text-gray-500">Loading…</p>
      ) : images.length === 0 ? (
        <p className="py-8 text-center text-sm text-gray-500">
          No product images match that search.
        </p>
      ) : (
        <div className="mt-4 grid max-h-96 grid-cols-3 gap-3 overflow-y-auto">
          {images.map((image) => (
            <button
              key={image.url}
              type="button"
              onClick={() => {
                onSelect(image.url);
                onClose();
              }}
              className="group overflow-hidden rounded-lg border border-gray-200 hover:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <img src={image.url} alt="" className="h-24 w-full object-cover" />
              <span className="block truncate px-2 py-1 text-left text-xs text-gray-600">
                {image.name}
              </span>
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
}
