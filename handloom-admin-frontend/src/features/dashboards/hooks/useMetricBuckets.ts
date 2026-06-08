import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

import type { BucketRow } from '@/shared/api/neonDataApi';
import { fetchMultiMetricBuckets } from '@/shared/api/neonDataApi';
import { usePreviousRange, useResolvedRange } from '@/shared/stores/dashboardFilters';

import { splitByMetric, totalCount } from '../lib/aggregate';

export interface MetricBuckets {
  from: Date;
  to: Date;
  prevFrom: Date;
  prevTo: Date;
  data: BucketRow[];
  prevData: BucketRow[];
  byMetric: Map<string, BucketRow[]>;
  prevByMetric: Map<string, BucketRow[]>;
  counts: Record<string, number>;
  prevCounts: Record<string, number>;
  isLoading: boolean;
  isError: boolean;
  refreshing: boolean;
  refetch: () => void;
}

/**
 * Fetches current + previous-window bucket rows for `metrics`, split per metric
 * with per-metric totals — the shared preamble for the Funnel and Products
 * dashboards. `key` namespaces the React Query cache (e.g. 'funnel').
 *
 * Pass a stable (module-level) `metrics` array so the derived memos don't
 * recompute every render.
 */
export function useMetricBuckets(key: string, metrics: readonly string[]): MetricBuckets {
  const { from, to } = useResolvedRange();
  const { prevFrom, prevTo } = usePreviousRange(from, to);

  const query = useQuery({
    queryKey: ['dashboards', key, from.toISOString(), to.toISOString()],
    queryFn: () => fetchMultiMetricBuckets({ metrics: [...metrics], from, to }),
  });

  const prevQuery = useQuery({
    queryKey: ['dashboards', `${key}:prev`, prevFrom.toISOString(), prevTo.toISOString()],
    queryFn: () => fetchMultiMetricBuckets({ metrics: [...metrics], from: prevFrom, to: prevTo }),
  });

  // Memoize raw rows so dependent memos stay stable when query.data is undefined.
  const data: BucketRow[] = useMemo(() => query.data ?? [], [query.data]);
  const prevData: BucketRow[] = useMemo(() => prevQuery.data ?? [], [prevQuery.data]);

  const byMetric = useMemo(() => splitByMetric(data, metrics), [data, metrics]);
  const prevByMetric = useMemo(() => splitByMetric(prevData, metrics), [prevData, metrics]);

  const { counts, prevCounts } = useMemo(() => {
    const c: Record<string, number> = {};
    const p: Record<string, number> = {};
    for (const metric of metrics) {
      c[metric] = totalCount(byMetric.get(metric) ?? []);
      p[metric] = totalCount(prevByMetric.get(metric) ?? []);
    }
    return { counts: c, prevCounts: p };
  }, [byMetric, prevByMetric, metrics]);

  return {
    from,
    to,
    prevFrom,
    prevTo,
    data,
    prevData,
    byMetric,
    prevByMetric,
    counts,
    prevCounts,
    // Gate the page on the current-window query only; previous-window deltas
    // degrade gracefully to "no prior data" without blocking render.
    isLoading: query.isLoading,
    isError: query.isError,
    refreshing: query.isFetching || prevQuery.isFetching,
    refetch: () => {
      void query.refetch();
      void prevQuery.refetch();
    },
  };
}
