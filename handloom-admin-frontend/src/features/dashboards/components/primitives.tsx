import { RefreshCw } from 'lucide-react';
import type { ReactNode } from 'react';
import { Area, AreaChart, ResponsiveContainer } from 'recharts';

import type { BucketRow } from '@/shared/api/neonDataApi';

import { aggregateByTime } from '../lib/aggregate';

// caveman shared cards. no logic, just chrome.

// Shared <thead> styling for dashboard data tables. stickyTableHeadClass pins
// the header for scrollable panels (adds the same classes Tailwind-wise).
export const tableHeadClass =
  'border-b border-neutral-200 text-left text-xs uppercase tracking-wide text-neutral-500';
export const stickyTableHeadClass = `sticky top-0 bg-white ${tableHeadClass}`;

interface CardProps {
  children: ReactNode;
  className?: string;
}

/**
 * Tailwind card wrapper used by every dashboard panel.
 */
export function Card({ children, className = '' }: CardProps) {
  return (
    <div className={`rounded-lg border border-neutral-200 bg-white p-4 shadow-sm ${className}`}>
      {children}
    </div>
  );
}

interface SectionTitleProps {
  children: ReactNode;
  subtitle?: string;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

export function SectionTitle({ children, subtitle, onRefresh, isRefreshing }: SectionTitleProps) {
  return (
    <div className="mb-3 flex items-start justify-between gap-3">
      <div>
        <h2 className="text-sm font-semibold text-neutral-900">{children}</h2>
        {subtitle ? <p className="text-xs text-neutral-500">{subtitle}</p> : null}
      </div>
      {onRefresh ? (
        <button
          type="button"
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-label="Refresh"
          title="Refresh"
          className="inline-flex items-center justify-center rounded-md p-1.5 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-50"
        >
          <RefreshCw size={14} className={isRefreshing ? 'animate-spin' : undefined} />
        </button>
      ) : null}
    </div>
  );
}

interface KPICardProps {
  title: string;
  value: number;
  data: BucketRow[];
  color?: string;
  format?: (n: number) => string;
  /** Previous-period value for ↑↓X% delta below the headline number. */
  previousValue?: number;
}

/**
 * KPI tile with title, big number, optional ↑↓ delta vs previous period,
 * and a 24-bucket sparkline.
 */
export function KPICard({
  title,
  value,
  data,
  color = '#6366f1',
  format,
  previousValue,
}: KPICardProps) {
  // 60-min bins for sparkline so a 24h window stays readable
  const series = aggregateByTime(data, 60, () => 'v');
  const display = format ? format(value) : value.toLocaleString('en-IN');
  // Slugify title for the SVG gradient id — spaces are invalid in XML
  // names, so a literal `spark-Product Views` makes the url(#...) lookup
  // fail and the area falls back to the SVG default fill (renders grey).
  const gradientId = `spark-${title.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;

  // Delta vs previous period — only shown when caller provided a previous
  // value. Hide on zero-previous to avoid divide-by-zero "Inf%" noise.
  let deltaPct: number | null = null;
  if (previousValue !== undefined && previousValue > 0) {
    deltaPct = ((value - previousValue) / previousValue) * 100;
  } else if (previousValue === 0 && value > 0) {
    deltaPct = null; // first traffic; meaningful delta requires a baseline
  }

  return (
    <Card className="flex flex-col gap-2">
      <div className="text-xs font-medium uppercase tracking-wide text-neutral-500">{title}</div>
      <div className="flex items-baseline gap-2">
        <div className="text-2xl font-semibold text-neutral-900">{display}</div>
        {deltaPct !== null ? (
          <div
            className={
              'text-xs font-medium ' + (deltaPct >= 0 ? 'text-emerald-600' : 'text-rose-600')
            }
            title="vs previous period of equal length"
          >
            {deltaPct >= 0 ? '↑' : '↓'} {Math.abs(deltaPct).toFixed(0)}%
          </div>
        ) : null}
      </div>
      <div className="h-12 w-full">
        {series.length > 1 ? (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={series}>
              <defs>
                <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Area
                type="monotone"
                dataKey="v"
                stroke={color}
                strokeWidth={1.5}
                fill={`url(#${gradientId})`}
                isAnimationActive={false}
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex h-full items-center text-xs text-neutral-400">no trend data</div>
        )}
      </div>
    </Card>
  );
}

interface InlineBarCellProps {
  value: number;
  max: number;
  color?: string;
  children?: ReactNode;
}

/**
 * Table cell with an inline bar drawn behind the content so readers can
 * compare ranked rows at a glance. width = (value/max) * 100%. Pass the
 * displayed number as `children` (formatted however caller wants).
 */
export function InlineBarCell({ value, max, color = '#6366f1', children }: InlineBarCellProps) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  return (
    <td className="relative py-2 pr-4 text-right tabular-nums">
      <div
        className="absolute inset-y-1 right-0 rounded-sm opacity-25"
        style={{ width: `${pct}%`, backgroundColor: color }}
      />
      <span className="relative">{children ?? value.toLocaleString('en-IN')}</span>
    </td>
  );
}

interface StatTileProps {
  label: string;
  value: string;
  tone?: 'good' | 'warn' | 'bad' | 'neutral';
  hint?: string;
}

/**
 * Single big-number tile with tone color (green/amber/red/neutral).
 */
export function StatTile({ label, value, tone = 'neutral', hint }: StatTileProps) {
  const toneClass =
    tone === 'good'
      ? 'text-emerald-600'
      : tone === 'warn'
        ? 'text-amber-600'
        : tone === 'bad'
          ? 'text-rose-600'
          : 'text-neutral-900';
  return (
    <Card className="flex flex-col gap-1">
      <div className="text-xs font-medium uppercase tracking-wide text-neutral-500">{label}</div>
      <div className={`text-2xl font-semibold ${toneClass}`}>{value}</div>
      {hint ? <div className="text-xs text-neutral-500">{hint}</div> : null}
    </Card>
  );
}

interface PanelStateProps {
  isLoading: boolean;
  isError: boolean;
  hasData: boolean;
  children: ReactNode;
}

/**
 * Common loading / error / empty wrapper for chart panels.
 */
export function PanelState({ isLoading, isError, hasData, children }: PanelStateProps) {
  if (isLoading) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-neutral-500">
        Loading...
      </div>
    );
  }
  if (isError) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-rose-600">
        Failed to load data
      </div>
    );
  }
  if (!hasData) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-neutral-400">
        No data in range
      </div>
    );
  }
  return <>{children}</>;
}
