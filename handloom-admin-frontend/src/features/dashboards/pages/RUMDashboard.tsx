import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import type { BucketRow } from '@/shared/api/neonDataApi';
import { fetchMultiMetricBuckets } from '@/shared/api/neonDataApi';
import { useResolvedRange } from '@/shared/stores/dashboardFilters';

import {
  Card,
  PanelState,
  SectionTitle,
  StatTile,
  stickyTableHeadClass,
  tableHeadClass,
} from '../components/primitives';
import {
  aggregateByTime,
  groupByKey,
  groupByLabel,
  shortTimeLabel,
  splitByMetric,
  totalCount,
} from '../lib/aggregate';

// RUM + service-health dashboard — 8 panels.

const RUM_METRICS = [
  'rum_lcp',
  'rum_inp',
  'rum_cls',
  'rum_ttfb',
  'rum_js_error',
  'session_started',
  'payment_completed',
  'http_request',
  'http_client_call',
  'db_query',
  'lambda_cold_start',
  'lambda_invocation',
] as const;

type VitalTone = 'good' | 'warn' | 'bad';

function vitalTone(pctGood: number): VitalTone {
  if (pctGood >= 75) return 'good';
  if (pctGood >= 50) return 'warn';
  return 'bad';
}

/**
 * Compute percent of rows in `good` bucket for a vitals metric.
 */
function percentGood(rows: BucketRow[]): { value: number; total: number } {
  const total = totalCount(rows);
  if (total === 0) return { value: 0, total: 0 };
  const good = rows.filter((r) => r.labels?.bucket === 'good').reduce((acc, r) => acc + r.count, 0);
  return { value: Math.round((good / total) * 1000) / 10, total };
}

