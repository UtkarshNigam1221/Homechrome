import type { BucketRow } from '@/shared/api/neonDataApi';

// caveman aggregate row -> chart-ready shape
// all helpers pure. no fetch, no state.

/**
 * Sum all `count` fields. Default reducer for "how many events in window".
 */
export function totalCount(rows: BucketRow[]): number {
  return rows.reduce((acc, r) => acc + (r.count ?? 0), 0);
}

/**
 * Sum all `sum_value` fields. Used for revenue/duration histograms.
 */
export function totalSum(rows: BucketRow[]): number {
  return rows.reduce((acc, r) => acc + (r.sum_value ?? 0), 0);
}

/**
 * Group rows by a single label key. Missing labels bucketed as 'unknown'.
 */
export function groupByLabel(rows: BucketRow[], labelKey: string): Map<string, BucketRow[]> {
  const out = new Map<string, BucketRow[]>();
  for (const r of rows) {
    const k = r.labels?.[labelKey] ?? 'unknown';
    const arr = out.get(k);
    if (arr) {
      arr.push(r);
    } else {
      out.set(k, [r]);
    }
  }
  return out;
}

/**
 * Bucket rows by their `metric` field, pre-seeding every entry in `metrics`
 * with an empty array so callers can rely on each key existing in the result.
 */
export function splitByMetric(
  rows: BucketRow[],
  metrics: readonly string[]
): Map<string, BucketRow[]> {
  const out = new Map<string, BucketRow[]>();
  for (const m of metrics) out.set(m, []);
  for (const r of rows) {
    out.get(r.metric)?.push(r);
  }
  return out;
}

/**
 * Group rows by a derived composite key.
 */
export function groupByKey<K extends string>(
  rows: BucketRow[],
  keyFn: (r: BucketRow) => K
): Map<K, BucketRow[]> {
  const out = new Map<K, BucketRow[]>();
  for (const r of rows) {
    const k = keyFn(r);
    const arr = out.get(k);
    if (arr) {
      arr.push(r);
    } else {
      out.set(k, [r]);
    }
  }
  return out;
}

/**
 * Floor an ISO timestamp down to `bucketMinutes` resolution.
 */
function floorTime(iso: string, bucketMinutes: number): string {
  const d = new Date(iso);
  const ms = bucketMinutes * 60 * 1000;
  return new Date(Math.floor(d.getTime() / ms) * ms).toISOString();
}

/**
 * Pivot rows into a time-indexed series suitable for Recharts.
 *
 * Output: array of {time, <seriesA>: n, <seriesB>: n, ...}, sorted ascending.
 *
 * - `bucketMinutes` controls the time bin width (e.g. 15 = quarter hour).
 * - `groupBy` maps a row to its series name. If you want everything in
 *   one series, return a constant.
 */
export function aggregateByTime(
  rows: BucketRow[],
  bucketMinutes: number,
  groupBy: (r: BucketRow) => string
): Array<{ time: string; [key: string]: number | string }> {
  const byTime = new Map<string, Record<string, number>>();
  const seriesNames = new Set<string>();
  for (const r of rows) {
    const t = floorTime(r.bucket_start, bucketMinutes);
    const series = groupBy(r);
    seriesNames.add(series);
    let slot = byTime.get(t);
    if (!slot) {
      slot = {};
      byTime.set(t, slot);
    }
    slot[series] = (slot[series] ?? 0) + (r.count ?? 0);
  }
  const sortedTimes = Array.from(byTime.keys()).sort();
  return sortedTimes.map((time) => {
    const slot = byTime.get(time) ?? {};
    const row: { time: string; [key: string]: number | string } = { time };
    // zero-fill series so Recharts lines stay continuous
    for (const s of seriesNames) {
      row[s] = slot[s] ?? 0;
    }
    return row;
  });
}

/**
 * Format an ISO time string into a short human-readable label for chart X axes.
 */
export function shortTimeLabel(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

/**
 * Paise -> INR (rupees) for display. Backend stores money as paise.
 */
export function paiseToINR(paise: number): number {
  return Math.round(paise) / 100;
}

/**
 * Format INR number with Indian locale grouping.
 */
export function formatINR(rupees: number): string {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: 0,
  }).format(rupees);
}
