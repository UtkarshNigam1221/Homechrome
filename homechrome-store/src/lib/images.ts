// Single source of truth for image variant URL construction.
// Original asset URL: https://cdn/.../<stem>.<ext>
// Variant URL:        https://cdn/.../<stem>-<width>.<format>

export const IMAGE_WIDTHS = [320, 640, 1080, 1920] as const;
export type ImageWidth = (typeof IMAGE_WIDTHS)[number];
export type VariantFormat = 'webp' | 'avif' | 'jpg' | 'png';

function stripExtension(baseUrl: string): string {
  const dot = baseUrl.lastIndexOf('.');
  if (dot < 0) throw new Error(`Asset URL missing extension: ${baseUrl}`);
  return baseUrl.slice(0, dot);
}

export function variantUrl(
  baseUrl: string,
  width: ImageWidth,
  format: VariantFormat,
): string {
  return `${stripExtension(baseUrl)}-${width}.${format}`;
}

export function srcSet(baseUrl: string, format: VariantFormat): string {
  const stem = stripExtension(baseUrl);
  return IMAGE_WIDTHS.map((w) => `${stem}-${w}.${format} ${w}w`).join(', ');
}

export function fallbackFormat(baseUrl: string): 'jpg' | 'png' {
  return baseUrl.toLowerCase().endsWith('.png') ? 'png' : 'jpg';
}
