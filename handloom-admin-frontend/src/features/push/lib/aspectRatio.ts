/** Android centre-crops the banner to roughly this shape. */
const TARGET_RATIO = 2;

/** How far from 2:1 an image may sit before the crop starts losing real content. */
const TOLERANCE = 0.45;

/** Past this, a panorama loses so much width that the subject can fall outside. */
const WIDE_LIMIT = 3.5;

export function bannerCropWarning(width: number, height: number): string | null {
  if (!width || !height) return null;

  const ratio = width / height;

  if (ratio < TARGET_RATIO - TOLERANCE) {
    const kept = Math.round((width / TARGET_RATIO / height) * 100);
    return `This image will be cropped to fit a notification banner — about ${kept}% of its height will show. A wide 2:1 image crops better.`;
  }

  if (ratio > WIDE_LIMIT) {
    const kept = Math.round(((height * TARGET_RATIO) / width) * 100);
    return `This image will be cropped to fit a notification banner — about ${kept}% of its width will show. A 2:1 image crops better.`;
  }

  return null;
}
