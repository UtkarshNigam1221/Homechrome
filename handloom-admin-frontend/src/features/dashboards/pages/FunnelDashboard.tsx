import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  FunnelChart,
  Funnel,
  LabelList,
  Pie,
  PieChart,
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
  InlineBarCell,
  KPICard,
  PanelState,
  SectionTitle,
  StatTile,
} from '../components/primitives';
import { groupByLabel, totalCount } from '../lib/aggregate';

// caveman funnel page. 5 KPIs + 4 ratios + timeseries + device + UTM table

// Funnel-top is site_visitor (every page load, anonymous-friendly).
// session_started fires on OTP verify so it's downstream of visit and
// inflates conversion if used as the denominator.
const FUNNEL_METRICS = [
  'site_visitor',
  'product_viewed',
  'cart_added',
  'checkout_initiated',
  'payment_completed',
] as const;

// distinct colors per funnel stage
const STAGE_COLOR: Record<string, string> = {
  site_visitor: '#6366f1', // indigo
  product_viewed: '#8b5cf6', // violet
  cart_added: '#10b981', // emerald
  checkout_initiated: '#f59e0b', // amber
  payment_completed: '#06b6d4', // cyan
};

const STAGE_LABEL: Record<string, string> = {
  site_visitor: 'Visitors',
  product_viewed: 'Product Views',
  cart_added: 'Cart Added',
  checkout_initiated: 'Checkout Started',
  payment_completed: 'Payment Completed',
};

// targets for funnel rate tiles (green if at or above)
const RATE_TARGETS: Record<string, number> = {
  'Visitor -> Cart': 5,
  'Cart -> Checkout': 50,
  'Checkout -> Pay Initiated': 70,
  'Pay Completion': 80,
};

function pct(num: number, denom: number): number {
  if (denom <= 0) return 0;
  return Math.round((num / denom) * 1000) / 10;
}

function rateTone(value: number, target: number): 'good' | 'warn' {
  return value >= target ? 'good' : 'warn';
}

