'use client';

import { CSSProperties } from 'react';

import { fallbackFormat, srcSet, variantUrl } from '@/lib/images';

interface AssetImageProps {
  /** Base asset URL with extension (e.g. https://cdn/.../<uuid>.jpg). */
  src: string;
  alt: string;
  /** CSS sizes attribute. Required so the browser can pick a variant. */
  sizes: string;
  /** Intrinsic width in pixels for layout-stability. */
  width: number;
  /** Intrinsic height in pixels for layout-stability. */
  height: number;
  loading?: 'lazy' | 'eager';
  fetchPriority?: 'high' | 'low' | 'auto';
  className?: string;
  style?: CSSProperties;
}

// Inline-element <picture> would not honour Mantine AspectRatio's
// "absolute fill its single child" contract; force block + full-size so
// when AspectRatio applies position:absolute; inset:0 to its child, the
// picture properly fills it and the inner <img> can inherit width/height.
const pictureStyle: CSSProperties = {
  display: 'block',
  width: '100%',
  height: '100%',
};

export function AssetImage({
  src,
  alt,
  sizes,
  width,
  height,
  loading = 'lazy',
  fetchPriority,
  className,
  style,
}: AssetImageProps) {
  const fmt = fallbackFormat(src);
  return (
    <picture style={pictureStyle}>
      <source type="image/avif" srcSet={srcSet(src, 'avif')} sizes={sizes} />
      <source type="image/webp" srcSet={srcSet(src, 'webp')} sizes={sizes} />
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={variantUrl(src, 640, fmt)}
        srcSet={srcSet(src, fmt)}
        sizes={sizes}
        width={width}
        height={height}
        loading={loading}
        fetchPriority={fetchPriority}
        alt={alt}
        className={className}
        style={style}
      />
    </picture>
  );
}
