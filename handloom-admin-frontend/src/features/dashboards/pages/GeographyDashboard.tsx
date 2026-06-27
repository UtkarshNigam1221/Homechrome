import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { CircleMarker, MapContainer, TileLayer, Tooltip as LeafletTooltip } from 'react-leaflet';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import type { BucketRow, CityCentroid } from '@/shared/api/neonDataApi';
import { fetchCityCentroids, fetchMultiMetricBuckets } from '@/shared/api/neonDataApi';
import { PageHeader } from '@/shared/components/ui';
import { useResolvedRange } from '@/shared/stores/dashboardFilters';

import { Card, PanelState, SectionTitle, stickyTableHeadClass } from '../components/primitives';
import {
  formatINR,
  groupByKey,
  groupByLabel,
  paiseToINR,
  rankByLabel,
  splitByMetric,
  totalCount,
  totalSum,
} from '../lib/aggregate';

// World geography page. Tile-based map via react-leaflet + OpenStreetMap.
// CircleMarker radius is in screen pixels regardless of zoom, so the dots
// stay readable from world view down to street level. India fits naturally
// as the default focus.

const GEO_METRICS = [
  'orders_placed',
  'orders_value',
  'customer_first_purchase',
  'site_visitor',
] as const;

// Default to India centroid; user can zoom out / pan globally.
const INDIA_CENTER: [number, number] = [22, 78];
const DEFAULT_ZOOM = 4;
const OSM_TILE_URL = 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
const OSM_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

interface CityDot {
  city: string;
  country: string;
  lng: number;
  lat: number;
  count: number;
  /** marker radius (log-scaled) */
  r: number;
}

// geoLabelKey builds the canonical `${city}|${country}` key used everywhere geo
// rows are grouped (map dots, leaderboards, centroid lookup). Normalising case
// here keeps every consumer consistent so a city never splits into two rows or
// fails to match its centroid because of casing drift.
function geoLabelKey(labels: Record<string, string> | undefined): string {
  return `${(labels?.city ?? 'unknown').toLowerCase()}|${(labels?.country ?? 'unknown').toUpperCase()}`;
}

function buildCityDots(rows: BucketRow[], centroids: Map<string, CityCentroid>): CityDot[] {
  // centroids keyed by `${city}|${country}` so we match unique pairs and
  // don't collide on same-named cities across countries (e.g. London UK vs CA).
  const grouped = groupByKey(rows, (row) => geoLabelKey(row.labels));

  const dots: CityDot[] = [];
  for (const [key, cityRows] of grouped) {
    const centroid = centroids.get(key);
    if (!centroid) continue;
    const count = totalCount(cityRows);
    if (count <= 0) continue;
    dots.push({
      city: centroid.city,
      country: centroid.country,
      lng: centroid.lng,
      lat: centroid.lat,
      count,
      // Leaflet CircleMarker radius is in screen pixels regardless of zoom,
      // so a single value works at every zoom level.
      r: Math.max(4, Math.min(24, Math.log10(count + 1) * 6)),
    });
  }
  return dots.sort((a, b) => b.count - a.count);
}

