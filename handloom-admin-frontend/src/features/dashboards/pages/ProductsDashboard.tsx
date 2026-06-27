import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import {
  Card,
  HeroBanner,
  HeroStat,
  InlineBarCell,
  KPICard,
  PanelState,
  SectionTitle,
  StatTile,
  stickyTableHeadClass,
  tableHeadClass,
} from '../components/primitives';
import { DEVICE_COLORS } from '../constants';
import { useMetricBuckets } from '../hooks/useMetricBuckets';
import {
  formatINR,
  groupByLabel,
  paiseToINR,
  rankByLabel,
  totalCount,
  totalSum,
} from '../lib/aggregate';

// Catalog engagement, conversion, search, coupons, filter usage. Five
// UX-quality fixes mirrored from Funnel: hero TL;DR, ↑↓ deltas on KPIs,
// donut for low-cardinality splits, inline bars in tables, per-tile
// refresh buttons.

const PRODUCT_METRICS = [
  'product_viewed',
  'product_purchased',
  'search_query',
  'coupon_applied',
  'coupon_redeemed',
  'catalog_filter_applied',
  'item_added_to_cart',
  'cart_added',
  'cart_item_removed',
  'checkout_initiated',
] as const;

// pie slice colors for coupon outcomes
const OUTCOME_COLORS: Record<string, string> = {
  valid: '#10b981',
  invalid: '#f43f5e',
  expired: '#f59e0b',
  limit_reached: '#a855f7',
  unknown: '#9ca3af',
};

function pieColor(outcome: string): string {
  return OUTCOME_COLORS[outcome] ?? '#6366f1';
}

