'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useEffect, useState } from 'react';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { PlayIcon } from '@heroicons/react/24/solid';
import { toast } from 'sonner';

import Button from '@/components/ui/button';
import { Container } from '@/components/ui/container';
import { QuantityStepper } from '@/components/ui/quantity-stepper';
import { useCart } from '@/hooks/useCart';
import { useScrollDepth } from '@/hooks/useScrollDepth';
import { track } from '@/lib/analytics';
import { calculateDiscountPercent, formatPrice } from '@/lib/utils';
import { useCartStore } from '@/stores/cart';
import { Product, ProductImage } from '@/types';

interface ProductDetailViewProps {
  product: Product;
}

export default function ProductDetailView({ product }: ProductDetailViewProps) {
  const sortedImages = [...(product.images || [])].sort((a, b) => {
    if (a.is_primary && !b.is_primary) return -1;
    if (!a.is_primary && b.is_primary) return 1;
    return a.sort_order - b.sort_order;
  });

  type GalleryItem =
    | { kind: 'video'; url: string; poster?: string }
    | { kind: 'image'; image: ProductImage };

  const galleryItems: GalleryItem[] = [
    ...(product.video_url
      ? [{ kind: 'video' as const, url: product.video_url, poster: product.video_poster_url }]
      : []),
    ...sortedImages.map((image) => ({ kind: 'image' as const, image })),
  ];

  const [selectedIndex, setSelectedIndex] = useState(0);
  const selectedItem: GalleryItem | null = galleryItems[selectedIndex] ?? null;
  const [quantity, setQuantity] = useState(1);
  const [loading, setLoading] = useState(false);
  const { addItem, updateQuantity, removeItem } = useCart();
  const cartQty = useCartStore((s) => s.getQuantity(product.id));

  useEffect(() => {
    track('product_viewed', {
      product_id: product.id,
      product_name: product.name,
      category_id: product.category_id,
      price: product.selling_price,
    });
  }, [product.id, product.name, product.category_id, product.selling_price]);

  useScrollDepth('product');

  const hasDiscount = product.mrp > product.selling_price;
  const discountPercent = calculateDiscountPercent(product.mrp, product.selling_price);

  const handleAddToCart = async () => {
    setLoading(true);
    try {
      await addItem(product.id, quantity);
      track('add_to_cart', {
        product_id: product.id,
        product_name: product.name,
        category_id: product.category_id,
        price: product.selling_price,
        quantity,
      });
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  const handleCartIncrement = async () => {
    setLoading(true);
    try {
      await updateQuantity(product.id, cartQty + 1);
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  const handleCartDecrement = async () => {
    setLoading(true);
    try {
      if (cartQty <= 1) {
        await removeItem(product.id);
      } else {
        await updateQuantity(product.id, cartQty - 1);
      }
    } catch {
      toast.error('Failed to update cart');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container className="py-10">
      {/* Breadcrumb */}
      <nav className="mb-6 text-sm text-muted-foreground">
        <ol className="flex items-center gap-2">
          <li>
            <Link href="/" className="hover:text-primary">
              Home
            </Link>
          </li>
          <li>/</li>
          <li>
            <Link href="/products" className="hover:text-primary">
              Products
            </Link>
          </li>
          <li>/</li>
          <li className="truncate text-foreground">{product.name}</li>
        </ol>
      </nav>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-2">
        {/* Image Gallery */}
        <div>
          {/* Main Media */}
          <div className="relative aspect-square overflow-hidden rounded-xl bg-gray-100">
            {selectedItem?.kind === 'video' ? (
              <video
                key={selectedItem.url}
                src={selectedItem.url}
                controls
                playsInline
                preload="metadata"
                poster={selectedItem.poster}
                className="h-full w-full object-contain"
              />
            ) : selectedItem?.kind === 'image' ? (
              <Image
                src={selectedItem.image.url}
                alt={selectedItem.image.alt_text || product.name}
                fill
                sizes="(max-width: 1024px) 100vw, 50vw"
                className="object-cover"
                priority
              />
            ) : (
              <div className="flex h-full items-center justify-center bg-primary-light/30">
                <PhotoIcon className="h-16 w-16 text-primary/40" />
              </div>
            )}
          </div>

          {/* Thumbnails */}
          {galleryItems.length > 1 && (
            <div className="mt-4 flex gap-3 overflow-x-auto pb-2">
              {galleryItems.map((item, index) => {
                const isActive = selectedIndex === index;
                const baseCls = `relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-lg border-2 transition-colors ${
                  isActive ? 'border-primary' : 'border-transparent hover:border-border'
                }`;

                if (item.kind === 'video') {
                  return (
                    <button
                      key="video"
                      type="button"
                      onClick={() => setSelectedIndex(index)}
                      className={baseCls}
                      aria-label="Play product video"
                    >
                      {item.poster ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={item.poster}
                          alt="Product video thumbnail"
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <div className="h-full w-full bg-gray-200" />
                      )}
                      <span className="absolute inset-0 flex items-center justify-center bg-black/30">
                        <PlayIcon className="h-6 w-6 text-white" />
                      </span>
                    </button>
                  );
                }

                const img = item.image;
                return (
                  <button
                    key={`img-${index}`}
                    type="button"
                    onClick={() => setSelectedIndex(index)}
                    className={baseCls}
                  >
                    <Image
                      src={img.url}
                      alt={img.alt_text || `${product.name} ${index + 1}`}
                      fill
                      sizes="80px"
                      className="object-cover"
                    />
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Product Info */}
        <div>
          <h1 className="text-2xl font-bold text-foreground sm:text-3xl">
            {product.name}
          </h1>

          {product.sku && (
            <p className="mt-1 text-sm text-muted-foreground">SKU: {product.sku}</p>
          )}

          {/* Price */}
          <div className="mt-4 flex items-baseline gap-3">
            <span className="text-3xl font-bold text-foreground">
              {formatPrice(product.selling_price)}
            </span>
            {hasDiscount && (
              <>
                <span className="text-lg text-muted-foreground line-through">
                  {formatPrice(product.mrp)}
                </span>
                <span className="rounded-md bg-red-100 px-2 py-0.5 text-sm font-semibold text-red-600">
                  -{discountPercent}%
                </span>
              </>
            )}
          </div>

          {/* Stock status */}
          <div className="mt-4">
            {product.in_stock ? (
              <span className="inline-flex items-center gap-1.5 text-sm font-medium text-green-600">
                <span className="h-2 w-2 rounded-full bg-green-500" />
                In Stock
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 text-sm font-medium text-red-500">
                <span className="h-2 w-2 rounded-full bg-red-500" />
                Out of Stock
              </span>
            )}
          </div>

          {/* Description */}
          {product.description && (
            <div className="mt-6">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground">
                Description
              </h2>
              <p className="mt-2 whitespace-pre-line leading-relaxed text-muted-foreground">
                {product.description}
              </p>
            </div>
          )}

          {/* Attributes */}
          {product.attributes && Object.keys(product.attributes).length > 0 && (
            <div className="mt-6">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground">
                Details
              </h2>
              <dl className="mt-3 space-y-2">
                {Object.entries(product.attributes).map(([key, value]) => (
                  <div key={key} className="flex gap-4 text-sm">
                    <dt className="w-32 flex-shrink-0 font-medium capitalize text-foreground">
                      {key.replace(/_/g, ' ')}
                    </dt>
                    <dd className="text-muted-foreground">{value}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}

          {/* Quantity + Add to Cart */}
          <div className="mt-8 border-t border-border pt-6">
            {cartQty > 0 ? (
              <div className="flex items-center gap-4">
                <QuantityStepper
                  value={cartQty}
                  onIncrement={handleCartIncrement}
                  onDecrement={handleCartDecrement}
                  disabled={loading}
                  variant="primary"
                  size="lg"
                />
                <span className="text-sm text-muted-foreground">in your cart</span>
              </div>
            ) : (
              <div className="flex items-center gap-4">
                <QuantityStepper
                  value={quantity}
                  onIncrement={() => setQuantity((q) => q + 1)}
                  onDecrement={() => setQuantity((q) => Math.max(1, q - 1))}
                  disableDecrement={quantity <= 1}
                />

                <Button
                  variant="primary"
                  size="lg"
                  className="flex-1"
                  onClick={handleAddToCart}
                  loading={loading}
                  disabled={!product.in_stock}
                >
                  {product.in_stock ? 'Add to Cart' : 'Out of Stock'}
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </Container>
  );
}