export function GeographyDashboard() {
  const { from, to } = useResolvedRange();

  const metricQuery = useQuery({
    queryKey: ['dashboards', 'geography', from.toISOString(), to.toISOString()],
    queryFn: () =>
      fetchMultiMetricBuckets({
        metrics: [...GEO_METRICS],
        from,
        to,
      }),
  });

  const centroidQuery = useQuery({
    queryKey: ['dashboards', 'geography', 'city-centroids'],
    queryFn: fetchCityCentroids,
    staleTime: 24 * 60 * 60 * 1000, // centroids rarely change
  });

  const data: BucketRow[] = useMemo(() => metricQuery.data ?? [], [metricQuery.data]);
  const centroids = useMemo(() => centroidQuery.data ?? [], [centroidQuery.data]);

  // Keyed lookup by `${city}|${country}` so duplicate city names across
  // countries each find their own centroid.
  const centroidMap = useMemo(() => {
    const m = new Map<string, CityCentroid>();
    for (const c of centroids) {
      // Defensive: cached row from a pre-cutover persister snapshot may not
      // have a country field. Skip those — they would key as `${city}|UNDEFINED`
      // and never match dot lookups anyway.
      if (!c?.city || !c?.country) continue;
      m.set(`${c.city.toLowerCase()}|${c.country.toUpperCase()}`, c);
    }
    return m;
  }, [centroids]);

  const byMetric = useMemo(() => splitByMetric(data, GEO_METRICS), [data]);

  const orderDots = useMemo(
    () => buildCityDots(byMetric.get('orders_placed') ?? [], centroidMap),
    [byMetric, centroidMap]
  );

  const newCustomerDots = useMemo(
    () => buildCityDots(byMetric.get('customer_first_purchase') ?? [], centroidMap),
    [byMetric, centroidMap]
  );

  const visitorDots = useMemo(
    () => buildCityDots(byMetric.get('site_visitor') ?? [], centroidMap),
    [byMetric, centroidMap]
  );

  // Top-20 leaderboard by visitors
  const topCitiesVisitors = useMemo(() => {
    const rows = byMetric.get('site_visitor') ?? [];
    const keyed = groupByKey(rows, (r) => geoLabelKey(r.labels));
    return Array.from(keyed.entries())
      .map(([key, group]) => {
        const [city, country] = key.split('|');
        return { city, country, count: totalCount(group) };
      })
      .sort((a, b) => b.count - a.count)
      .slice(0, 20);
  }, [byMetric]);

  // Top 20 cities by orders
  const topCities = useMemo(() => {
    const rows = byMetric.get('orders_placed') ?? [];
    const keyed = groupByKey(rows, (r) => geoLabelKey(r.labels));
    return Array.from(keyed.entries())
      .map(([key, group]) => {
        const [city, country] = key.split('|');
        return { city, country, count: totalCount(group) };
      })
      .sort((a, b) => b.count - a.count)
      .slice(0, 20);
  }, [byMetric]);

  // Top 20 cities by revenue
  const topCitiesRevenue = useMemo(() => {
    const rows = byMetric.get('orders_value') ?? [];
    const keyed = groupByLabel(rows, 'city');
    return Array.from(keyed.entries())
      .map(([city, group]) => ({
        city,
        orders: totalCount(group),
        revenue: paiseToINR(totalSum(group)),
      }))
      .sort((a, b) => b.revenue - a.revenue)
      .slice(0, 20);
  }, [byMetric]);

  // Orders by country (bar chart, top 15)
  const countryBars = useMemo(
    () => rankByLabel(byMetric.get('orders_placed') ?? [], 'country', { limit: 15 }),
    [byMetric]
  );

  const isLoading = metricQuery.isLoading || centroidQuery.isLoading;
  const isError = metricQuery.isError || centroidQuery.isError;

  // Refresh handlers — each tile owns a button. Map tiles refresh both
  // metric counts and centroids since both feed the geomap; leaderboard
  // / bar tiles only need the metric counts.
  const refreshMap = () => {
    void metricQuery.refetch();
    void centroidQuery.refetch();
  };
  const refreshMetrics = () => {
    void metricQuery.refetch();
  };
  const mapRefreshing = metricQuery.isFetching || centroidQuery.isFetching;
  const metricsRefreshing = metricQuery.isFetching;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Geography"
        subtitle="Worldwide orders, revenue, new customers, and visitors. Markers placed from CloudFront-resolved city centroids, auto-populated on first sighting."
      />

      <WorldDotsCard
        title="Visitors by city"
        subtitle="site_visitor counts (1 per RUM page_view, anonymous-friendly)"
        dots={visitorDots}
        fill="#8b5cf6"
        nameLabel="visitors"
        isLoading={isLoading}
        isError={isError}
        onRefresh={refreshMap}
        isRefreshing={mapRefreshing}
      />

      <Card>
        <SectionTitle
          subtitle="site_visitor grouped by city + country, top 20"
          onRefresh={refreshMetrics}
          isRefreshing={metricsRefreshing}
        >
          Top 20 cities by visitors
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={topCitiesVisitors.length > 0}>
          <div className="max-h-96 overflow-y-auto">
            <table className="min-w-full text-sm">
              <thead className={stickyTableHeadClass}>
                <tr>
                  <th className="py-2 pr-4">#</th>
                  <th className="py-2 pr-4">city</th>
                  <th className="py-2 pr-4">country</th>
                  <th className="py-2 pr-4 text-right">visitors</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {topCitiesVisitors.map((r, i) => (
                  <tr key={`${r.city}-${r.country}`}>
                    <td className="py-2 pr-4 text-neutral-500 tabular-nums">{i + 1}</td>
                    <td className="py-2 pr-4 font-medium text-neutral-900">{r.city}</td>
                    <td className="py-2 pr-4 text-neutral-700">{r.country}</td>
                    <td className="py-2 pr-4 text-right tabular-nums">
                      {r.count.toLocaleString('en-IN')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </PanelState>
      </Card>

      <WorldDotsCard
        title="Orders by city"
        subtitle="orders_placed grouped by city + country"
        dots={orderDots}
        fill="#6366f1"
        nameLabel="orders"
        isLoading={isLoading}
        isError={isError}
        onRefresh={refreshMap}
        isRefreshing={mapRefreshing}
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <SectionTitle
            subtitle="orders_placed grouped by city + country"
            onRefresh={refreshMetrics}
            isRefreshing={metricsRefreshing}
          >
            Top 20 cities by orders
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={topCities.length > 0}>
            <div className="max-h-96 overflow-y-auto">
              <table className="min-w-full text-sm">
                <thead className={stickyTableHeadClass}>
                  <tr>
                    <th className="py-2 pr-4">city</th>
                    <th className="py-2 pr-4">country</th>
                    <th className="py-2 pr-4 text-right">orders</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-100">
                  {topCities.map((r) => (
                    <tr key={`${r.city}-${r.country}`}>
                      <td className="py-2 pr-4 font-medium text-neutral-900">{r.city}</td>
                      <td className="py-2 pr-4 text-neutral-700">{r.country}</td>
                      <td className="py-2 pr-4 text-right tabular-nums">
                        {r.count.toLocaleString('en-IN')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </PanelState>
        </Card>

        <Card>
          <SectionTitle
            subtitle="orders_value sum / 100, INR"
            onRefresh={refreshMetrics}
            isRefreshing={metricsRefreshing}
          >
            Top 20 cities by revenue
          </SectionTitle>
          <PanelState isLoading={isLoading} isError={isError} hasData={topCitiesRevenue.length > 0}>
            <div className="max-h-96 overflow-y-auto">
              <table className="min-w-full text-sm">
                <thead className={stickyTableHeadClass}>
                  <tr>
                    <th className="py-2 pr-4">city</th>
                    <th className="py-2 pr-4 text-right">orders</th>
                    <th className="py-2 pr-4 text-right">revenue</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-100">
                  {topCitiesRevenue.map((r) => (
                    <tr key={r.city}>
                      <td className="py-2 pr-4 font-medium text-neutral-900">{r.city}</td>
                      <td className="py-2 pr-4 text-right tabular-nums">
                        {r.orders.toLocaleString('en-IN')}
                      </td>
                      <td className="py-2 pr-4 text-right tabular-nums text-emerald-700">
                        {formatINR(r.revenue)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </PanelState>
        </Card>
      </div>

      <Card>
        <SectionTitle
          subtitle="orders_placed grouped by country, top 15"
          onRefresh={refreshMetrics}
          isRefreshing={metricsRefreshing}
        >
          Orders by country
        </SectionTitle>
        <PanelState isLoading={isLoading} isError={isError} hasData={countryBars.length > 0}>
          <div className="h-96 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={countryBars} layout="vertical" margin={{ left: 60 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" fontSize={11} stroke="#737373" />
                <YAxis type="category" dataKey="key" fontSize={11} stroke="#737373" width={80} />
                <Tooltip contentStyle={{ fontSize: 12 }} />
                <Bar dataKey="count" fill="#f59e0b" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </PanelState>
      </Card>

      <WorldDotsCard
        title="New customers by city"
        subtitle="customer_first_purchase by city"
        dots={newCustomerDots}
        fill="#10b981"
        nameLabel="new customers"
        isLoading={isLoading}
        isError={isError}
        onRefresh={refreshMap}
        isRefreshing={mapRefreshing}
      />
    </div>
  );
}

interface WorldDotsCardProps {
  title: string;
  subtitle: string;
  dots: CityDot[];
  fill: string;
  nameLabel: string;
  isLoading: boolean;
  isError: boolean;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

function WorldDotsCard({
  title,
  subtitle,
  dots,
  fill,
  nameLabel,
  isLoading,
  isError,
  onRefresh,
  isRefreshing,
}: WorldDotsCardProps) {
  return (
    <Card>
      <SectionTitle subtitle={subtitle} onRefresh={onRefresh} isRefreshing={isRefreshing}>
        {title}
      </SectionTitle>
      <PanelState isLoading={isLoading} isError={isError} hasData={dots.length > 0}>
        <div className="h-[480px] w-full overflow-hidden rounded-md">
          <MapContainer
            center={INDIA_CENTER}
            zoom={DEFAULT_ZOOM}
            minZoom={2}
            maxZoom={18}
            scrollWheelZoom
            worldCopyJump
            style={{ height: '100%', width: '100%' }}
          >
            <TileLayer url={OSM_TILE_URL} attribution={OSM_ATTRIBUTION} />
            {dots.map((d) => (
              <CircleMarker
                key={`${d.city}-${d.country}`}
                center={[d.lat, d.lng]}
                radius={d.r}
                pathOptions={{
                  color: '#ffffff',
                  weight: 1,
                  fillColor: fill,
                  fillOpacity: 0.75,
                }}
              >
                <LeafletTooltip direction="top" offset={[0, -4]} opacity={0.95}>
                  <div className="text-xs">
                    <div className="font-medium">
                      {d.city} ({d.country})
                    </div>
                    <div>
                      {d.count.toLocaleString('en-IN')} {nameLabel}
                    </div>
                  </div>
                </LeafletTooltip>
              </CircleMarker>
            ))}
          </MapContainer>
        </div>
      </PanelState>
    </Card>
  );
}
