import { describe, expect, it } from 'vitest';

import { bannerCropWarning } from '../aspectRatio';

describe('bannerCropWarning', () => {
  it('accepts a 2:1 banner', () => {
    expect(bannerCropWarning(1024, 512)).toBeNull();
  });

  it('accepts a mild deviation', () => {
    expect(bannerCropWarning(1000, 600)).toBeNull();
  });

  // A portrait product shot loses most of its height to the centre crop.
  it('warns on a portrait image', () => {
    expect(bannerCropWarning(800, 1200)).toMatch(/cropped/i);
  });

  it('warns on a square image', () => {
    expect(bannerCropWarning(900, 900)).toMatch(/cropped/i);
  });

  // A panorama loses its sides to the same centre crop.
  it('warns on a very wide image', () => {
    const warning = bannerCropWarning(3000, 300);
    expect(warning).toMatch(/cropped/i);
    expect(warning).toMatch(/width/i);
  });

  it('accepts a wide image still close to 2:1', () => {
    expect(bannerCropWarning(1024, 320)).toBeNull();
  });

  it('ignores an unmeasured image', () => {
    expect(bannerCropWarning(0, 0)).toBeNull();
  });
});