export function RUMDashboard() {
  const { from, to } = useResolvedRange();

  const query = useQuery({
    queryKey: ['dashboards', 'rum', from.toISOString(), to.toISOString()],
    queryFn: () =>
      fetchMultiMetricBuckets({
        metrics: [...RUM_METRICS],
        from,
        to,
      }),
  });

  // memoize raw rows so dependent useMemos stay stable when query.data is undefined
  const data: BucketRow[] = useMemo(() => query.data ?? [], [query.data]);

  const byMetric = useMemo(() => splitByMetric(data, RUM_METRICS), [data]);

  const lcp = percentGood(byMetric.get('rum_lcp') ?? []);
  const inp = percentGood(byMetric.get('rum_inp') ?? []);
  const cls = percentGood(byMetric.get('rum_cls') ?? []);
  const ttfb = percentGood(byMetric.get('rum_ttfb') ?? []);

  // 2. LCP distribution by page_type
  const lcpByPage = useMemo(() => {
    const rows = byMetric.get('rum_lcp') ?? [];
    const keyed = groupByKey(
      rows,
      (r) => `${r.labels?.page_type ?? 'unknown'}|${r.labels?.bucket ?? 'unknown'}`
    );
    // pivot to {page_type, good, needs_improvement, poor}
    const pages = new Map<
      string,
      { page_type: string; good: number; needs_improvement: number; poor: number }
    >();
    for (const [key, group] of keyed) {
      const [page, bucket] = key.split('|');
      let row = pages.get(page);
      if (!row) {
        row = { page_type: page, good: 0, needs_improvement: 0, poor: 0 };
        pages.set(page, row);
      }
      const n = totalCount(group);
      if (bucket === 'good') row.good += n;
      else if (bucket === 'needs_improvement') row.needs_improvement += n;
      else if (bucket === 'poor') row.poor += n;
    }
    return Array.from(pages.values()).sort(
      (a, b) => b.good + b.needs_improvement + b.poor - (a.good + a.needs_improvement + a.poor)
    );
  }, [byMetric]);

  // 3. JS errors table
  const jsErrors = useMemo(() => {
    const rows = byMetric.get('rum_js_error') ?? [];
    const keyed = groupByKey(
      rows,
      (r) => `${r.labels?.page_type ?? 'unknown'}|${r.labels?.error_type ?? 'unknown'}`
    );
    return Array.from(keyed.entries())
      .map(([key, group]) => {
        const [page, type] = key.split('|');
        return { page_type: page, error_type: type, count: totalCount(group) };
      })
      .sort((a, b) => b.count - a.count)
      .slice(0, 20);
  }, [byMetric]);

  // 4. Mobile vs desktop conversion (sessions vs payments)
  const deviceConversion = useMemo(() => {
    const sessions = groupByLabel(byMetric.get('session_started') ?? [], 'device_type');
    const payments = groupByLabel(byMetric.get('payment_completed') ?? [], 'device_type');
    const allDevices = new Set<string>([...sessions.keys(), ...payments.keys()]);
    return Array.from(allDevices).map((device) => {
      const s = totalCount(sessions.get(device) ?? []);
      const p = totalCount(payments.get(device) ?? []);
      return {
        device,
        sessions: s,
        payments: p,
        conversion: s > 0 ? Math.round((p / s) * 1000) / 10 : 0,
      };
    });
  }, [byMetric]);

  // 5. HTTP error rate by service: 4xx + 5xx counts per service
  const httpErrors = useMemo(() => {
    const rows = byMetric.get('http_request') ?? [];
    const grouped = groupByLabel(rows, 'service');
    return Array.from(grouped.entries())
      .map(([service, group]) => {
        let c4xx = 0;
        let c5xx = 0;
        for (const r of group) {
          const sc = r.labels?.status_class ?? '';
          if (sc === '4xx') c4xx += r.count;
          else if (sc === '5xx') c5xx += r.count;
        }
        return { service, '4xx': c4xx, '5xx': c5xx };
      })
      .filter((r) => r['4xx'] + r['5xx'] > 0)
      .sort((a, b) => b['5xx'] + b['4xx'] - (a['5xx'] + a['4xx']));
  }, [byMetric]);

  // 6. Cold-start rate by service
  const coldStartTable = useMemo(() => {
    const cold = groupByLabel(byMetric.get('lambda_cold_start') ?? [], 'service');
    const inv = groupByLabel(byMetric.get('lambda_invocation') ?? [], 'service');
    const all = new Set<string>([...cold.keys(), ...inv.keys()]);
    return Array.from(all)
      .map((service) => {
        const c = totalCount(cold.get(service) ?? []);
        const i = totalCount(inv.get(service) ?? []);
        return {
          service,
          cold_starts: c,
          invocations: i,
          rate: i > 0 ? Math.round((c / i) * 1000) / 10 : 0,
        };
      })
      .sort((a, b) => b.rate - a.rate);
  }, [byMetric]);

  // 7. Outbound by host
  const outboundByHost = useMemo(() => {
    const rows = byMetric.get('http_client_call') ?? [];
    const grouped = groupByLabel(rows, 'target_host');
    return Array.from(grouped.entries())
      .map(([host, group]) => {
        let ok = 0;
        let c4xx = 0;
        let c5xx = 0;
        for (const r of group) {
          const sc = r.labels?.status_class ?? '';
          if (sc === '4xx') c4xx += r.count;
          else if (sc === '5xx') c5xx += r.count;
          else ok += r.count;
        }
        return { host, ok, '4xx': c4xx, '5xx': c5xx };
      })
      .sort((a, b) => b.ok + b['4xx'] + b['5xx'] - (a.ok + a['4xx'] + a['5xx']))
      .slice(0, 15);
  }, [byMetric]);

  // 8. DB query rate timeseries by operation
  const dbSeries = useMemo(
    () =>
      aggregateByTime(byMetric.get('db_query') ?? [], 5, (r) => r.labels?.operation ?? 'unknown'),
    [byMetric]
  );
  const dbOps = useMemo(() => {
    const ops = new Set<string>();
    for (const r of byMetric.get('db_query') ?? []) {
      ops.add(r.labels?.operation ?? 'unknown');
    }
    return Array.from(ops);
  }, [byMetric]);

  const dbOpColors = ['#6366f1', '#10b981', '#f59e0b', '#f43f5e', '#06b6d4', '#a855f7'];

  const isLoading = query.isLoading;
  const isError = query.isError;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-neutral-900">RUM & Service Health</h1>
        <p className="text-sm text-neutral-600">
          Web vitals, JS errors, HTTP error rates, lambda cold starts, gateway calls, DB latency.
        </p>
      </div>

      {/* 1. Web vitals stat row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile
          label="LCP good"
          value={`${lcp.value}%`}
          tone={vitalTone(lcp.value)}
          hint={`${lcp.total.toLocaleString('en-IN')} samples`}
        />
        <StatTile
          label="INP good"
          value={`${inp.value}%`}
          tone={vitalTone(inp.value)}
          hint={`${inp.total.toLocaleString('en-IN')} samples`}
        />
        <StatTile
          label="CLS good"
          value={`${cls.value}%`}
          tone={vitalTone(cls.value)}
          hint={`${cls.total.toLocaleString('en-IN')} samples`}
        />
        <StatTile
          label="TTFB good"
          value={`${ttfb.value}%`}
          tone={vitalTone(ttfb.value)}
          hint={`${ttfb.total.toLocaleString('en-IN')} samples`}
        />
      </div>

      {/* 2. LCP distribution by page_type stacked */}
      <Card>
        <SectionTitle subtitle="rum_lcp grouped by page_type x bucket">
          LCP distribution by page
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={lcpByPage.length > 0}>
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={lcpByPage}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="page_type" fontSize={11} stroke="#737373" />
                <YAxis fontSize={11} stroke="#737373" />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                <Bar dataKey="good" stackId="lcp" fill="#10b981" />
                <Bar dataKey="needs_improvement" stackId="lcp" fill="#f59e0b" />
                <Bar dataKey="poor" stackId="lcp" fill="#f43f5e" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* 3. JS errors table */}
      <Card>
        <SectionTitle subtitle="rum_js_error, top 20 by count">JS errors by page</SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={jsErrors.length > 0}>
          <div className="max-h-80 overflow-y-auto">
            <table className="min-w-full text-sm">
              <thead className={stickyTableHeadClass}>
                <tr>
                  <th className="py-2 pr-4">page_type</th>
                  <th className="py-2 pr-4">error_type</th>
                  <th className="py-2 pr-4 text-right">count</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {jsErrors.map((r) => (
                  <tr key={`${r.page_type}-${r.error_type}`}>
                    <td className="py-2 pr-4 font-medium text-neutral-900">{r.page_type}</td>
                    <td className="py-2 pr-4 text-neutral-700">{r.error_type}</td>
                    <td className="py-2 pr-4 text-right tabular-nums text-rose-600">
                      {r.count.toLocaleString('en-IN')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </PanelState>
      </Card>

      {/* 4. Device conversion table */}
      <Card>
        <SectionTitle subtitle="session_started + payment_completed">
          Mobile vs desktop conversion
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={deviceConversion.length > 0}>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className={tableHeadClass}>
                <tr>
                  <th className="py-2 pr-4">device</th>
                  <th className="py-2 pr-4 text-right">sessions</th>
                  <th className="py-2 pr-4 text-right">payments</th>
                  <th className="py-2 pr-4 text-right">conv %</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {deviceConversion.map((r) => (
                  <tr key={r.device}>
                    <td className="py-2 pr-4 font-medium text-neutral-900">{r.device}</td>
                    <td className="py-2 pr-4 text-right tabular-nums">
                      {r.sessions.toLocaleString('en-IN')}
                    </td>
                    <td className="py-2 pr-4 text-right tabular-nums">
                      {r.payments.toLocaleString('en-IN')}
                    </td>
                    <td className="py-2 pr-4 text-right tabular-nums text-indigo-700">
                      {r.conversion}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </PanelState>
      </Card>

      {/* 5. HTTP error rate by service */}
      <Card>
        <SectionTitle subtitle="http_request 4xx + 5xx per service">
          HTTP errors by service
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={httpErrors.length > 0}>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={httpErrors}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="service" fontSize={11} stroke="#737373" />
                <YAxis fontSize={11} stroke="#737373" />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                <Bar dataKey="4xx" stackId="e" fill="#f59e0b" />
                <Bar dataKey="5xx" stackId="e" fill="#f43f5e" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* 6. Cold start table */}
      <Card>
        <SectionTitle subtitle="lambda_cold_start / lambda_invocation">
          Lambda cold-start rate
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={coldStartTable.length > 0}>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className={tableHeadClass}>
                <tr>
                  <th className="py-2 pr-4">service</th>
                  <th className="py-2 pr-4 text-right">cold starts</th>
                  <th className="py-2 pr-4 text-right">invocations</th>
                  <th className="py-2 pr-4 text-right">rate %</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {coldStartTable.map((r) => (
                  <tr key={r.service}>
                    <td className="py-2 pr-4 font-medium text-neutral-900">{r.service}</td>
                    <td className="py-2 pr-4 text-right tabular-nums">
                      {r.cold_starts.toLocaleString('en-IN')}
                    </td>
                    <td className="py-2 pr-4 text-right tabular-nums">
                      {r.invocations.toLocaleString('en-IN')}
                    </td>
                    <td
                      className={`py-2 pr-4 text-right tabular-nums ${
                        r.rate > 10
                          ? 'text-rose-600'
                          : r.rate > 5
                            ? 'text-amber-600'
                            : 'text-emerald-700'
                      }`}
                    >
                      {r.rate}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </PanelState>
      </Card>

      {/* 7. Outbound by host */}
      <Card>
        <SectionTitle subtitle="http_client_call grouped by target_host, top 15">
          Outbound gateway calls
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={outboundByHost.length > 0}>
          <div className="h-80 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={outboundByHost}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="host" fontSize={10} stroke="#737373" />
                <YAxis fontSize={11} stroke="#737373" />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                <Bar dataKey="ok" stackId="o" fill="#10b981" />
                <Bar dataKey="4xx" stackId="o" fill="#f59e0b" />
                <Bar dataKey="5xx" stackId="o" fill="#f43f5e" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* 8. DB query timeseries by operation */}
      <Card>
        <SectionTitle subtitle="db_query grouped by operation, 5-min bins">
          DB queries by operation
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={dbSeries.length > 0}>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={dbSeries}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis
                  dataKey="time"
                  tickFormatter={shortTimeLabel}
                  fontSize={11}
                  stroke="#737373"
                />
                <YAxis fontSize={11} stroke="#737373" />
                <Tooltip
                  labelFormatter={(v) => shortTimeLabel(String(v))}
                  contentStyle={{ fontSize: 12 }}
                />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                {dbOps.map((op, i) => (
                  <Line
                    key={op}
                    type="monotone"
                    dataKey={op}
                    stroke={dbOpColors[i % dbOpColors.length]}
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>
    </div>
  );
}
