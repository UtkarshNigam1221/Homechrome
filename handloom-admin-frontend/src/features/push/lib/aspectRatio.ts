/** Android centre-crops the banner to roughly this shape. */
const TARGET_RATIO = 2;

/** Below this, the crop takes so much of the image that it misleads the sender. */
const TOLERANCE = 0.45;

export function bannerCropWarning(width: number, height: number): string | null {
  if (!width || !height) return null;

  const ratio = width / height;
  if (ratio >= TARGET_RATIO - TOLERANCE) return null;

  const kept = Math.round((width / TARGET_RATIO / height) * 100);
  return `This image will be cropped to fit a notification banner — about ${kept}% of its height will show. A wide 2:1 image crops better.`;
}
