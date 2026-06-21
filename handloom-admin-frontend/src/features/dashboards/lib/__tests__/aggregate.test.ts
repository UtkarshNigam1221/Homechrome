import { describe, expect, it } from 'vitest';

import type { BucketRow } from '@/shared/api/neonDataApi';

import { rankByLabel } from '../aggregate';

const row = (device: string, count: number): BucketRow =>
  ({
    metric: 'm',
    bucket_start: '2026-01-01T00:00:00Z',
    count,
    labels: { device },
  }) as unknown as BucketRow;

describe('rankByLabel', () => {
  it('groups by label, sums counts, sorts desc', () => {
    const out = rankByLabel([row('mobile', 2), row('desktop', 5), row('mobile', 3)], 'device');
    expect(out).toEqual([
      { key: 'mobile', count: 5 },
      { key: 'desktop', count: 5 },
    ]);
  });

  it('buckets missing labels as unknown and can exclude them', () => {
    const rows = [
      row('mobile', 1),
      { metric: 'm', bucket_start: 'x', count: 9 } as unknown as BucketRow,
    ];
    expect(rankByLabel(rows, 'device')).toContainEqual({ key: 'unknown', count: 9 });
    expect(rankByLabel(rows, 'device', { excludeUnknown: true })).toEqual([
      { key: 'mobile', count: 1 },
    ]);
  });

  it('applies top-N limit after sorting', () => {
    const out = rankByLabel([row('a', 1), row('b', 2), row('c', 3)], 'device', { limit: 2 });
    expect(out).toEqual([
      { key: 'c', count: 3 },
      { key: 'b', count: 2 },
    ]);
  });
});
