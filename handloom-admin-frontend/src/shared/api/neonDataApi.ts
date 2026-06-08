import type { AxiosInstance } from 'axios';
import axios from 'axios';

import { getNeonAuthToken } from '@/shared/auth/neonAuth';

// Neon Data API: REST over Postgres, JWT-authed, role-scoped via RLS.
// Browser hits Neon directly — no backend hop. JWT is minted on demand
// from Neon Auth (Better Auth) — see src/shared/auth/neonAuth.ts.

const BASE_URL = import.meta.env.VITE_NEON_DATA_API_URL ?? '';

if (!BASE_URL) {
  console.warn('VITE_NEON_DATA_API_URL not set — dashboards will not load.');
}

const client: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  headers: {
    Accept: 'application/json',
    Prefer: 'count=exact',
  },
});

// Attach a fresh Neon Auth JWT to every request. If no session, request
// goes out unauthenticated — Data API will 401 and the NeonAuthGate
// component will redirect the user to sign in.
client.interceptors.request.use(async (config) => {
  const token = await getNeonAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

/**
 * Generic Data API row fetcher.
 *
 * Data API exposes tables via PostgREST-style URLs:
 *   GET /rest/v1/metric_counters?metric=eq.session_started&bucket_start=gte.2026-06-01
 *
 * Query operators we use:
 *   eq.   — equal
 *   gte.  — greater-than-or-equal
 *   lte.  — less-than-or-equal
 *   in.   — in list (comma-separated, parenthesised)
 *   order — column.asc / column.desc
 *   select — projection (e.g. metric,bucket_start,count)
 *   limit — row cap
 */
// Tables the dashboards are allowed to read. RLS + role grants on the Neon side
// are the real security boundary; this allowlist is defense-in-depth so a typo
// or future code path can't point the browser at an unintended table.
const ALLOWED_TABLES = new Set(['metric_counters', 'city_centroids']);

export async function fetchRows<T = Record<string, unknown>>(
  table: string,
  params: Record<string, string | number | undefined>
): Promise<T[]> {
  if (!ALLOWED_TABLES.has(table)) {
    throw new Error(`neonDataApi: table "${table}" is not in the dashboard allowlist`);
  }
  // strip empty/undefined params — Data API rejects empty strings as filters
  const clean: Record<string, string> = {};
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') clean[k] = String(v);
  }
  const res = await client.get<T[]>(`/rest/v1/${table}`, { params: clean });
  warnIfTruncated(table, res.headers['content-range'], res.data.length);
  return res.data;
}

// PostgREST returns a Content-Range header (e.g. "0-9999/52340") when
// Prefer: count=exact is set. If the total exceeds the rows we got back, the
// client-side aggregation is operating on a silently truncated set and the
// dashboard numbers are wrong — surface that loudly rather than hiding it.
function warnIfTruncated(table: string, contentRange: unknown, returned: number): void {
  if (typeof contentRange !== 'string') return;
  const total = Number(contentRange.split('/')[1]);
  if (Number.isFinite(total) && total > returned) {
    console.error(
      `neonDataApi: "${table}" returned ${returned} of ${total} rows — results are truncated and ` +
        `aggregated dashboard values will be INCORRECT. Narrow the time range or raise the row limit.`
    );
  }
}

// ─── High-level query helpers tailored to metric_counters ───

export interface BucketRow {
  metric: string;
  labels: Record<string, string>;
  bucket_start: string;
  count: number;
  sum_value: number;
  retention_class: 'business' | 'service';
}

/**
 * Fetch buckets for multiple metrics in one request (using in.).
 */
export function fetchMultiMetricBuckets(opts: {
  metrics: string[];
  from: Date;
  to: Date;
  limit?: number;
}): Promise<BucketRow[]> {
  return fetchRows<BucketRow>('metric_counters', {
    metric: `in.(${opts.metrics.join(',')})`,
    bucket_start: `gte.${opts.from.toISOString()}`,
    and: `(bucket_start.lte.${opts.to.toISOString()})`,
    select: 'metric,labels,bucket_start,count,sum_value',
    order: 'bucket_start.asc',
    limit: opts.limit ?? 10000,
  });
}

export interface CityCentroid {
  city: string;
  country: string;
  lat: number;
  lng: number;
}

export function fetchCityCentroids(): Promise<CityCentroid[]> {
  return fetchRows<CityCentroid>('city_centroids', {
    select: 'city,country,lat,lng',
    order: 'city.asc',
  });
}
