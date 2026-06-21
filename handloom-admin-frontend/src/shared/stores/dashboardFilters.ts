import { useEffect, useMemo, useState } from 'react';
import { create } from 'zustand';

// Time-range presets the dashboards offer. `custom` opens a date picker.
type Range = '1h' | '6h' | '24h' | '7d' | '30d' | 'custom';

interface DashboardFiltersState {
  range: Range;
  customFrom?: Date;
  customTo?: Date;
  setRange: (r: Range) => void;
  setCustom: (from: Date, to: Date) => void;
}

export const useDashboardFilters = create<DashboardFiltersState>((set) => ({
  range: '24h',
  setRange: (range) => set({ range }),
  setCustom: (customFrom, customTo) => set({ range: 'custom', customFrom, customTo }),
}));

function rangeToMs(range: Range): number {
  switch (range) {
    case '1h':
      return 60 * 60 * 1000;
    case '6h':
      return 6 * 60 * 60 * 1000;
    case '24h':
      return 24 * 60 * 60 * 1000;
    case '7d':
      return 7 * 24 * 60 * 60 * 1000;
    case '30d':
      return 30 * 24 * 60 * 60 * 1000;
    default:
      return 24 * 60 * 60 * 1000;
  }
}

// minuteBucket returns the epoch-ms of the start of the current minute.
function currentMinuteBucket(): number {
  return Math.floor(Date.now() / 60_000) * 60_000;
}

// useResolvedRange returns memoized {from, to} for the current range.
// `from` and `to` align to a bucket at the START of the current minute, so
// calling this hook every render does NOT produce a new Date object each tick
// (which would cause infinite useQuery refetches). The bucket lives in state
// and is advanced by a timer — reading Date.now() during render is impure.
export function useResolvedRange(): { from: Date; to: Date } {
  const range = useDashboardFilters((s) => s.range);
  const customFromMs = useDashboardFilters((s) => s.customFrom?.getTime());
  const customToMs = useDashboardFilters((s) => s.customTo?.getTime());

  // Seeded once (lazy init), then advanced every minute by the timer below.
  const [minuteBucket, setMinuteBucket] = useState(currentMinuteBucket);
  useEffect(() => {
    const id = setInterval(() => setMinuteBucket(currentMinuteBucket()), 60_000);
    return () => clearInterval(id);
  }, []);

  return useMemo(() => {
    if (range === 'custom' && customFromMs && customToMs) {
      return { from: new Date(customFromMs), to: new Date(customToMs) };
    }
    const to = new Date(minuteBucket);
    const from = new Date(minuteBucket - rangeToMs(range));
    return { from, to };
  }, [range, customFromMs, customToMs, minuteBucket]);
}

// usePreviousRange returns the equal-length window immediately preceding
// [from, to], used to compute period-over-period (↑↓X%) deltas on KPI cards.
// Memoized on the epoch boundaries so it stays referentially stable across
// renders and doesn't trigger refetch loops.
export function usePreviousRange(from: Date, to: Date): { prevFrom: Date; prevTo: Date } {
  const prevFromMs = from.getTime() * 2 - to.getTime();
  const prevToMs = from.getTime();
  return useMemo(
    () => ({ prevFrom: new Date(prevFromMs), prevTo: new Date(prevToMs) }),
    [prevFromMs, prevToMs]
  );
}
