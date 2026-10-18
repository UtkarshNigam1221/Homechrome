'use client';

import { useState } from 'react';

const WIDTHS = [320, 640, 1080, 1920] as const;
const ORIGINAL_EXT_RE = /\.(jpe?g|png)$/i;

function buildSrcset(url: string, fmt: 'webp' | 'avif' | 'original'): string {
  const match = url.match(ORIGINAL_EXT_RE);
  if (!match) return '';
  const dot = url.lastIndexOf('.');
  const stem = url.slice(0, dot);
  const origExt = url.slice(dot + 1).toLowerCase();
  const ext = fmt === 'original' ? (origExt === 'png' ? 'png' : 'jpg') : fmt;
  return WIDTHS.map((w) => `${stem}-${w}.${ext} ${w}w`).join(', ');
}

interface SmartImageProps {
  src: string;
  alt: string;
  sizes: string;
  className?: string;
  style?: React.CSSProperties;
  loading?: 'lazy' | 'eager';
  fetchPriority?: 'high' | 'low' | 'auto';
}

export function SmartImage({
  src,
  alt,
  sizes,
  className,
  style,
  loading = 'lazy',
  fetchPriority,
}: SmartImageProps) {
  const [failed, setFailed] = useState(false);

  // If src isn't an image we can variant (e.g. webp source, external URL),
  // fall back to bare <img src>.
  if (!ORIGINAL_EXT_RE.test(src) || failed) {
    return (
      // Intentional bare <img>: src is either non-jpg/png (already-optimized webp/avif,
      // external URL) or a fallback after variant load failure. next/image adds no value here.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={alt}
        className={className}
        style={style}
        loading={loading}
        fetchPriority={fetchPriority}
        onError={() => setFailed(true)}
      />
    );
  }

  return (
    <picture>
      <source type="image/avif" srcSet={buildSrcset(src, 'avif')} sizes={sizes} />
      <source type="image/webp" srcSet={buildSrcset(src, 'webp')} sizes={sizes} />
      <img
        src={src}
        srcSet={buildSrcset(src, 'original')}
        sizes={sizes}
        alt={alt}
        className={className}
        style={style}
        loading={loading}
        fetchPriority={fetchPriority}
        onError={() => setFailed(true)}
      />
    </picture>
  );
}