export function ProductsDashboard() {
  const {
    byMetric,
    counts,
    prevCounts,
    isLoading,
    isError,
    refreshing,
    refetch: refresh,
  } = useMetricBuckets('products', PRODUCT_METRICS);

  // 1. Top 10 viewed products
  const topViewed = useMemo(
    () => rankByLabel(byMetric.get('product_viewed') ?? [], 'product_id', { limit: 10 }),
    [byMetric]
  );

  // 2. Top 10 purchased + revenue
  const topPurchased = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('product_purchased') ?? [], 'product_id');
    return Array.from(grouped.entries())
      .map(([pid, rows]) => ({
        product_id: pid,
        units: totalCount(rows),
        revenue: paiseToINR(totalSum(rows)),
      }))
      .sort((a, b) => b.revenue - a.revenue)
      .slice(0, 10);
  }, [byMetric]);

  // 3. Conversion per top viewed product — horizontal bars + inline %
  const conversion = useMemo(() => {
    const views = groupByLabel(byMetric.get('product_viewed') ?? [], 'product_id');
    const purchases = groupByLabel(byMetric.get('product_purchased') ?? [], 'product_id');
    return Array.from(views.entries())
      .map(([pid, rows]) => {
        const v = totalCount(rows);
        const p = totalCount(purchases.get(pid) ?? []);
        return {
          product_id: pid,
          views: v,
          purchases: p,
          conversion: v > 0 ? Math.round((p / v) * 1000) / 10 : 0,
        };
      })
      .sort((a, b) => b.views - a.views)
      .slice(0, 10);
  }, [byMetric]);

  // 4. Top categories viewed (NEW)
  const topCategories = useMemo(
    () =>
      rankByLabel(byMetric.get('product_viewed') ?? [], 'category_id', {
        excludeUnknown: true,
        limit: 10,
      }),
    [byMetric]
  );

  // 5. Product views by device (NEW)
  const viewsByDevice = useMemo(
    () => rankByLabel(byMetric.get('product_viewed') ?? [], 'device_type'),
    [byMetric]
  );

  // 6. Search behaviour — total volume + zero-result rate (no time chart,
  // it was noisy at low traffic. Replaced with the intent breakdown below).
  const searchRows = useMemo(() => byMetric.get('search_query') ?? [], [byMetric]);
  const totalSearches = totalCount(searchRows);
  const zeroResultCount = searchRows
    .filter((r) => (r.labels?.has_results ?? 'false') === 'false')
    .reduce((acc, r) => acc + r.count, 0);
  const zeroResultRate =
    totalSearches > 0 ? Math.round((zeroResultCount / totalSearches) * 1000) / 10 : 0;

  // Top search intents — combined "<sorted-intents>_<category>" labels,
  // top 20 by count. Tells you what visitors are actually shopping for
  // without persisting any raw query strings (cardinality stays bounded).
  const searchIntents = useMemo(
    () => rankByLabel(searchRows, 'intent', { excludeUnknown: true, limit: 20 }),
    [searchRows]
  );

  // 7. Coupon performance
  const couponOutcomePie = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('coupon_applied') ?? [], 'outcome');
    return Array.from(grouped.entries()).map(([outcome, rows]) => ({
      name: outcome,
      value: totalCount(rows),
    }));
  }, [byMetric]);

  const couponRedeemedTable = useMemo(() => {
    const grouped = groupByLabel(byMetric.get('coupon_redeemed') ?? [], 'coupon_code');
    return Array.from(grouped.entries())
      .map(([code, rows]) => ({
        code,
        uses: totalCount(rows),
        discount: paiseToINR(totalSum(rows)),
      }))
      .sort((a, b) => b.discount - a.discount)
      .slice(0, 20);
  }, [byMetric]);

  // 8. Filter usage — backend only emits filter_key (truncated 32 chars).
  // filter_value isn't emitted yet so we just show filter keys.
  const filterUsage = useMemo(
    () => rankByLabel(byMetric.get('catalog_filter_applied') ?? [], 'filter_key', { limit: 20 }),
    [byMetric]
  );

  // 9. Cart funnel stats
  const itemsAdded = counts.item_added_to_cart ?? 0;
  const itemsRemoved = counts.cart_item_removed ?? 0;
  const cartsCreated = counts.cart_added ?? 0;
  const checkoutsStarted = counts.checkout_initiated ?? 0;
  const abandonment =
    cartsCreated > 0
      ? Math.max(0, Math.round((1 - checkoutsStarted / cartsCreated) * 1000) / 10)
      : 0;

  // Hero TL;DR — three killer numbers at the top.
  const heroViews = counts.product_viewed ?? 0;
  const heroPurchases = counts.product_purchased ?? 0;
  const heroPrevPurchases = prevCounts.product_purchased ?? 0;
  const heroPurchaseDelta =
    heroPrevPurchases > 0 ? ((heroPurchases - heroPrevPurchases) / heroPrevPurchases) * 100 : null;
  const topSeller = topPurchased[0];

  return (
    <div className="space-y-6">
      {/* Hero TL;DR */}
      <HeroBanner title="Products" gradient="from-violet-50 to-emerald-50">
        <HeroStat value={heroViews} label="product views" color="text-violet-700" />
        <HeroStat
          value={heroPurchases}
          label="units purchased"
          color="text-emerald-700"
          delta={heroPurchaseDelta}
          deltaTitle="units sold vs previous period of equal length"
        />
        {topSeller ? (
          <span className="text-neutral-700">
            top seller:{' '}
            <span className="font-semibold text-neutral-900">{topSeller.product_id}</span>{' '}
            <span className="text-xs text-neutral-500">({formatINR(topSeller.revenue)})</span>
          </span>
        ) : null}
      </HeroBanner>

      {/* KPI cards with ↑↓ deltas */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Product Views"
          value={heroViews}
          previousValue={prevCounts.product_viewed}
          data={byMetric.get('product_viewed') ?? []}
          color="#8b5cf6"
        />
        <KPICard
          title="Units Purchased"
          value={heroPurchases}
          previousValue={prevCounts.product_purchased}
          data={byMetric.get('product_purchased') ?? []}
          color="#10b981"
        />
        <KPICard
          title="Items Added to Cart"
          value={itemsAdded}
          previousValue={prevCounts.item_added_to_cart}
          data={byMetric.get('item_added_to_cart') ?? []}
          color="#6366f1"
        />
        <KPICard
          title="Searches"
          value={totalSearches}
          previousValue={prevCounts.search_query}
          data={searchRows}
          color="#06b6d4"
        />
      </div>

      {/* Cart funnel stat tiles */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatTile
          label="Cart abandonment"
          value={`${abandonment}%`}
          tone={abandonment > 70 ? 'bad' : abandonment > 50 ? 'warn' : 'good'}
          hint="1 - checkout_initiated / cart_added"
        />
        <StatTile
          label="Items removed from cart"
          value={itemsRemoved.toLocaleString('en-IN')}
          tone="neutral"
        />
        <StatTile
          label="Zero-result search rate"
          value={`${zeroResultRate}%`}
          tone={zeroResultRate > 25 ? 'bad' : zeroResultRate > 10 ? 'warn' : 'good'}
          hint="has_results=false / total searches"
        />
      </div>

      {/* Top viewed + Top categories side-by-side */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <SectionTitle
            subtitle="product_viewed grouped by product_id, top 10"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Top viewed products
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={topViewed.length > 0}>
            <div className="h-80 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={topViewed} layout="vertical" margin={{ left: 60 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis type="number" fontSize={11} stroke="#737373" />
                  <YAxis type="category" dataKey="key" fontSize={11} stroke="#737373" width={120} />
                  <Tooltip contentStyle={{ fontSize: 12 }} />
                  <Bar dataKey="count" fill="#8b5cf6" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>

        <Card>
          <SectionTitle
            subtitle="product_viewed grouped by category_id, top 10"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Top categories viewed
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={topCategories.length > 0}>
            <div className="h-80 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={topCategories} layout="vertical" margin={{ left: 60 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis type="number" fontSize={11} stroke="#737373" />
                  <YAxis type="category" dataKey="key" fontSize={11} stroke="#737373" width={120} />
                  <Tooltip contentStyle={{ fontSize: 12 }} />
                  <Bar dataKey="count" fill="#06b6d4" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>
      </div>

      {/* Top purchased (table with inline bars) + Views by device (donut) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <SectionTitle
            subtitle="product_purchased, top 10 by revenue (paise / 100)"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Top purchased products
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={topPurchased.length > 0}>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className={tableHeadClass}>
                  <tr>
                    <th className="py-2 pr-4">product_id</th>
                    <th className="py-2 pr-4 text-right">units</th>
                    <th className="py-2 pr-4 text-right">revenue</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-100">
                  {topPurchased.map((r) => {
                    const maxRevenue = topPurchased[0]?.revenue ?? 0;
                    return (
                      <tr key={r.product_id}>
                        <td className="py-2 pr-4 font-medium text-neutral-900">{r.product_id}</td>
                        <td className="py-2 pr-4 text-right tabular-nums">
                          {r.units.toLocaleString('en-IN')}
                        </td>
                        <InlineBarCell value={r.revenue} max={maxRevenue} color="#10b981">
                          <span className="text-emerald-700">{formatINR(r.revenue)}</span>
                        </InlineBarCell>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </PanelState>
        </Card>

        <Card>
          <SectionTitle
            subtitle="product_viewed grouped by device_type"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Views by device
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={viewsByDevice.length > 0}>
            <div className="h-72 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={viewsByDevice}
                    dataKey="count"
                    nameKey="key"
                    innerRadius={55}
                    outerRadius={95}
                    paddingAngle={2}
                    isAnimationActive={false}
                    label={(entry) => {
                      const e = entry as { name?: string | number; percent?: number };
                      const pct = ((e.percent ?? 0) * 100).toFixed(0);
                      return `${e.name ?? ''} ${pct}%`;
                    }}
                    labelLine={false}
                  >
                    {viewsByDevice.map((d) => (
                      <Cell key={d.key} fill={DEVICE_COLORS[d.key] ?? DEVICE_COLORS.unknown} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ fontSize: 12 }}
                    formatter={(value) =>
                      typeof value === 'number'
                        ? value.toLocaleString('en-IN')
                        : String(value ?? '')
                    }
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>
      </div>

      {/* Conversion — horizontal bars so product names are readable */}
      <Card>
        <SectionTitle
          subtitle="purchases / views × 100, top 10 by view count"
          onRefresh={refresh}
          isRefreshing={refreshing}
        >
          Conversion rate per product
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={conversion.length > 0}>
          <div className="h-80 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={conversion} layout="vertical" margin={{ left: 60 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" unit="%" fontSize={11} stroke="#737373" />
                <YAxis
                  type="category"
                  dataKey="product_id"
                  fontSize={11}
                  stroke="#737373"
                  width={120}
                />
                <Tooltip
                  contentStyle={{ fontSize: 12 }}
                  formatter={(value, _name, props) => {
                    const num = typeof value === 'number' ? value : Number(value ?? 0);
                    const payload = props.payload as
                      | { views?: number; purchases?: number }
                      | undefined;
                    return [
                      `${num}% (${payload?.purchases ?? 0}/${payload?.views ?? 0})`,
                      'conversion',
                    ];
                  }}
                />
                <Bar dataKey="conversion" fill="#10b981" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* Top search intents — what people are actually shopping for */}
      <Card>
        <SectionTitle
          subtitle='zero-shot classified by embedder: "<intents>_<category>", top 20'
          onRefresh={refresh}
          isRefreshing={refreshing}
        >
          Top search intents
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={searchIntents.length > 0}>
          <div className="h-96 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={searchIntents} layout="vertical" margin={{ left: 160 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" fontSize={11} stroke="#737373" />
                <YAxis type="category" dataKey="key" fontSize={11} stroke="#737373" width={220} />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Bar dataKey="count" fill="#06b6d4" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      {/* Coupon performance */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <SectionTitle
            subtitle="coupon_applied outcomes"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Coupon outcomes
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={couponOutcomePie.length > 0}>
            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={couponOutcomePie}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={90}
                    isAnimationActive={false}
                    label={(entry: { name?: string; percent?: number }) => {
                      const pct = ((entry.percent ?? 0) * 100).toFixed(0);
                      return `${entry.name ?? ''} ${pct}%`;
                    }}
                    labelLine={false}
                  >
                    {couponOutcomePie.map((entry) => (
                      <Cell key={entry.name} fill={pieColor(entry.name)} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </PanelState>
        </Card>

        <Card>
          <SectionTitle
            subtitle="coupon_redeemed, top 20 by discount given"
            onRefresh={refresh}
            isRefreshing={refreshing}
          >
            Coupon redemptions
          </SectionTitle>
          <PanelState
            isLoading={isLoading}
            isError={isError}
            hasData={couponRedeemedTable.length > 0}
          >
            <div className="max-h-64 overflow-y-auto">
              <table className="min-w-full text-sm">
                <thead className={stickyTableHeadClass}>
                  <tr>
                    <th className="py-2 pr-4">code</th>
                    <th className="py-2 pr-4 text-right">uses</th>
                    <th className="py-2 pr-4 text-right">discount</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-100">
                  {couponRedeemedTable.map((r) => {
                    const maxDiscount = couponRedeemedTable[0]?.discount ?? 0;
                    return (
                      <tr key={r.code}>
                        <td className="py-2 pr-4 font-medium text-neutral-900">{r.code}</td>
                        <td className="py-2 pr-4 text-right tabular-nums">
                          {r.uses.toLocaleString('en-IN')}
                        </td>
                        <InlineBarCell value={r.discount} max={maxDiscount} color="#f59e0b">
                          <span className="text-amber-700">{formatINR(r.discount)}</span>
                        </InlineBarCell>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </PanelState>
        </Card>
      </div>

      {/* Filter usage — relabelled (backend emits filter_key only) */}
      <Card>
        <SectionTitle
          subtitle="catalog_filter_applied grouped by filter_key, top 20"
          onRefresh={refresh}
          isRefreshing={refreshing}
        >
          Filter usage
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={filterUsage.length > 0}>
          <div className="h-96 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={filterUsage} layout="vertical" margin={{ left: 120 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" fontSize={11} stroke="#737373" />
                <YAxis type="category" dataKey="key" fontSize={11} stroke="#737373" width={180} />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Bar dataKey="count" fill="#8b5cf6" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>
    </div>
  );
}