export function FunnelDashboard() {
  const { from, to } = useResolvedRange();

  // session_started + customer_first_purchase fetched alongside the linear
  // funnel for the Authentication + Attribution sections — both are parallel
  // signals (OTP-verified users, channel-attributed new customers).
  const FETCH_METRICS = [...FUNNEL_METRICS, 'session_started', 'customer_first_purchase'];

  // Previous-period window of equal length, shifted backwards. Used to
  // compute ↑↓X% deltas on each KPI card. Memoised on the bucket
  // boundaries so we don't re-fetch on every render.
  const prevFromMs = from.getTime() * 2 - to.getTime();
  const prevToMs = from.getTime();
  const prevFrom = useMemo(() => new Date(prevFromMs), [prevFromMs]);
  const prevTo = useMemo(() => new Date(prevToMs), [prevToMs]);

  const query = useQuery({
    queryKey: ['dashboards', 'funnel', from.toISOString(), to.toISOString()],
    queryFn: () =>
      fetchMultiMetricBuckets({
        metrics: FETCH_METRICS,
        from,
        to,
      }),
  });

  const prevQuery = useQuery({
    queryKey: ['dashboards', 'funnel:prev', prevFrom.toISOString(), prevTo.toISOString()],
    queryFn: () =>
      fetchMultiMetricBuckets({
        metrics: FETCH_METRICS,
        from: prevFrom,
        to: prevTo,
      }),
  });

  // memoize raw rows so dependent useMemos stay stable when query.data is undefined
  const data: BucketRow[] = useMemo(() => query.data ?? [], [query.data]);
  const prevData: BucketRow[] = useMemo(() => prevQuery.data ?? [], [prevQuery.data]);

  // split rows per metric once. cheap, called per render but data small
  const byMetric = useMemo(() => {
    const m = new Map<string, BucketRow[]>();
    for (const metric of FETCH_METRICS) m.set(metric, []);
    for (const row of data) {
      const arr = m.get(row.metric);
      if (arr) arr.push(row);
    }
    return m;
  }, [data]);

  const prevByMetric = useMemo(() => {
    const m = new Map<string, BucketRow[]>();
    for (const metric of FETCH_METRICS) m.set(metric, []);
    for (const row of prevData) {
      const arr = m.get(row.metric);
      if (arr) arr.push(row);
    }
    return m;
  }, [prevData]);

  const counts: Record<string, number> = {};
  const prevCounts: Record<string, number> = {};
  for (const metric of FETCH_METRICS) {
    counts[metric] = totalCount(byMetric.get(metric) ?? []);
    prevCounts[metric] = totalCount(prevByMetric.get(metric) ?? []);
  }

  // Authentication split — session_started carries is_new_user="true|false".
  // Sum once so the tiles + the new-customer ratio stay consistent.
  const authNewCount = useMemo(() => {
    return (byMetric.get('session_started') ?? [])
      .filter((r) => r.labels?.is_new_user === 'true')
      .reduce((acc, r) => acc + r.count, 0);
  }, [byMetric]);
  const authReturningCount = useMemo(() => {
    return (byMetric.get('session_started') ?? [])
      .filter((r) => r.labels?.is_new_user === 'false')
      .reduce((acc, r) => acc + r.count, 0);
  }, [byMetric]);

  // Funnel-snapshot for the static funnel chart: each stage rendered as
  // a trapezoidal slice that shrinks proportional to the count, so drop-
  // off is visible at a glance instead of having to read five overlaid
  // lines on a time chart.
  const funnelData = useMemo(
    () =>
      FUNNEL_METRICS.map((m) => ({
        name: STAGE_LABEL[m],
        value: counts[m] ?? 0,
        fill: STAGE_COLOR[m],
      })),
    [counts]
  );

  // device split for site_visitor (top of funnel — every page load)
  const visitorDeviceSeries = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('site_visitor') ?? [], 'device_type');
    return Array.from(grouped.entries())
      .map(([device, rows]) => ({
        device,
        count: totalCount(rows),
      }))
      .sort((a, b) => b.count - a.count);
  }, [byMetric]);

  // device split for payment_completed (funnel-bottom conversion)
  const deviceSeries = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('payment_completed') ?? [], 'device_type');
    return Array.from(grouped.entries())
      .map(([device, rows]) => ({
        device,
        count: totalCount(rows),
      }))
      .sort((a, b) => b.count - a.count);
  }, [byMetric]);

  // Attribution — utm_source breakdown for the two conversion events.
  // Builds (source, count) rows for each, drops the "unknown" bucket
  // since direct/no-utm traffic would otherwise dominate.
  const paymentsBySource = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('payment_completed') ?? [], 'utm_source');
    return Array.from(grouped.entries())
      .filter(([source]) => source !== 'unknown')
      .map(([source, rows]) => ({ source, count: totalCount(rows) }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 10);
  }, [byMetric]);

  const newCustomersBySource = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('customer_first_purchase') ?? [], 'utm_source');
    return Array.from(grouped.entries())
      .filter(([source]) => source !== 'unknown')
      .map(([source, rows]) => ({ source, count: totalCount(rows) }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 10);
  }, [byMetric]);

  // UTM source breakdown for site_visitor — top-of-funnel attribution.
  // Filters out the synthetic "unknown" row so direct/no-utm traffic
  // doesn't dominate the table.
  const utmRows = useMemo(() => {
    const visitors = byMetric.get('site_visitor') ?? [];
    const keyed = new Map<
      string,
      { source: string; medium: string; campaign: string; count: number }
    >();
    for (const r of visitors) {
      const source = r.labels?.utm_source ?? 'unknown';
      if (source === 'unknown') continue;
      const medium = r.labels?.utm_medium ?? 'unknown';
      const campaign = r.labels?.utm_campaign ?? 'unknown';
      const key = `${source}|${medium}|${campaign}`;
      const existing = keyed.get(key);
      if (existing) {
        existing.count += r.count;
      } else {
        keyed.set(key, { source, medium, campaign, count: r.count });
      }
    }
    return Array.from(keyed.values())
      .sort((a, b) => b.count - a.count)
      .slice(0, 20);
  }, [byMetric]);

  // funnel ratios — Visitor->Cart now uses site_visitor as the denominator
  // so anonymous browsers count too (session_started used to skip them).
  const rateVisitorCart = pct(counts.cart_added, counts.site_visitor);
  const rateCartCheckout = pct(counts.checkout_initiated, counts.cart_added);
  const rateCheckoutPayInit = pct(counts.payment_completed, counts.checkout_initiated);
  const ratePayCompletion = pct(counts.payment_completed, counts.checkout_initiated);

  const isLoading = query.isLoading;
  const isError = query.isError;

  // Hero TL;DR — one-line answer to "is this period good or bad?".
  const heroVisitors = counts.site_visitor ?? 0;
  const heroOrders = counts.payment_completed ?? 0;
  const heroConversion = rateVisitorCart > 0 ? ratePayCompletion : 0; // shown only when there's traffic
  const heroPrevOrders = prevCounts.payment_completed ?? 0;
  const heroOrderDelta =
    heroPrevOrders > 0
      ? ((heroOrders - heroPrevOrders) / heroPrevOrders) * 100
      : null;

  return (
    <div className="space-y-6">
      {/* Hero — TL;DR for the selected window */}
      <div className="rounded-lg border border-neutral-200 bg-gradient-to-r from-indigo-50 to-emerald-50 p-4">
        <h1 className="text-xl font-semibold text-neutral-900">Funnel</h1>
        <div className="mt-2 flex flex-wrap items-baseline gap-x-6 gap-y-1 text-sm">
          <span className="text-neutral-700">
            <span className="text-2xl font-bold text-indigo-700">
              {heroVisitors.toLocaleString('en-IN')}
            </span>{' '}
            visitors
          </span>
          <span className="text-neutral-700">
            <span className="text-2xl font-bold text-cyan-700">
              {heroOrders.toLocaleString('en-IN')}
            </span>{' '}
            orders
            {heroOrderDelta !== null ? (
              <span
                className={
                  'ml-2 text-xs ' +
                  (heroOrderDelta >= 0 ? 'text-emerald-600' : 'text-rose-600')
                }
                title="orders vs previous period of equal length"
              >
                {heroOrderDelta >= 0 ? '↑' : '↓'} {Math.abs(heroOrderDelta).toFixed(0)}%
              </span>
            ) : null}
          </span>
          <span className="text-neutral-700">
            <span className="text-2xl font-bold text-emerald-700">{heroConversion}%</span>{' '}
            conversion
          </span>
        </div>
      </div>

      {/* Row 1 - KPI cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {FUNNEL_METRICS.map((m) => (
          <KPICard
            key={m}
            title={STAGE_LABEL[m]}
            value={counts[m] ?? 0}
            previousValue={prevCounts[m]}
            data={byMetric.get(m) ?? []}
            color={STAGE_COLOR[m]}
          />
        ))}
      </div>

      {/* Row 2 - Funnel rates */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile
          label="Visitor -> Cart"
          value={`${rateVisitorCart}%`}
          tone={rateTone(rateVisitorCart, RATE_TARGETS['Visitor -> Cart'])}
          hint="cart_added / site_visitor"
        />
        <StatTile
          label="Cart -> Checkout"
          value={`${rateCartCheckout}%`}
          tone={rateTone(rateCartCheckout, RATE_TARGETS['Cart -> Checkout'])}
          hint="checkout_initiated / cart_added"
        />
        <StatTile
          label="Checkout -> Pay Initiated"
          value={`${rateCheckoutPayInit}%`}
          tone={rateTone(rateCheckoutPayInit, RATE_TARGETS['Checkout -> Pay Initiated'])}
          hint="payment_completed / checkout_initiated"
        />
        <StatTile
          label="Pay Completion"
          value={`${ratePayCompletion}%`}
          tone={rateTone(ratePayCompletion, RATE_TARGETS['Pay Completion'])}
          hint="payment_completed / checkout_initiated"
        />
      </div>

      {/* Row 3 - Funnel timeseries */}
      <Card>
        <SectionTitle subtitle="Drop-off across stages for the selected window">
          Visit-to-payment funnel
        </SectionTitle>
        <PanelState
          isLoading={isLoading}
          isError={isError}
          hasData={funnelData.some((d) => d.value > 0)}
        >
          <div className="h-80 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <FunnelChart>
                <Tooltip
                  contentStyle={{ fontSize: 12 }}
                  formatter={(value, _name, props) => {
                    const num = typeof value === 'number' ? value : Number(value ?? 0);
                    const stage = (props.payload as { name?: string } | undefined)?.name ?? '';
                    return [num.toLocaleString('en-IN'), stage];
                  }}
                />
                <Funnel dataKey="value" data={funnelData} isAnimationActive={false}>
                  <LabelList
                    position="right"
                    fill="#171717"
                    stroke="none"
                    fontSize={12}
                    dataKey="name"
                  />
                  <LabelList
                    position="center"
                    fill="#ffffff"
                    stroke="none"
                    fontSize={12}
                    fontWeight={600}
                    formatter={(v) =>
                      typeof v === 'number' ? v.toLocaleString('en-IN') : String(v ?? '')
                    }
                  />
                </Funnel>
              </FunnelChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* Row 4a — Authentication (logged-in users) */}
      <div>
        <h2 className="text-base font-semibold text-neutral-900">Authentication</h2>
        <p className="text-sm text-neutral-600">
          session_started fires on OTP verify — separate from the visit funnel
          since it captures the customer commit signal, not the visit signal.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <KPICard
          title="Sign-ins"
          value={counts.session_started ?? 0}
          data={byMetric.get('session_started') ?? []}
          color="#0ea5e9"
        />
        <KPICard
          title="New Customers"
          value={authNewCount}
          data={(byMetric.get('session_started') ?? []).filter(
            (r) => r.labels?.is_new_user === 'true',
          )}
          color="#22c55e"
        />
        <KPICard
          title="Returning Sign-ins"
          value={authReturningCount}
          data={(byMetric.get('session_started') ?? []).filter(
            (r) => r.labels?.is_new_user === 'false',
          )}
          color="#9333ea"
        />
      </div>

      {/* Row 4b — Device splits side-by-side as donuts */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <DeviceDonutCard
          title="Visitors by device"
          subtitle="site_visitor grouped by device_type"
          data={visitorDeviceSeries}
          isLoading={isLoading}
          isError={isError}
        />
        <DeviceDonutCard
          title="Payments by device"
          subtitle="payment_completed grouped by device_type"
          data={deviceSeries}
          isLoading={isLoading}
          isError={isError}
        />
      </div>

      {/* Row 5 — Attribution: conversion by source */}
      <div>
        <h2 className="text-base font-semibold text-neutral-900">Attribution</h2>
        <p className="text-sm text-neutral-600">
          Which marketing source drives orders + new customers. Direct traffic
          (no utm) is filtered out so paid channels stand on their own.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <SectionTitle subtitle="payment_completed grouped by utm_source, top 10">
            Payments by source
          </SectionTitle>
          <PanelState
            isLoading={isLoading}
            isError={isError}
            hasData={paymentsBySource.length > 0}
          >
            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={paymentsBySource} layout="vertical" margin={{ left: 60 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis type="number" fontSize={11} stroke="#737373" />
                  <YAxis
                    type="category"
                    dataKey="source"
                    fontSize={11}
                    stroke="#737373"
                    width={100}
                  />
                  <Tooltip contentStyle={{ fontSize: 12 }} />
                  <Bar dataKey="count" fill="#06b6d4" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>

        <Card>
          <SectionTitle subtitle="customer_first_purchase grouped by utm_source, top 10">
            New customers by source
          </SectionTitle>
          <PanelState
            isLoading={isLoading}
            isError={isError}
            hasData={newCustomersBySource.length > 0}
          >
            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={newCustomersBySource} layout="vertical" margin={{ left: 60 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis type="number" fontSize={11} stroke="#737373" />
                  <YAxis
                    type="category"
                    dataKey="source"
                    fontSize={11}
                    stroke="#737373"
                    width={100}
                  />
                  <Tooltip contentStyle={{ fontSize: 12 }} />
                  <Bar dataKey="count" fill="#22c55e" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>
      </div>

      {/* Row 6 - UTM source detail table (full tuple) */}
      <Card>
        <SectionTitle subtitle="Top 20 UTM-tagged visitor sources">Acquisition by UTM</SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={utmRows.length > 0}>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="border-b border-neutral-200 text-left text-xs uppercase tracking-wide text-neutral-500">
                <tr>
                  <th className="py-2 pr-4">utm_source</th>
                  <th className="py-2 pr-4">utm_medium</th>
                  <th className="py-2 pr-4">utm_campaign</th>
                  <th className="py-2 pr-4 text-right">visitors</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {utmRows.map((r) => {
                  const max = utmRows[0]?.count ?? 0;
                  return (
                    <tr key={`${r.source}-${r.medium}-${r.campaign}`}>
                      <td className="py-2 pr-4 font-medium text-neutral-900">{r.source}</td>
                      <td className="py-2 pr-4 text-neutral-700">{r.medium || '-'}</td>
                      <td className="py-2 pr-4 text-neutral-700">{r.campaign || '-'}</td>
                      <InlineBarCell value={r.count} max={max} color="#6366f1" />
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </PanelState>
      </Card>
    </div>
  );
}

const DEVICE_COLOR: Record<string, string> = {
  mobile: '#6366f1', // indigo
  desktop: '#06b6d4', // cyan
  tablet: '#f59e0b', // amber
  unknown: '#a3a3a3', // neutral
};

interface DeviceDonutCardProps {
  title: string;
  subtitle: string;
  data: { device: string; count: number }[];
  isLoading: boolean;
  isError: boolean;
}

/**
 * Donut chart for ≤4-category device splits. Inline % labels make the
 * proportions readable at a glance — far less work than reading bars at
 * different heights for the same low-cardinality data.
 */
function DeviceDonutCard({ title, subtitle, data, isLoading, isError }: DeviceDonutCardProps) {
  const total = data.reduce((acc, d) => acc + d.count, 0);
  return (
    <Card>
      <SectionTitle subtitle={subtitle}>{title}</SectionTitle>
      <PanelState isLoading={isLoading} isError={isError} hasData={total > 0}>
        <div className="h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                dataKey="count"
                nameKey="device"
                innerRadius={50}
                outerRadius={85}
                paddingAngle={2}
                isAnimationActive={false}
                label={(entry: { device?: string; percent?: number }) => {
                  const pct = ((entry.percent ?? 0) * 100).toFixed(0);
                  return `${entry.device ?? ''} ${pct}%`;
                }}
                labelLine={false}
              >
                {data.map((d) => (
                  <Cell
                    key={d.device}
                    fill={DEVICE_COLOR[d.device] ?? DEVICE_COLOR.unknown}
                  />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{ fontSize: 12 }}
                formatter={(value) => {
                  if (typeof value === 'number') return value.toLocaleString('en-IN');
                  return String(value ?? '');
                }}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </PanelState>
    </Card>
  );
}
